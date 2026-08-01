package builder

import (
	"cmp"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	viewTypegenFileName          = "opencore.gen.ts"
	viewPayloadUnresolvedPrefix  = "could not derive the payload type of"
	viewSendNameUnresolvedPrefix = "could not resolve a WebView send() event name"
)

// viewEntry is one message discovered on the client side
type viewEntry struct {
	EventName   string
	KeyType     string
	PayloadType string
}

// methodInfo describes one method declaration and where it sits in the file
type methodInfo struct {
	name       string
	offset     int
	paramNames []string
}

func isClientSideFile(path string, text string) bool {
	segments := strings.Split(filepath.ToSlash(path), "/")
	if slices.Contains(segments, "server") {
		return false
	}
	return slices.Contains(segments, "client") || strings.Contains(text, "@Client.")
}

// splitTopLevelArgs splits a decorator/call argument list on commas that are not nested inside
// brackets or string literals.
func splitTopLevelArgs(args string) []string {
	var parts []string
	depth := 0
	var quote byte
	inQuote := false
	start := 0

	for i := 0; i < len(args); i++ {
		c := args[i]
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
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, args[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, args[start:])
	return parts
}

func parseParamNames(params string) []string {
	var names []string
	for _, part := range splitTopLevelArgs(params) {
		if name := parameterIdentifier(part); name != "" {
			names = append(names, name)
		}
	}
	return names
}

var parameterModifiers = []string{"public ", "private ", "protected ", "readonly "}

func parameterIdentifier(declaration string) string {
	name := strings.TrimSpace(declaration)
	for _, modifier := range parameterModifiers {
		name = strings.TrimPrefix(name, modifier)
	}
	name = strings.TrimPrefix(name, "...")
	if idx := strings.IndexAny(name, ":=?"); idx >= 0 {
		name = name[:idx]
	}
	return strings.TrimSpace(name)
}

func collectMethods(text string) []methodInfo {
	var methods []methodInfo

	offset := 0
	for _, line := range strings.SplitAfter(text, "\n") {
		if method, ok := methodDeclaredAt(text, line, offset); ok {
			methods = append(methods, method)
		}
		offset += len(line)
	}

	return methods
}

func methodDeclaredAt(text string, line string, offset int) (methodInfo, bool) {
	m := methodDeclPattern.FindStringSubmatch(line)
	if m == nil || nonMethodNames[m[1]] {
		return methodInfo{}, false
	}

	openRel := strings.Index(line, "(")
	if openRel < 0 {
		return methodInfo{}, false
	}
	params, _, ok := extractDecoratorArgs(text, offset+openRel)
	if !ok {
		return methodInfo{}, false
	}

	return methodInfo{name: m[1], offset: offset, paramNames: parseParamNames(params)}, true
}

func enclosingMethod(methods []methodInfo, offset int) *methodInfo {
	var found *methodInfo
	for i := range methods {
		if methods[i].offset <= offset {
			found = &methods[i]
			continue
		}
		break
	}
	return found
}

func scanFileForViewTypes(
	text string,
	relPath string,
	classes []classInfo,
	aliasFor func(className string) string,
	sourceFile string,
	selfImportPath string,
	baseDir string,
) (uiSends []viewEntry, uiReceives []viewEntry, warnings []SourceValidationIssue) {
	scan := &viewScan{
		fileScan: fileScan{
			text:       blankComments(text),
			relPath:    relPath,
			importPath: selfImportPath,
			sourceFile: sourceFile,
			baseDir:    baseDir,
			classes:    classes,
		},
		aliasFor: aliasFor,
	}
	scan.syms = parseFileSymbols(scan.text)
	scan.methods = collectMethods(scan.text)

	uiSends, sendWarnings := scan.messagesTheViewMaySend()
	uiReceives, receiveWarnings := scan.messagesTheViewReceives()

	return uiSends, uiReceives, append(sendWarnings, receiveWarnings...)
}

type viewScan struct {
	fileScan
	methods  []methodInfo
	aliasFor func(className string) string
}

func (s *viewScan) messagesTheViewMaySend() ([]viewEntry, []SourceValidationIssue) {
	var entries []viewEntry
	var warnings []SourceValidationIssue

	for _, loc := range clientOnViewHead.FindAllStringIndex(s.text, -1) {
		args, endIdx, balanced := extractDecoratorArgs(s.text, loc[1]-1)
		if !balanced {
			continue
		}

		eventName, keyType, resolved := s.resolveName(args)
		if !resolved {
			warnings = append(warnings, *s.issue(loc[0],
				"could not resolve the @Client.OnView event name; skipped by typegen"))
			continue
		}

		methodName, found := methodNameAfter(s.text, endIdx+1)
		if !found {
			continue
		}

		entries = append(entries, viewEntry{
			EventName:   eventName,
			KeyType:     keyType,
			PayloadType: s.handlerPayloadExpr(loc[0], methodName),
		})
	}

	return entries, warnings
}

func (s *viewScan) messagesTheViewReceives() ([]viewEntry, []SourceValidationIssue) {
	var entries []viewEntry
	var warnings []SourceValidationIssue

	for _, loc := range webViewSendPattern.FindAllStringIndex(s.text, -1) {
		args, _, balanced := extractDecoratorArgs(s.text, loc[1]-1)
		if !balanced {
			continue
		}

		parts := splitTopLevelArgs(args)
		if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
			continue
		}

		eventName, keyType, resolved := s.resolveName(parts[0])
		if !resolved {
			warnings = append(warnings, *s.issue(loc[0], viewSendNameUnresolvedPrefix+"; skipped by typegen"))
			continue
		}

		payload := s.sendPayloadExpr(parts, loc[0])
		if payload == "" {
			payload = "unknown"
			warnings = append(warnings, *s.issuef(loc[0],
				"%s %q; the WebView receives `unknown` (install typescript so the checker pass can resolve it, or forward the handler's parameter directly)",
				viewPayloadUnresolvedPrefix, eventName))
		}

		entries = append(entries, viewEntry{EventName: eventName, KeyType: keyType, PayloadType: payload})
	}

	return entries, warnings
}

func (s *viewScan) resolveName(arg string) (string, string, bool) {
	return resolveNameArgument(arg, s.syms, s.sourceFile, s.importPath, s.baseDir)
}

func (s *viewScan) handlerPayloadExpr(offset int, methodName string) string {
	return fmt.Sprintf("__Payload<Parameters<%s[%s]>>",
		s.aliasAt(offset), quoteTSString(methodName))
}

func (s *viewScan) sendPayloadExpr(parts []string, offset int) string {
	if len(parts) < 2 {
		return "undefined"
	}
	arg := strings.TrimSpace(parts[1])
	if arg == "undefined" {
		return "undefined"
	}
	return forwardedPayloadType(arg, s.methods, offset, s.aliasAt(offset))
}

func (s *viewScan) aliasAt(offset int) string {
	return s.aliasFor(enclosingClass(s.classes, offset))
}

func forwardedPayloadType(arg string, methods []methodInfo, offset int, classAlias string) string {
	method := enclosingMethod(methods, offset)
	if method == nil {
		return ""
	}
	for i, param := range method.paramNames {
		if param == arg {
			return fmt.Sprintf("Parameters<%s[%s]>[%d]", classAlias, quoteTSString(method.name), i)
		}
	}
	return ""
}

func resolveNameArgument(
	arg string,
	syms *fileSymbols,
	sourceFile string,
	selfImportPath string,
	baseDir string,
) (eventName string, keyType string, ok bool) {
	if literal, found := firstStringLiteral(arg); found {
		return literal, "", true
	}
	resolvedKey, display, resolved := resolveEventNameExpression(
		arg, syms, sourceFile, selfImportPath, baseDir,
	)
	if !resolved {
		return display, "", false
	}
	return display, resolvedKey, true
}

func writeViewHelpers(b *strings.Builder, entries []viewEntry) {
	if slices.ContainsFunc(entries, func(e viewEntry) bool { return e.KeyType != "" }) {
		b.WriteString("/** Single-key map, used when an event name comes from a shared const object. */\n")
		b.WriteString("type __Entry<K extends PropertyKey, V> = { [P in K]: V }\n")
	}
	if slices.ContainsFunc(entries, usesPayloadHelper) {
		b.WriteString("/** First parameter of a handler, or `undefined` when it takes none. */\n")
		b.WriteString("type __Payload<P extends unknown[]> = P extends [infer First, ...unknown[]] ? First : undefined\n")
	}
	b.WriteString("\n")
}

func usesPayloadHelper(e viewEntry) bool {
	return strings.Contains(e.PayloadType, "__Payload<")
}

func writeViewControllerAliases(b *strings.Builder, controllerImports map[string]string) {
	aliases := slices.Sorted(maps.Keys(controllerImports))
	for _, alias := range aliases {
		_, className, _ := strings.Cut(alias, "_")
		fmt.Fprintf(b, "type %s = InstanceType<typeof import(%s).%s>\n",
			alias, quoteTSString(controllerImports[alias]), className)
	}
	if len(aliases) > 0 {
		b.WriteString("\n")
	}
}

func renderViewTypesFile(
	uiSends []viewEntry,
	uiReceives []viewEntry,
	controllerImports map[string]string,
) string {
	var b strings.Builder
	writeGeneratedHeader(&b, "Types for this WebView's messages, derived from the client script.")

	if len(uiSends) == 0 && len(uiReceives) == 0 {
		b.WriteString("export type UiCanSend = Record<string, never>\n")
		b.WriteString("export type UiReceives = Record<string, never>\n")
		return b.String()
	}

	writeViewHelpers(&b, slices.Concat(uiSends, uiReceives))
	writeViewControllerAliases(&b, controllerImports)

	writeViewMap(&b, "UiCanSend",
		"/** Messages this WebView may send to the client (from `@Client.OnView` handlers). */\n",
		uiSends)
	writeViewMap(&b, "UiReceives",
		"/** Messages the client sends to this WebView (from its `send()` calls). */\n",
		uiReceives)

	return b.String()
}

func writeViewMap(b *strings.Builder, name string, doc string, entries []viewEntry) {
	literals, computed := partitionViewEntries(entries)

	b.WriteString(doc)
	fmt.Fprintf(b, "export type %s =", name)

	if len(literals) == 0 && len(computed) == 0 {
		b.WriteString(" Record<string, never>\n\n")
		return
	}

	if len(literals) > 0 {
		b.WriteString(" {\n")
		for _, e := range literals {
			fmt.Fprintf(b, "  %s: %s\n", quoteTSString(e.EventName), e.PayloadType)
		}
		b.WriteString("}")
	}
	for i, e := range computed {
		if len(literals) > 0 || i > 0 {
			b.WriteString("\n  &")
		}
		fmt.Fprintf(b, " __Entry<%s, %s> /* %s */", e.KeyType, e.PayloadType, e.EventName)
	}
	b.WriteString("\n\n")
}

func partitionViewEntries(entries []viewEntry) (literals []viewEntry, computed []viewEntry) {
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

func (rb *ResourceBuilder) generateViewTypes(resourcePath string) (*TypegenResult, bool, error) {
	resourcePath = filepath.Clean(resourcePath)

	viewPath := rb.viewPathFor(resourcePath)
	if viewPath == "" {
		return nil, false, nil
	}

	outDir := filepath.Join(viewPath, ".opencore")
	collector := newViewCollector(resourcePath, outDir)

	err := walkSourceFiles(resourcePath, viewPath, collector.collectFile)
	if err != nil {
		return nil, true, err
	}

	collector.uiReceives, collector.warnings = rb.refineViewPayloads(
		resourcePath, viewPath, outDir, collector.scannedFiles,
		collector.uiReceives, collector.warnings,
	)
	collector.finalise(filepath.Base(viewPath))

	changed, err := writeIfChanged(filepath.Join(outDir, viewTypegenFileName), collector.render())
	if err != nil {
		return nil, true, err
	}

	return &TypegenResult{Warnings: collector.warnings, Changed: changed}, true, nil
}

type viewCollector struct {
	resourcePath string
	outDir       string

	uiSends      []viewEntry
	uiReceives   []viewEntry
	warnings     []SourceValidationIssue
	scannedFiles []string

	controllerImports map[string]string
	aliasByClass      map[string]string
	aliasIndex        int
}

func newViewCollector(resourcePath string, outDir string) *viewCollector {
	return &viewCollector{
		resourcePath:      resourcePath,
		outDir:            outDir,
		controllerImports: map[string]string{},
		aliasByClass:      map[string]string{},
	}
}

func (c *viewCollector) collectFile(path string, text string) error {
	if !isViewMessageSource(path, text) {
		return nil
	}

	classes := collectClasses(text)
	if len(classes) == 0 {
		return nil
	}

	importPath, err := moduleImportPath(c.outDir, path)
	if err != nil {
		return err
	}

	c.scannedFiles = append(c.scannedFiles, path)

	sends, receives, warnings := scanFileForViewTypes(
		text, relativeSourcePath(c.resourcePath, path), classes,
		c.aliasFactory(importPath), path, importPath, c.outDir,
	)

	c.uiSends = append(c.uiSends, sends...)
	c.uiReceives = append(c.uiReceives, receives...)
	c.warnings = append(c.warnings, warnings...)
	return nil
}

func (c *viewCollector) aliasFactory(importPath string) func(className string) string {
	return func(className string) string {
		key := importPath + "#" + className
		if existing, seen := c.aliasByClass[key]; seen {
			return existing
		}
		alias := fmt.Sprintf("V%d_%s", c.aliasIndex, className)
		c.aliasIndex++
		c.aliasByClass[key] = alias
		c.controllerImports[alias] = importPath
		return alias
	}
}

func (c *viewCollector) finalise(viewName string) {
	sortViewEntries(c.uiSends)
	sortViewEntries(c.uiReceives)
	c.warnings = append(c.warnings, duplicateViewWarnings(viewName, c.uiReceives)...)
	sortValidationIssues(c.warnings)
}

func (c *viewCollector) render() string {
	return renderViewTypesFile(
		c.uiSends, c.uiReceives,
		pruneUnusedControllerImports(c.controllerImports, c.uiSends, c.uiReceives),
	)
}

func isViewMessageSource(path string, text string) bool {
	if !isClientSideFile(path, text) {
		return false
	}
	return strings.Contains(text, "@Client.OnView") || strings.Contains(text, ".send(")
}

func writeIfChanged(outFile string, content string) (bool, error) {
	if existing, err := os.ReadFile(outFile); err == nil && string(existing) == content {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(outFile), 0755); err != nil {
		return false, err
	}
	if err := os.WriteFile(outFile, []byte(content), 0644); err != nil {
		return false, err
	}
	return true, nil
}

func sortViewEntries(entries []viewEntry) {
	slices.SortStableFunc(entries, func(a, b viewEntry) int {
		return cmp.Or(
			strings.Compare(a.EventName, b.EventName),
			strings.Compare(a.PayloadType, b.PayloadType),
		)
	})
}

func duplicateViewWarnings(relFile string, entries []viewEntry) []SourceValidationIssue {
	var warnings []SourceValidationIssue

	for i := 1; i < len(entries); i++ {
		previous, current := entries[i-1], entries[i]
		if previous.EventName != current.EventName || previous.PayloadType == current.PayloadType {
			continue
		}
		warnings = append(warnings, SourceValidationIssue{
			File: relFile,
			Message: fmt.Sprintf(
				"WebView message %q is sent with two different payload types (%s and %s); the view is typed with the first",
				current.EventName, previous.PayloadType, current.PayloadType,
			),
		})
	}

	return warnings
}

func (rb *ResourceBuilder) removeGeneratedViewTypes(resourcePath string) error {
	viewPath := rb.viewPathFor(filepath.Clean(resourcePath))
	if viewPath == "" {
		return nil
	}
	outFile := filepath.Join(viewPath, ".opencore", viewTypegenFileName)
	if err := os.Remove(outFile); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
