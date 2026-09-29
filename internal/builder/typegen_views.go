package builder

import (
	"cmp"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// identifierPattern matches a bare JavaScript identifier, used to spot alias references inside a rendered type expression.
var identifierPattern = regexp.MustCompile(`[A-Za-z_$][A-Za-z0-9_$]*`)

const viewTypegenFileName = "opencore.gen.ts"

// viewEntry is one message discovered on the client side
type viewEntry struct {
	EventName   string
	PayloadType string
}

const payloadHelperDecl = "/** First parameter of a handler, or `undefined` when it takes none. */\n" +
	"type __Payload<P extends unknown[]> = P extends [infer First, ...unknown[]] ? First : undefined\n"

func writeViewHelpers(b *strings.Builder, entries []viewEntry) {
	if slices.ContainsFunc(entries, usesPayloadHelper) {
		b.WriteString(payloadHelperDecl)
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
	b.WriteString(doc)
	fmt.Fprintf(b, "export type %s =", name)

	unique := firstPerViewName(entries)
	if len(unique) == 0 {
		b.WriteString(" Record<string, never>\n\n")
		return
	}

	b.WriteString(" {\n")
	for _, e := range unique {
		fmt.Fprintf(b, "  %s: %s\n", quoteTSString(e.EventName), e.PayloadType)
	}
	b.WriteString("}\n\n")
}

func firstPerViewName(entries []viewEntry) []viewEntry {
	seen := map[string]bool{}
	var unique []viewEntry
	for _, e := range entries {
		if seen[e.EventName] {
			continue
		}
		seen[e.EventName] = true
		unique = append(unique, e)
	}
	return unique
}

func (rb *ResourceBuilder) generateViewTypes(resourcePath string) (*TypegenResult, bool, error) {
	resourcePath = filepath.Clean(resourcePath)

	viewPath := rb.viewPathFor(resourcePath)
	if viewPath == "" {
		return nil, false, nil
	}

	outDir := filepath.Join(viewPath, ".opencore")
	collector, err := rb.collectViewMessages(resourcePath, outDir, filepath.Base(viewPath))
	if err != nil {
		return nil, true, err
	}

	changed, err := writeIfChanged(filepath.Join(outDir, viewTypegenFileName), collector.render())
	if err != nil {
		return nil, true, err
	}

	return &TypegenResult{Warnings: collector.warnings, Changed: changed}, true, nil
}

func (rb *ResourceBuilder) collectViewMessages(resourcePath string, outDir string, label string) (*viewCollector, error) {
	analysis, err := rb.analyzeResource(resourcePath)
	if err != nil {
		return nil, err
	}
	collector := newViewCollector(outDir)
	for _, source := range analysis.sources {
		if err := collector.collectFile(source); err != nil {
			return nil, err
		}
	}
	collector.finalise(label)
	return collector, nil
}

type viewCollector struct {
	outDir string

	uiSends    []viewEntry
	uiReceives []viewEntry
	warnings   []SourceValidationIssue

	controllerImports map[string]string
	aliasByClass      map[string]string
	aliasIndex        int
}

func newViewCollector(outDir string) *viewCollector {
	return &viewCollector{
		outDir:            outDir,
		controllerImports: map[string]string{},
		aliasByClass:      map[string]string{},
	}
}

func (c *viewCollector) collectFile(source analyzedSource) error {
	if len(source.result.Views) == 0 && len(source.result.Sends) == 0 {
		return nil
	}

	importPath, err := moduleImportPath(c.outDir, source.path)
	if err != nil {
		return err
	}
	aliasFor := c.aliasFactory(importPath)

	for _, view := range source.result.Views {
		c.uiSends = append(c.uiSends, viewEntry{
			EventName: view.Event,
			PayloadType: fmt.Sprintf("__Payload<Parameters<%s[%s]>>",
				aliasFor(view.ClassName), quoteTSString(view.MethodName)),
		})
	}
	for _, send := range source.result.Sends {
		payload, err := c.payloadType(send, aliasFor)
		if err != nil {
			return err
		}
		c.uiReceives = append(c.uiReceives, viewEntry{EventName: send.Event, PayloadType: payload})
	}
	return nil
}

func (c *viewCollector) payloadType(send analyzedSend, aliasFor func(className string) string) (string, error) {
	if p := send.Payload; p.Kind == "param" {
		return fmt.Sprintf("Parameters<%s[%s]>[%d]", aliasFor(p.ClassName), quoteTSString(p.MethodName), p.Index), nil
	}
	return renderImportMarks(send.Payload.Type, c.outDir)
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

// registered returns the messages this resource contributes to the framework's `Register`.
func (c *viewCollector) registered() registeredViews {
	return registeredViews{
		send:              c.uiReceives,
		receive:           c.uiSends,
		controllerImports: pruneUnusedControllerImports(c.controllerImports, c.uiSends, c.uiReceives),
	}
}

// registeredViews holds the `viewSend`/`viewReceive` maps a resource's generated file registers.
type registeredViews struct {
	send              []viewEntry
	receive           []viewEntry
	controllerImports map[string]string
}

func (v registeredViews) empty() bool {
	return len(v.send) == 0 && len(v.receive) == 0
}

func (v registeredViews) all() []viewEntry {
	return slices.Concat(v.send, v.receive)
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

// pruneUnusedControllerImports keeps only the controller aliases some rendered payload refers to.
func pruneUnusedControllerImports(
	controllerImports map[string]string,
	entries ...[]viewEntry,
) map[string]string {
	referenced := map[string]bool{}
	for _, entry := range slices.Concat(entries...) {
		for _, identifier := range identifierPattern.FindAllString(entry.PayloadType, -1) {
			referenced[identifier] = true
		}
	}

	pruned := make(map[string]string, len(controllerImports))
	for alias, importPath := range controllerImports {
		if referenced[alias] {
			pruned[alias] = importPath
		}
	}
	return pruned
}
