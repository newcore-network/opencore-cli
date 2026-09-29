package builder

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/newcore-network/opencore-cli/internal/ui"
)

// typegenKind identifies which Register map an entry feeds.
type typegenKind string

const (
	kindServerNet typegenKind = "serverEvents"
	kindClientNet typegenKind = "clientEvents"
	kindServerRPC typegenKind = "serverRpc"
	kindClientRPC typegenKind = "clientRpc"
	kindCommand   typegenKind = "commands"
)

// typegenEntry is one decorated handler discovered during the scan.
type typegenEntry struct {
	Kind        typegenKind
	EventName   string // source text of the name, used for ordering and de-duplication
	ImportPath  string // relative to the .opencore directory, extension stripped
	ClassName   string
	MethodName  string
	Description string
	Usage       string
	// SourceFile and SourceLine locate the decorator, for diagnostics only.
	SourceFile string
	SourceLine int
}

type TypegenResult struct {
	Entries  []typegenEntry
	Warnings []SourceValidationIssue
	// Changed is false when the emitted content matched what was already on disk.
	Changed bool
}

type TypegenOptions struct {
	// Strict makes unknown event names a compile error in the consuming project.
	Strict bool
	// FrameworkPackage is the module specifier to augment. Defaults to "@open-core/framework".
	FrameworkPackage string
	// ResourceKey is the `Register` key this resource's maps are filed under. Set per resource by
	// generateTypes; see resourceRegisterKey.
	ResourceKey string
}

const defaultFrameworkPackage = "@open-core/framework"
const typegenFileName = "opencore.gen.ts"

// registerModuleSuffix is the subpath that *declares* the `Register` interface.
//
// The augmentation must target the declaring module, not the package root. `Register` is only
// re-exported by the root entry, and TypeScript's declaration merging does not follow
// re-exports: augmenting '@open-core/framework' would silently create a second, unrelated
// interface and every narrowing would quietly fall back to `string`.
const registerModuleSuffix = "/register"

func walkSourceFiles(resourcePath string, viewPath string, visit func(path string) error) error {
	skipDir := newViewDirSkipper(viewPath)

	return filepath.WalkDir(resourcePath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipDir(path, d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !isScannableSource(d.Name()) {
			return nil
		}
		return visit(path)
	})
}

func isScannableSource(name string) bool {
	return strings.HasSuffix(name, ".ts") && !strings.HasSuffix(name, ".d.ts")
}

func relativeSourcePath(root string, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

func moduleImportPath(baseDir string, path string) (string, error) {
	rel, err := filepath.Rel(baseDir, path)
	if err != nil {
		return "", err
	}
	spec := filepath.ToSlash(rel)
	if !strings.HasPrefix(spec, ".") {
		spec = "./" + spec
	}
	return strings.TrimSuffix(spec, ".ts"), nil
}

func typegenEntries(analysis *resourceAnalysis, resourcePath string, baseDir string) ([]typegenEntry, []SourceValidationIssue, error) {
	var entries []typegenEntry
	var warnings []SourceValidationIssue

	for _, source := range analysis.sources {
		relPath := relativeSourcePath(resourcePath, source.path)
		for _, warning := range source.result.Warnings {
			warnings = append(warnings, SourceValidationIssue{File: relPath, Line: warning.Line, Message: warning.Message})
		}
		if len(source.result.Handlers) == 0 {
			continue
		}

		importPath, err := moduleImportPath(baseDir, source.path)
		if err != nil {
			return nil, nil, err
		}
		for _, handler := range source.result.Handlers {
			entries = append(entries, typegenEntry{
				Kind:        handler.Kind,
				EventName:   handler.Event,
				ImportPath:  importPath,
				ClassName:   handler.ClassName,
				MethodName:  handler.MethodName,
				Description: handler.Description,
				Usage:       handler.Usage,
				SourceFile:  relPath,
				SourceLine:  handler.Line,
			})
		}
	}

	sortEntriesDeterministically(entries)
	warnings = append(warnings, duplicateEntryWarnings(entries)...)
	sortValidationIssues(warnings)

	return entries, warnings, nil
}

func sortValidationIssues(issues []SourceValidationIssue) {
	slices.SortStableFunc(issues, func(a, b SourceValidationIssue) int {
		return cmp.Or(
			strings.Compare(a.File, b.File),
			cmp.Compare(a.Line, b.Line),
			strings.Compare(a.Message, b.Message),
		)
	})
}

func sortEntriesDeterministically(entries []typegenEntry) {
	slices.SortStableFunc(entries, func(a, b typegenEntry) int {
		return cmp.Or(
			strings.Compare(string(a.Kind), string(b.Kind)),
			strings.Compare(a.EventName, b.EventName),
			strings.Compare(a.ImportPath, b.ImportPath),
			strings.Compare(a.ClassName, b.ClassName),
			strings.Compare(a.MethodName, b.MethodName),
		)
	})
}

func isSameHandler(a typegenEntry, b typegenEntry) bool {
	return a.ImportPath == b.ImportPath && a.ClassName == b.ClassName && a.MethodName == b.MethodName
}

func claimsSameName(a typegenEntry, b typegenEntry) bool {
	return a.Kind == b.Kind && a.EventName == b.EventName
}

func duplicateEntryWarnings(entries []typegenEntry) []SourceValidationIssue {
	var warnings []SourceValidationIssue

	for i := 1; i < len(entries); i++ {
		previous, current := entries[i-1], entries[i]
		if !claimsSameName(previous, current) || isSameHandler(previous, current) {
			continue
		}

		warnings = append(warnings, SourceValidationIssue{
			File: current.SourceFile,
			Line: current.SourceLine,
			Message: fmt.Sprintf(
				"%q is already handled by %s.%s (%s:%d); this handler is ignored by typegen",
				current.EventName, previous.ClassName, previous.MethodName,
				previous.SourceFile, previous.SourceLine,
			),
		})
	}

	return warnings
}

func quoteTSString(s string) string {
	replacer := strings.NewReplacer("\\", "\\\\", "'", "\\'", "\n", "\\n", "\r", "")
	return "'" + replacer.Replace(s) + "'"
}

func renderTypegenFile(entries []typegenEntry, views registeredViews, opts TypegenOptions) string {
	var b strings.Builder
	writeGeneratedHeader(&b, "Regenerated by `opencore dev` and `opencore build`.")

	if len(entries) == 0 && views.empty() {
		b.WriteString("export {};\n")
		return b.String()
	}

	registerModule := registerModuleOf(opts)
	aliases := newControllerAliases(entries)
	byKind := groupByKind(entries)

	writeTypegenPreamble(&b, byKind, views, registerModule)
	aliases.writeDeclarations(&b)
	writeViewControllerAliases(&b, views.controllerImports)

	var members []registerMember
	for _, section := range sectionsPresentIn(byKind) {
		writeSectionMap(&b, section, byKind[section.kind], aliases)
		members = append(members, registerMember{section.registerAs, section.iface})
	}
	members = append(members, writeRegisteredViewMaps(&b, views)...)
	writeRegisterAugmentation(&b, registerModule, opts, members)

	return b.String()
}

// registerMember is one map a generated file files under its resource's `Register` key.
type registerMember struct {
	name  string
	iface string
}

func writeRegisteredViewMaps(b *strings.Builder, views registeredViews) []registerMember {
	var members []registerMember
	if len(views.send) > 0 {
		writeViewMap(b, "GenViewSend",
			"/** Messages the client sends to a WebView (from its `send()` calls). */\n", views.send)
		members = append(members, registerMember{"viewSend", "GenViewSend"})
	}
	if len(views.receive) > 0 {
		writeViewMap(b, "GenViewReceive",
			"/** Messages a WebView sends to the client (from `@Client.OnView` handlers). */\n", views.receive)
		members = append(members, registerMember{"viewReceive", "GenViewReceive"})
	}
	return members
}

func registerModuleOf(opts TypegenOptions) string {
	pkg := opts.FrameworkPackage
	if pkg == "" {
		pkg = defaultFrameworkPackage
	}
	return pkg + registerModuleSuffix
}

func writeGeneratedHeader(b *strings.Builder, subtitle string) {
	b.WriteString("// Auto-generated by OpenCore CLI. Do not edit.\n")
	fmt.Fprintf(b, "// %s\n", subtitle)
	b.WriteString("/* eslint-disable */\n")
	b.WriteString("// biome-ignore-all lint: generated file\n\n")
}

func groupByKind(entries []typegenEntry) map[typegenKind][]typegenEntry {
	byKind := map[typegenKind][]typegenEntry{}
	for _, e := range entries {
		byKind[e.Kind] = append(byKind[e.Kind], e)
	}
	return byKind
}

func writeTypegenPreamble(
	b *strings.Builder,
	byKind map[typegenKind][]typegenEntry,
	views registeredViews,
	registerModule string,
) {
	if len(byKind[kindServerNet]) > 0 || len(byKind[kindServerRPC]) > 0 {
		fmt.Fprintf(b, "import type { DropFirst } from %s\n\n", quoteTSString(registerModule))
	}

	if slices.ContainsFunc(views.all(), usesPayloadHelper) {
		b.WriteString(payloadHelperDecl)
		b.WriteString("\n")
	}
}

type controllerRef struct {
	importPath string
	className  string
}

type controllerAliases struct {
	order []controllerRef
	byRef map[controllerRef]string
}

func newControllerAliases(entries []typegenEntry) *controllerAliases {
	aliases := &controllerAliases{byRef: map[controllerRef]string{}}

	for _, e := range entries {
		ref := controllerRef{e.ImportPath, e.ClassName}
		if _, seen := aliases.byRef[ref]; !seen {
			aliases.byRef[ref] = ""
			aliases.order = append(aliases.order, ref)
		}
	}
	slices.SortFunc(aliases.order, func(a, b controllerRef) int {
		return cmp.Or(
			strings.Compare(a.importPath, b.importPath),
			strings.Compare(a.className, b.className),
		)
	})
	for i, ref := range aliases.order {
		aliases.byRef[ref] = fmt.Sprintf("T%d_%s", i, ref.className)
	}

	return aliases
}

func (a *controllerAliases) of(e typegenEntry) string {
	return a.byRef[controllerRef{e.ImportPath, e.ClassName}]
}

func (a *controllerAliases) writeDeclarations(b *strings.Builder) {
	for _, ref := range a.order {
		fmt.Fprintf(b, "type %s = InstanceType<typeof import(%s).%s>\n",
			a.byRef[ref], quoteTSString(ref.importPath), ref.className)
	}
	if len(a.order) > 0 {
		b.WriteString("\n")
	}
}

type typegenSection struct {
	kind         typegenKind
	iface        string
	registerAs   string
	dropPlayer   bool
	isRPC        bool
	isCommandMap bool
}

var typegenSections = []typegenSection{
	{kind: kindServerNet, iface: "GenServerEvents", registerAs: "serverEvents", dropPlayer: true},
	{kind: kindClientNet, iface: "GenClientEvents", registerAs: "clientEvents"},
	{kind: kindServerRPC, iface: "GenServerRpc", registerAs: "serverRpc", dropPlayer: true, isRPC: true},
	{kind: kindClientRPC, iface: "GenClientRpc", registerAs: "clientRpc", isRPC: true},
	{kind: kindCommand, iface: "GenCommands", registerAs: "commands", isCommandMap: true},
}

func sectionsPresentIn(byKind map[typegenKind][]typegenEntry) []typegenSection {
	var present []typegenSection
	for _, section := range typegenSections {
		if len(byKind[section.kind]) > 0 {
			present = append(present, section)
		}
	}
	return present
}

func writeSectionMap(
	b *strings.Builder,
	section typegenSection,
	entries []typegenEntry,
	aliases *controllerAliases,
) {
	fmt.Fprintf(b, "export type %s = {\n", section.iface)
	for _, e := range firstPerEventName(entries) {
		fmt.Fprintf(b, "  %s: %s\n", quoteTSString(e.EventName), section.valueExpr(e, aliases))
	}
	b.WriteString("}\n\n")
}

func firstPerEventName(entries []typegenEntry) []typegenEntry {
	seen := map[string]bool{}
	var unique []typegenEntry
	for _, e := range entries {
		if seen[e.EventName] {
			continue
		}
		seen[e.EventName] = true
		unique = append(unique, e)
	}
	return unique
}

func (s typegenSection) valueExpr(e typegenEntry, aliases *controllerAliases) string {
	switch {
	case s.isCommandMap:
		return commandValueExpr(e)
	case s.isRPC:
		return fmt.Sprintf("{ args: %s; result: %s }",
			s.argsExpr(e, aliases), resultExpr(e, aliases))
	default:
		return s.argsExpr(e, aliases)
	}
}

func (s typegenSection) argsExpr(e typegenEntry, aliases *controllerAliases) string {
	params := fmt.Sprintf("Parameters<%s[%s]>", aliases.of(e), quoteTSString(e.MethodName))
	if s.dropPlayer {
		return fmt.Sprintf("DropFirst<%s>", params)
	}
	return params
}

func resultExpr(e typegenEntry, aliases *controllerAliases) string {
	return fmt.Sprintf("Awaited<ReturnType<%s[%s]>>", aliases.of(e), quoteTSString(e.MethodName))
}

func commandValueExpr(e typegenEntry) string {
	var fields []string
	if e.Description != "" {
		fields = append(fields, fmt.Sprintf("description: %s", quoteTSString(e.Description)))
	}
	if e.Usage != "" {
		fields = append(fields, fmt.Sprintf("usage: %s", quoteTSString(e.Usage)))
	}
	if len(fields) == 0 {
		return "Record<string, never>"
	}
	return "{ " + strings.Join(fields, "; ") + " }"
}

func writeRegisterAugmentation(
	b *strings.Builder,
	registerModule string,
	opts TypegenOptions,
	members []registerMember,
) {
	fmt.Fprintf(b, "declare module %s {\n", quoteTSString(registerModule))
	b.WriteString("  interface Register {\n")
	if opts.Strict {
		b.WriteString("    strict: true\n")
	}
	fmt.Fprintf(b, "    %s: {\n", quoteTSString(opts.ResourceKey))
	for _, member := range members {
		fmt.Fprintf(b, "      %s: %s\n", member.name, member.iface)
	}
	b.WriteString("    }\n")
	b.WriteString("  }\n")
	b.WriteString("}\n")
}

func resourceRegisterKey(projectPath string, resourcePath string) string {
	cleaned := filepath.ToSlash(filepath.Clean(resourcePath))

	if rel, err := filepath.Rel(projectPath, resourcePath); err == nil {
		rel = filepath.ToSlash(rel)
		if !strings.HasPrefix(rel, "../") && rel != ".." {
			cleaned = rel
		}
	}

	cleaned = strings.TrimPrefix(cleaned, "./")
	if cleaned == "" || cleaned == "." {
		cleaned = "root"
	}
	return "resource:" + cleaned
}

func (rb *ResourceBuilder) generateTypes(resourcePath string, opts TypegenOptions) (*TypegenResult, error) {
	resourcePath = filepath.Clean(resourcePath)
	outDir := filepath.Join(resourcePath, ".opencore")
	outFile := filepath.Join(outDir, typegenFileName)

	analysis, err := rb.analyzeResource(resourcePath)
	if err != nil {
		return nil, err
	}
	entries, warnings, err := typegenEntries(analysis, resourcePath, outDir)
	if err != nil {
		return nil, err
	}

	// WebView messages are collected here as well as for the view's own file: this is the file
	// that registers them with the framework, so it reports their diagnostics too.
	views, err := rb.collectViewMessages(resourcePath, outDir, "WebView messages")
	if err != nil {
		return nil, err
	}
	warnings = append(warnings, views.warnings...)
	sortValidationIssues(warnings)

	if opts.ResourceKey == "" {
		opts.ResourceKey = resourceRegisterKey(rb.projectPath, resourcePath)
	}

	changed, err := writeIfChanged(outFile, renderTypegenFile(entries, views.registered(), opts))
	if err != nil {
		return nil, err
	}

	return &TypegenResult{Entries: entries, Warnings: warnings, Changed: changed}, nil
}

func (rb *ResourceBuilder) RunTypegen(resourcePath string) (changed bool) {
	if !rb.typegenEnabled {
		rb.removeStaleTypes(resourcePath)
		return false
	}

	resourceChanged, ok := rb.runResourceTypegen(resourcePath)
	if !ok {
		return false
	}
	viewChanged := rb.runViewTypegen(resourcePath)

	return resourceChanged || viewChanged
}

func (rb *ResourceBuilder) removeStaleTypes(resourcePath string) {
	if err := removeGeneratedTypes(resourcePath); err != nil {
		fmt.Println(ui.Warning(fmt.Sprintf("Could not remove stale generated types: %v", err)))
	}
	if err := rb.removeGeneratedViewTypes(resourcePath); err != nil {
		fmt.Println(ui.Warning(fmt.Sprintf("Could not remove stale generated view types: %v", err)))
	}
}

func (rb *ResourceBuilder) runResourceTypegen(resourcePath string) (changed bool, ok bool) {
	result, err := rb.generateTypes(resourcePath, rb.typegenOptions)
	if err != nil {
		fmt.Println(ui.Warning(fmt.Sprintf(
			"Type generation skipped for %s: %v", filepath.Base(resourcePath), err)))
		return false, false
	}
	rb.reportTypegenWarnings(resourcePath, result)
	return result.Changed, true
}

func (rb *ResourceBuilder) runViewTypegen(resourcePath string) bool {
	result, hasViews, err := rb.generateViewTypes(resourcePath)
	if err != nil {
		fmt.Println(ui.Warning(fmt.Sprintf(
			"View type generation skipped for %s: %v", filepath.Base(resourcePath), err)))
		return false
	}
	if !hasViews || result == nil {
		return false
	}
	// Its warnings are not reported: the resource's own file scans the same messages and
	// already reported them.
	return result.Changed
}

func (rb *ResourceBuilder) reportTypegenWarnings(resourcePath string, result *TypegenResult) {
	if len(result.Warnings) == 0 {
		return
	}

	rb.typegenWarnedMutex.Lock()
	if rb.typegenWarned == nil {
		rb.typegenWarned = make(map[string]bool)
	}
	alreadyWarned := rb.typegenWarned[resourcePath]
	rb.typegenWarned[resourcePath] = true
	rb.typegenWarnedMutex.Unlock()

	if alreadyWarned {
		return
	}

	name := filepath.Base(resourcePath)
	fmt.Println(ui.Warning(fmt.Sprintf(
		"typegen: %d handler(s) in %s could not be typed", len(result.Warnings), name,
	)))
	for _, issue := range result.Warnings {
		fmt.Println(ui.Muted("  " + issue.String()))
	}
}

func removeGeneratedTypes(resourcePath string) error {
	outFile := filepath.Join(filepath.Clean(resourcePath), ".opencore", typegenFileName)
	if err := os.Remove(outFile); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
