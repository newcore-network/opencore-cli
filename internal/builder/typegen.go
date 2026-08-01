package builder

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/newcore-network/opencore-cli/internal/ui"
)

var (
	// Decorator heads. The argument list is extracted separately via a balanced-paren scan so
	// that declarations spanning multiple lines are handled.
	serverOnNetHead   = regexp.MustCompile(`@Server\.OnNet\s*\(`)
	serverOnRPCHead   = regexp.MustCompile(`@Server\.OnRPC\s*\(`)
	serverCommandHead = regexp.MustCompile(`@Server\.Command\s*\(`)
	clientOnNetHead   = regexp.MustCompile(`@Client\.OnNet\s*\(`)
	clientOnRPCHead   = regexp.MustCompile(`@Client\.OnRPC\s*\(`)

	// `export class Foo`, `class Foo`, `export default class Foo`.
	classDeclPattern = regexp.MustCompile(`(?m)^\s*(?:export\s+)?(?:default\s+)?(?:abstract\s+)?class\s+([A-Za-z_$][A-Za-z0-9_$]*)`)

	// A method declaration line: optional modifiers, a name, optional type params, then `(`.
	methodDeclPattern = regexp.MustCompile(`^\s*(?:(?:public|private|protected|readonly|static|async|override)\s+)*([A-Za-z_$][A-Za-z0-9_$]*)\s*(?:<[^>]*>)?\s*\(`)

	// First single/double/back-quoted literal inside a decorator's argument list.
	stringLiteralPattern = regexp.MustCompile("^\\s*(?:'([^']*)'|\"([^\"]*)\"|`([^`$\\\\]*)`)")

	// A dotted identifier reference such as `CharacterNetEvents.UI_OPEN`, which is how event
	// names are written in practice once a project keeps them in a shared const object.
	identifierRefPattern = regexp.MustCompile(`^\s*([A-Za-z_$][A-Za-z0-9_$]*)((?:\s*\.\s*[A-Za-z_$][A-Za-z0-9_$]*)*)\s*(?:,|\)|$)`)

	// `import { A, B as C } from 'spec'` / `import type { … } from 'spec'`.
	namedImportPattern = regexp.MustCompile(`(?m)^\s*import\s+(?:type\s+)?\{([^}]*)\}\s*from\s*['"]([^'"]+)['"]`)
	// `import * as NS from 'spec'`.
	namespaceImportPattern = regexp.MustCompile(`(?m)^\s*import\s+(?:type\s+)?\*\s+as\s+([A-Za-z_$][A-Za-z0-9_$]*)\s+from\s*['"]([^'"]+)['"]`)
	// `import Default from 'spec'`.
	defaultImportPattern = regexp.MustCompile(`(?m)^\s*import\s+(?:type\s+)?([A-Za-z_$][A-Za-z0-9_$]*)\s*(?:,\s*\{[^}]*\})?\s*from\s*['"]([^'"]+)['"]`)

	// `someView.send(` / `WebView.send(` — the client pushing a message into a WebView.
	webViewSendPattern = regexp.MustCompile(`(?:this\s*\.\s*)?([A-Za-z_$][A-Za-z0-9_$]*)\s*\.\s*send\s*\(`)

	// `@Client.OnView(` — a handler for messages arriving from a WebView.
	clientOnViewHead = regexp.MustCompile(`@Client\.OnView\s*\(`)

	// `export const NAME =` — a constant this file exposes to the generated module.
	exportedConstPattern = regexp.MustCompile(`(?m)^\s*export\s+const\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*=`)
	// `const NAME = 'literal'` — a file-local constant we can inline directly.
	localStringConstPattern = regexp.MustCompile("(?m)^\\s*(?:export\\s+)?const\\s+([A-Za-z_$][A-Za-z0-9_$]*)\\s*(?::[^=]+)?=\\s*(?:'([^']*)'|\"([^\"]*)\"|`([^`$\\\\]*)`)\\s*(?:as\\s+const\\s*)?;?\\s*$")

	// `command: 'name'` inside a @Server.Command config object.
	commandFieldPattern     = regexp.MustCompile("command\\s*:\\s*(?:'([^']*)'|\"([^\"]*)\"|`([^`$\\\\]*)`)")
	descriptionFieldPattern = regexp.MustCompile("description\\s*:\\s*(?:'([^']*)'|\"([^\"]*)\"|`([^`$\\\\]*)`)")
	usageFieldPattern       = regexp.MustCompile("usage\\s*:\\s*(?:'([^']*)'|\"([^\"]*)\"|`([^`$\\\\]*)`)")
)

// Names that look like a method declaration but are control flow or a constructor.
var nonMethodNames = map[string]bool{
	"constructor": true,
	"if":          true,
	"for":         true,
	"while":       true,
	"switch":      true,
	"catch":       true,
	"return":      true,
	"function":    true,
	"do":          true,
}

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
	Kind       typegenKind
	EventName  string // source text of the name, used for ordering and de-duplication
	ImportPath string // relative to the .opencore directory, extension stripped
	ClassName  string
	MethodName string
	// KeyType is a TypeScript type expression for the event-name key. Empty means EventName is
	// a plain literal and can be quoted directly.
	KeyType     string
	Description string
	Usage       string
	// SourceFile and SourceLine locate the decorator, for diagnostics only.
	SourceFile string
	SourceLine int
}

// fileSymbols captures the bindings of one source file that the name resolver needs.
type fileSymbols struct {
	// importedFrom maps a local binding to the module specifier it came from.
	importedFrom map[string]string
	// exportedConsts holds names reachable through `typeof import(...)` on the file itself.
	exportedConsts map[string]bool
	// localStrings holds constants with a plain string initialiser, inlinable even when private.
	localStrings map[string]string
}

func parseFileSymbols(text string) *fileSymbols {
	return &fileSymbols{
		importedFrom:   collectImportBindings(text),
		exportedConsts: collectExportedConsts(text),
		localStrings:   collectLocalStringConsts(text),
	}
}

func collectImportBindings(text string) map[string]string {
	bindings := map[string]string{}

	for _, m := range namedImportPattern.FindAllStringSubmatch(text, -1) {
		for clause := range strings.SplitSeq(m[1], ",") {
			if local := localBindingOf(clause); local != "" {
				bindings[local] = m[2]
			}
		}
	}
	for _, m := range namespaceImportPattern.FindAllStringSubmatch(text, -1) {
		bindings[m[1]] = m[2]
	}
	for _, m := range defaultImportPattern.FindAllStringSubmatch(text, -1) {
		if _, exists := bindings[m[1]]; !exists {
			bindings[m[1]] = m[2]
		}
	}

	return bindings
}

func localBindingOf(clause string) string {
	local := strings.TrimPrefix(strings.TrimSpace(clause), "type ")
	if _, alias, renamed := strings.Cut(local, " as "); renamed {
		local = alias
	}
	return strings.TrimSpace(local)
}

func collectExportedConsts(text string) map[string]bool {
	consts := map[string]bool{}
	for _, m := range exportedConstPattern.FindAllStringSubmatch(text, -1) {
		consts[m[1]] = true
	}
	return consts
}

func collectLocalStringConsts(text string) map[string]string {
	consts := map[string]string{}
	for _, m := range localStringConstPattern.FindAllStringSubmatch(text, -1) {
		if literal, ok := firstNonEmpty(m[2:]); ok {
			consts[m[1]] = literal
		}
	}
	return consts
}

func firstNonEmpty(groups []string) (string, bool) {
	for _, group := range groups {
		if group != "" {
			return group, true
		}
	}
	return "", false
}

func resolveModuleImportPath(spec string, sourceFile string, baseDir string) (string, bool) {
	if !strings.HasPrefix(spec, ".") {
		return spec, true
	}

	absolute := filepath.Join(filepath.Dir(sourceFile), spec)
	rel, err := filepath.Rel(baseDir, absolute)
	if err != nil {
		return "", false
	}
	rel = filepath.ToSlash(rel)
	if !strings.HasPrefix(rel, ".") {
		rel = "./" + rel
	}
	return rel, true
}

func resolveEventNameExpression(
	args string,
	syms *fileSymbols,
	sourceFile string,
	selfImportPath string,
	baseDir string,
) (keyType string, display string, ok bool) {
	m := identifierRefPattern.FindStringSubmatch(args)
	if m == nil {
		return "", "", false
	}

	base := m[1]
	var props []string
	for part := range strings.SplitSeq(m[2], ".") {
		if part = strings.TrimSpace(part); part != "" {
			props = append(props, part)
		}
	}

	display = dottedName(base, props)

	if literal, inlinable := syms.inlinableLocalConst(base, props); inlinable {
		return "", literal, true
	}

	moduleExpr, ok := syms.moduleExprFor(base, sourceFile, selfImportPath, baseDir)
	if !ok {
		return "", display, false
	}

	for _, prop := range props {
		moduleExpr += fmt.Sprintf("[%s]", quoteTSString(prop))
	}

	return moduleExpr, display, true
}

func dottedName(base string, props []string) string {
	if len(props) == 0 {
		return base
	}
	return base + "." + strings.Join(props, ".")
}

func (s *fileSymbols) inlinableLocalConst(base string, props []string) (string, bool) {
	if len(props) > 0 {
		return "", false
	}
	literal, found := s.localStrings[base]
	return literal, found
}

func (s *fileSymbols) moduleExprFor(
	base string,
	sourceFile string,
	selfImportPath string,
	baseDir string,
) (string, bool) {
	if spec, imported := s.importedFrom[base]; imported {
		resolved, resolvable := resolveModuleImportPath(spec, sourceFile, baseDir)
		if !resolvable {
			return "", false
		}
		return fmt.Sprintf("typeof import(%s).%s", quoteTSString(resolved), base), true
	}
	if s.exportedConsts[base] {
		return fmt.Sprintf("typeof import(%s).%s", quoteTSString(selfImportPath), base), true
	}
	return "", false
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

func extractDecoratorArgs(text string, openIdx int) (string, int, bool) {
	depth := 0
	var quote byte
	inQuote := false

	for i := openIdx; i < len(text); i++ {
		c := text[i]

		if inQuote {
			if c == '\\' {
				i++
				continue
			}
			if c == quote {
				inQuote = false
			}
			continue
		}

		switch c {
		case '\'', '"', '`':
			inQuote = true
			quote = c
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return text[openIdx+1 : i], i, true
			}
		}
	}

	return "", 0, false
}

func firstStringLiteral(args string) (string, bool) {
	return matchField(stringLiteralPattern, args)
}

func matchField(pattern *regexp.Regexp, text string) (string, bool) {
	m := pattern.FindStringSubmatch(text)
	if m == nil {
		return "", false
	}
	return firstNonEmpty(m[1:])
}

func methodNameAfter(text string, endIdx int) (string, bool) {
	for line := range strings.SplitSeq(text[endIdx:], "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "@") || strings.HasPrefix(trimmed, ")") {
			continue
		}
		m := methodDeclPattern.FindStringSubmatch(line)
		if m == nil || nonMethodNames[m[1]] {
			return "", false
		}
		return m[1], true
	}
	return "", false
}

type scanState int

const (
	stateCode scanState = iota
	stateLineComment
	stateBlockComment
	stateQuoted
)

func blankComments(text string) string {
	blanker := &commentBlanker{out: []byte(text), state: stateCode}
	return blanker.run()
}

type commentBlanker struct {
	out       []byte
	state     scanState
	quoteChar byte
	i         int
}

func (b *commentBlanker) run() string {
	for b.i = 0; b.i < len(b.out); b.i++ {
		switch b.state {
		case stateCode:
			b.stepCode()
		case stateLineComment:
			b.stepLineComment()
		case stateBlockComment:
			b.stepBlockComment()
		case stateQuoted:
			b.stepQuoted()
		}
	}
	return string(b.out)
}

func (b *commentBlanker) stepCode() {
	switch {
	case b.opensWith('/'):
		b.state = stateLineComment
	case b.opensWith('*'):
		b.state = stateBlockComment
	case b.isQuote():
		b.state = stateQuoted
		b.quoteChar = b.out[b.i]
		return
	default:
		return
	}
	b.blankPair()
}

func (b *commentBlanker) stepLineComment() {
	if b.out[b.i] == '\n' {
		b.state = stateCode
		return
	}
	b.blank(b.i)
}

func (b *commentBlanker) stepBlockComment() {
	closing := b.out[b.i] == '*' && b.peekIs('/')
	b.blank(b.i)
	if closing {
		b.blankPair()
		b.state = stateCode
	}
}

func (b *commentBlanker) stepQuoted() {
	switch b.out[b.i] {
	case '\\':
		b.i++
	case b.quoteChar:
		b.state = stateCode
	}
}

func (b *commentBlanker) opensWith(marker byte) bool {
	return b.out[b.i] == '/' && b.peekIs(marker)
}

func (b *commentBlanker) peekIs(c byte) bool {
	return b.i+1 < len(b.out) && b.out[b.i+1] == c
}

func (b *commentBlanker) isQuote() bool {
	c := b.out[b.i]
	return c == '\'' || c == '"' || c == '`'
}

func (b *commentBlanker) blankPair() {
	b.blank(b.i)
	b.blank(b.i + 1)
	b.i++
}

func (b *commentBlanker) blank(i int) {
	if b.out[i] != '\n' && b.out[i] != '\r' {
		b.out[i] = ' '
	}
}

type classInfo struct {
	name   string
	offset int
}

func collectClasses(text string) []classInfo {
	var classes []classInfo
	for _, loc := range classDeclPattern.FindAllStringSubmatchIndex(text, -1) {
		if loc[2] < 0 {
			continue
		}
		classes = append(classes, classInfo{name: text[loc[2]:loc[3]], offset: loc[0]})
	}
	return classes
}

func enclosingClass(classes []classInfo, offset int) string {
	name := classes[0].name
	for _, class := range classes {
		if class.offset > offset {
			break
		}
		name = class.name
	}
	return name
}

// lineOf reports the 1-based line number of a byte offset.
func lineOf(text string, offset int) int {
	if offset > len(text) {
		offset = len(text)
	}
	return strings.Count(text[:offset], "\n") + 1
}

type decoratorSpec struct {
	head *regexp.Regexp
	kind typegenKind
}

var typegenDecorators = []decoratorSpec{
	{serverOnNetHead, kindServerNet},
	{clientOnNetHead, kindClientNet},
	{serverOnRPCHead, kindServerRPC},
	{clientOnRPCHead, kindClientRPC},
	{serverCommandHead, kindCommand},
}

type fileScan struct {
	text       string
	relPath    string
	importPath string
	sourceFile string
	baseDir    string
	classes    []classInfo
	syms       *fileSymbols
}

func scanFileForTypegen(
	text string,
	relPath string,
	importPath string,
	sourceFile string,
	baseDir string,
) ([]typegenEntry, []SourceValidationIssue) {
	blanked := blankComments(text)

	classes := collectClasses(blanked)
	if len(classes) == 0 {
		return nil, nil
	}

	scan := &fileScan{
		text:       blanked,
		relPath:    relPath,
		importPath: importPath,
		sourceFile: sourceFile,
		baseDir:    baseDir,
		classes:    classes,
		syms:       parseFileSymbols(blanked),
	}

	var entries []typegenEntry
	var warnings []SourceValidationIssue

	for _, spec := range typegenDecorators {
		for _, loc := range spec.head.FindAllStringIndex(scan.text, -1) {
			entry, warning := scan.entryAt(spec, loc[0], loc[1]-1)
			if warning != nil {
				warnings = append(warnings, *warning)
				continue
			}
			entries = append(entries, *entry)
		}
	}

	return entries, warnings
}

func (s *fileScan) entryAt(spec decoratorSpec, start int, openIdx int) (*typegenEntry, *SourceValidationIssue) {
	args, endIdx, balanced := extractDecoratorArgs(s.text, openIdx)
	if !balanced {
		return nil, s.issue(start, "unbalanced parentheses in decorator; skipped by typegen")
	}

	name, issue := s.resolveName(spec.kind, args, start)
	if issue != nil {
		return nil, issue
	}

	methodName, found := methodNameAfter(s.text, endIdx+1)
	if !found {
		return nil, s.issuef(start,
			"could not resolve the method decorated for %q; skipped by typegen", name.event)
	}

	metadata := commandMetadata(spec.kind, args)

	return &typegenEntry{
		Kind:        spec.kind,
		EventName:   name.event,
		KeyType:     name.keyType,
		ImportPath:  s.importPath,
		ClassName:   enclosingClass(s.classes, start),
		MethodName:  methodName,
		Description: metadata.description,
		Usage:       metadata.usage,
		SourceFile:  s.relPath,
		SourceLine:  lineOf(s.text, start),
	}, nil
}

type resolvedName struct {
	event   string
	keyType string
}

func (s *fileScan) resolveName(kind typegenKind, args string, start int) (resolvedName, *SourceValidationIssue) {
	if literal, ok := firstStringLiteral(args); ok {
		return resolvedName{event: literal}, nil
	}
	if kind == kindCommand {
		if configured, ok := matchField(commandFieldPattern, args); ok {
			return resolvedName{event: configured}, nil
		}
	}

	keyType, display, ok := resolveEventNameExpression(args, s.syms, s.sourceFile, s.importPath, s.baseDir)
	if !ok {
		return resolvedName{}, s.issuef(start,
			"could not resolve the event name %q; skipped by typegen (use a string literal or an imported const)",
			strings.TrimSpace(display))
	}
	return resolvedName{event: display, keyType: keyType}, nil
}

type commandFields struct {
	description string
	usage       string
}

func commandMetadata(kind typegenKind, args string) commandFields {
	if kind != kindCommand {
		return commandFields{}
	}
	description, _ := matchField(descriptionFieldPattern, args)
	usage, _ := matchField(usageFieldPattern, args)
	return commandFields{description: description, usage: usage}
}

func (s *fileScan) issue(offset int, message string) *SourceValidationIssue {
	return &SourceValidationIssue{
		File:    s.relPath,
		Line:    lineOf(s.text, offset),
		Message: message,
	}
}

func (s *fileScan) issuef(offset int, format string, args ...any) *SourceValidationIssue {
	return s.issue(offset, fmt.Sprintf(format, args...))
}

func walkSourceFiles(resourcePath string, viewPath string, visit func(path string, text string) error) error {
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

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		return visit(path, string(content))
	})
}

func isScannableSource(name string) bool {
	return strings.HasSuffix(name, ".ts") && !strings.HasSuffix(name, ".d.ts")
}

func containsDecorators(text string) bool {
	return strings.Contains(text, "@Server.") || strings.Contains(text, "@Client.")
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

// scanResourceForTypegen walks a resource and collects every typegen entry in it.
func scanResourceForTypegen(resourcePath string, baseDir string, viewPath string) ([]typegenEntry, []SourceValidationIssue, error) {
	var entries []typegenEntry
	var warnings []SourceValidationIssue

	err := walkSourceFiles(resourcePath, viewPath, func(path string, text string) error {
		if !containsDecorators(text) {
			return nil
		}

		importPath, err := moduleImportPath(baseDir, path)
		if err != nil {
			return err
		}

		fileEntries, fileWarnings := scanFileForTypegen(
			text, relativeSourcePath(resourcePath, path), importPath, path, baseDir,
		)
		entries = append(entries, fileEntries...)
		warnings = append(warnings, fileWarnings...)
		return nil
	})
	if err != nil {
		return nil, nil, err
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

func renderTypegenFile(entries []typegenEntry, opts TypegenOptions) string {
	var b strings.Builder
	writeGeneratedHeader(&b, "Regenerated by `opencore dev` and `opencore build`.")

	if len(entries) == 0 {
		b.WriteString("export {};\n")
		return b.String()
	}

	registerModule := registerModuleOf(opts)
	aliases := newControllerAliases(entries)
	byKind := groupByKind(entries)
	registered := sectionsPresentIn(byKind)

	writeTypegenPreamble(&b, entries, byKind, registerModule)
	aliases.writeDeclarations(&b)

	for _, section := range registered {
		writeSectionMap(&b, section, byKind[section.kind], aliases)
	}
	writeRegisterAugmentation(&b, registerModule, opts, registered)

	return b.String()
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
	entries []typegenEntry,
	byKind map[typegenKind][]typegenEntry,
	registerModule string,
) {
	if len(byKind[kindServerNet]) > 0 || len(byKind[kindServerRPC]) > 0 {
		fmt.Fprintf(b, "import type { DropFirst } from %s\n\n", quoteTSString(registerModule))
	}
	if slices.ContainsFunc(entries, func(e typegenEntry) bool { return e.KeyType != "" }) {
		b.WriteString("/** Single-key map, used when an event name comes from a shared const object. */\n")
		b.WriteString("type __Entry<K extends PropertyKey, V> = { [P in K]: V }\n\n")
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
	b.WriteString("\n")
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
	literals, computed := partitionByKeyKind(entries)
	fmt.Fprintf(b, "export type %s =", section.iface)

	if len(literals) > 0 {
		b.WriteString(" {\n")
		for _, e := range literals {
			fmt.Fprintf(b, "  %s: %s\n", quoteTSString(e.EventName), section.valueExpr(e, aliases))
		}
		b.WriteString("}")
	}
	for i, e := range computed {
		if len(literals) > 0 || i > 0 {
			b.WriteString("\n  &")
		}
		fmt.Fprintf(b, " __Entry<%s, %s> /* %s */",
			e.KeyType, section.valueExpr(e, aliases), e.EventName)
	}

	b.WriteString("\n\n")
}

func partitionByKeyKind(entries []typegenEntry) (literals []typegenEntry, computed []typegenEntry) {
	seen := map[string]bool{}
	for _, e := range entries {
		if seen[e.EventName] {
			continue
		}
		seen[e.EventName] = true
		if e.KeyType == "" {
			literals = append(literals, e)
		} else {
			computed = append(computed, e)
		}
	}
	return literals, computed
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
	sections []typegenSection,
) {
	fmt.Fprintf(b, "declare module %s {\n", quoteTSString(registerModule))
	b.WriteString("  interface Register {\n")
	if opts.Strict {
		b.WriteString("    strict: true\n")
	}
	fmt.Fprintf(b, "    %s: {\n", quoteTSString(opts.ResourceKey))
	for _, section := range sections {
		fmt.Fprintf(b, "      %s: %s\n", section.registerAs, section.iface)
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

	entries, warnings, err := scanResourceForTypegen(resourcePath, outDir, rb.viewPathFor(resourcePath))
	if err != nil {
		return nil, err
	}

	if opts.ResourceKey == "" {
		opts.ResourceKey = resourceRegisterKey(rb.projectPath, resourcePath)
	}

	changed, err := writeIfChanged(outFile, renderTypegenFile(entries, opts))
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
	rb.reportTypegenWarnings(resourcePath+"/views", result)
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
