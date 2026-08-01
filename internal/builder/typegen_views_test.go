package builder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readViewGenFile(t *testing.T, resourcePath string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(resourcePath, "ui", ".opencore", viewTypegenFileName))
	if err != nil {
		t.Fatalf("failed to read generated view file: %v", err)
	}
	return string(content)
}

func TestGenerateViewTypes_BothDirections(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	// A view directory must exist for view typegen to run at all.
	if err := os.MkdirAll(filepath.Join(resourcePath, "ui"), 0755); err != nil {
		t.Fatal(err)
	}

	writeTestFile(t, resourcePath, "src/client/character-ui.controller.ts", `
import { Client } from '@open-core/framework/client'

@Client.Controller()
export class CharacterUiController {
  // UI → client: what the WebView may send.
  @Client.OnView('character:select')
  onSelect(payload: SelectPayload): void {}

  @Client.OnView('character:ready')
  onReady(): void {}

  // client → UI: what the WebView receives. The payload forwards this handler's parameter.
  @Client.OnNet('character:ui:open')
  onOpen(state: CharacterUiState): void {
    this.characterView.send('character:show', state)
  }
}
`)

	result, hasViews, err := rb.generateViewTypes(resourcePath)
	if err != nil {
		t.Fatalf("generateViewTypes returned error: %v", err)
	}
	if !hasViews {
		t.Fatal("expected the ui/ directory to be detected")
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", result.Warnings)
	}

	content := readViewGenFile(t, resourcePath)

	// Direction 1: names from @Client.OnView, payload = the handler's first parameter.
	// `__Payload` rather than a bare `[0]`, so a no-arg handler does not index an empty tuple.
	if !strings.Contains(content, "'character:select': __Payload<Parameters<") {
		t.Fatalf("expected the UI send map to be derived from OnView handlers, got:\n%s", content)
	}
	if !strings.Contains(content, "'character:ready': __Payload<Parameters<") {
		t.Fatalf("expected a no-arg handler to resolve to undefined, got:\n%s", content)
	}
	if !strings.Contains(content, "export type UiCanSend") {
		t.Fatalf("expected UiCanSend, got:\n%s", content)
	}

	// Direction 2: names from send() calls, payload recovered from the enclosing handler.
	if !strings.Contains(content, "'character:show': Parameters<") {
		t.Fatalf("expected the UI receive map to recover the forwarded payload, got:\n%s", content)
	}
	if !strings.Contains(content, "export type UiReceives") {
		t.Fatalf("expected UiReceives, got:\n%s", content)
	}

	// The UI does not depend on the framework, so nothing may be augmented.
	if strings.Contains(content, "declare module") {
		t.Fatalf("the view file must not augment the framework, got:\n%s", content)
	}

	// Import paths must be relative to the view's .opencore directory.
	if !strings.Contains(content, "'../../src/client/character-ui.controller'") {
		t.Fatalf("expected an import path relative to ui/.opencore, got:\n%s", content)
	}
}

func TestGenerateViewTypes_HonoursConfiguredViewPath(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	viewPath := filepath.Join(resourcePath, "panel")
	if err := os.MkdirAll(viewPath, 0755); err != nil {
		t.Fatal(err)
	}
	rb.SetViewPathResolver(func(string) string { return viewPath })

	writeTestFile(t, resourcePath, "src/client/a.controller.ts", `
@Client.Controller()
export class AController {
  @Client.OnView('panel:submit')
  onSubmit(payload: SubmitPayload): void {}
}
`)

	_, hasViews, err := rb.generateViewTypes(resourcePath)
	if err != nil {
		t.Fatalf("generateViewTypes returned error: %v", err)
	}
	if !hasViews {
		t.Fatal("expected the configured view directory to be used")
	}

	content, err := os.ReadFile(filepath.Join(viewPath, ".opencore", viewTypegenFileName))
	if err != nil {
		t.Fatalf("expected the generated file in the configured view directory: %v", err)
	}
	if !strings.Contains(string(content), "'panel:submit'") {
		t.Fatalf("expected the OnView handler in the generated file, got:\n%s", content)
	}
}

func TestGenerateTypes_SkipsConfiguredViewDirectory(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	viewPath := filepath.Join(resourcePath, "panel")
	if err := os.MkdirAll(viewPath, 0755); err != nil {
		t.Fatal(err)
	}
	rb.SetViewPathResolver(func(string) string { return viewPath })

	writeTestFile(t, resourcePath, "panel/stray.controller.ts", `
@Client.Controller()
export class StrayController {
  @Client.OnNet('panel:stray')
  onStray(): void {}
}
`)

	if _, err := rb.generateTypes(resourcePath, TypegenOptions{}); err != nil {
		t.Fatalf("generateTypes returned error: %v", err)
	}

	content := readGenFile(t, resourcePath)
	if strings.Contains(content, "panel:stray") {
		t.Fatalf("the configured view directory must not be scanned as client code, got:\n%s", content)
	}
}

func TestGenerateViewTypes_DiscoversNuiDirectory(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	if err := os.MkdirAll(filepath.Join(resourcePath, "nui"), 0755); err != nil {
		t.Fatal(err)
	}

	writeTestFile(t, resourcePath, "src/client/a.controller.ts", `
@Client.Controller()
export class AController {
  @Client.OnView('nui:submit')
  onSubmit(payload: SubmitPayload): void {}
}
`)

	_, hasViews, err := rb.generateViewTypes(resourcePath)
	if err != nil {
		t.Fatalf("generateViewTypes returned error: %v", err)
	}
	if !hasViews {
		t.Fatal("expected nui/ to be discovered as a view directory")
	}
	if _, err := os.Stat(filepath.Join(resourcePath, "nui", ".opencore", viewTypegenFileName)); err != nil {
		t.Fatalf("expected the generated file in nui/.opencore: %v", err)
	}
}

func TestGenerateViewTypes_AttributesHandlersToTheirOwnClass(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	if err := os.MkdirAll(filepath.Join(resourcePath, "ui"), 0755); err != nil {
		t.Fatal(err)
	}

	writeTestFile(t, resourcePath, "src/client/pair.controller.ts", `
@Client.Controller()
export class FirstUiController {
  @Client.OnView('first:submit')
  onSubmit(payload: FirstPayload): void {}
}

@Client.Controller()
export class SecondUiController {
  @Client.OnView('second:submit')
  onSecondSubmit(payload: SecondPayload): void {}
}
`)

	if _, _, err := rb.generateViewTypes(resourcePath); err != nil {
		t.Fatalf("generateViewTypes returned error: %v", err)
	}

	content := readViewGenFile(t, resourcePath)

	if !strings.Contains(content, "'first:submit': __Payload<Parameters<V0_FirstUiController['onSubmit']>>") {
		t.Fatalf("expected the first handler on its own class, got:\n%s", content)
	}
	if !strings.Contains(content, "'second:submit': __Payload<Parameters<V1_SecondUiController['onSecondSubmit']>>") {
		t.Fatalf("expected the second handler on SecondUiController, got:\n%s", content)
	}
	// Both classes come from the same file, so both aliases must import the same module.
	if strings.Count(content, "'../../src/client/pair.controller'") != 2 {
		t.Fatalf("expected both aliases to import the controller file, got:\n%s", content)
	}
}

func TestGenerateViewTypes_NoViewDirectoryIsSkipped(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	writeTestFile(t, resourcePath, "src/client/a.controller.ts", `
@Client.Controller()
export class AController {
  @Client.OnView('x:y')
  onY(): void {}
}
`)

	_, hasViews, err := rb.generateViewTypes(resourcePath)
	if err != nil {
		t.Fatalf("generateViewTypes returned error: %v", err)
	}
	if hasViews {
		t.Fatal("expected no view directory to be detected")
	}
}

func TestGenerateViewTypes_ResolvesConstReferences(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	if err := os.MkdirAll(filepath.Join(resourcePath, "ui"), 0755); err != nil {
		t.Fatal(err)
	}

	writeTestFile(t, resourcePath, "src/shared/view-events.ts", `
export const CharacterViewEvents = {
  SELECT: 'character:select',
  SHOW: 'character:show',
} as const
`)
	writeTestFile(t, resourcePath, "src/client/ui.controller.ts", `
import { CharacterViewEvents } from '../shared/view-events'

@Client.Controller()
export class UiController {
  @Client.OnView(CharacterViewEvents.SELECT)
  onSelect(payload: SelectPayload): void {}
}
`)

	if _, _, err := rb.generateViewTypes(resourcePath); err != nil {
		t.Fatalf("generateViewTypes returned error: %v", err)
	}

	content := readViewGenFile(t, resourcePath)
	want := "typeof import('../../src/shared/view-events').CharacterViewEvents['SELECT']"
	if !strings.Contains(content, want) {
		t.Fatalf("expected %q in the output, got:\n%s", want, content)
	}
}

func TestGenerateViewTypes_UndefinedPayload(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	if err := os.MkdirAll(filepath.Join(resourcePath, "ui"), 0755); err != nil {
		t.Fatal(err)
	}

	writeTestFile(t, resourcePath, "src/client/ui.controller.ts", `
@Client.Controller()
export class UiController {
  @Client.OnNet('x:clear')
  onClear(): void {
    this.view.send('ui:clear', undefined)
  }
}
`)

	if _, _, err := rb.generateViewTypes(resourcePath); err != nil {
		t.Fatalf("generateViewTypes returned error: %v", err)
	}

	content := readViewGenFile(t, resourcePath)
	if !strings.Contains(content, "'ui:clear': undefined") {
		t.Fatalf("expected an undefined payload, got:\n%s", content)
	}
}

func TestGenerateViewTypes_NoPayloadArgumentIsUndefined(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	if err := os.MkdirAll(filepath.Join(resourcePath, "ui"), 0755); err != nil {
		t.Fatal(err)
	}

	writeTestFile(t, resourcePath, "src/client/ui.controller.ts", `
@Client.Controller()
export class UiController {
  @Client.OnNet('x:close')
  onClose(): void {
    this.view.send('ui:close')
  }
}
`)

	result, _, err := rb.generateViewTypes(resourcePath)
	if err != nil {
		t.Fatalf("generateViewTypes returned error: %v", err)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("a message with no payload is not an unresolved payload, got %v", result.Warnings)
	}

	content := readViewGenFile(t, resourcePath)
	if !strings.Contains(content, "'ui:close': undefined") {
		t.Fatalf("expected an undefined payload, got:\n%s", content)
	}
}

func TestGenerateViewTypes_UnresolvablePayloadWarns(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	if err := os.MkdirAll(filepath.Join(resourcePath, "ui"), 0755); err != nil {
		t.Fatal(err)
	}

	writeTestFile(t, resourcePath, "src/client/ui.controller.ts", `
@Client.Controller()
export class UiController {
  @Client.OnNet('bank:balanceChanged')
  onBalanceChanged(state: BankState): void {
    this.view.send('bank:view:render', { cash: state.cash, bank: state.bank })
  }
}
`)

	result, _, err := rb.generateViewTypes(resourcePath)
	if err != nil {
		t.Fatalf("generateViewTypes returned error: %v", err)
	}

	if len(result.Warnings) != 1 {
		t.Fatalf("expected one warning for the object literal payload, got %v", result.Warnings)
	}
	if !strings.Contains(result.Warnings[0].Message, "bank:view:render") {
		t.Fatalf("the warning must name the message, got %q", result.Warnings[0].Message)
	}

	// The generated type still exists — the build is never failed, the gap is only reported.
	content := readViewGenFile(t, resourcePath)
	if !strings.Contains(content, "'bank:view:render': unknown") {
		t.Fatalf("expected the payload to fall back to unknown, got:\n%s", content)
	}
}

func TestGenerateViewTypes_IsIdempotent(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	if err := os.MkdirAll(filepath.Join(resourcePath, "ui"), 0755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, resourcePath, "src/client/ui.controller.ts", `
@Client.Controller()
export class UiController {
  @Client.OnView('a:b')
  onB(p: P): void {}
}
`)

	first, _, err := rb.generateViewTypes(resourcePath)
	if err != nil {
		t.Fatalf("generateViewTypes returned error: %v", err)
	}
	if !first.Changed {
		t.Fatal("expected the first run to write the file")
	}

	second, _, err := rb.generateViewTypes(resourcePath)
	if err != nil {
		t.Fatalf("generateViewTypes returned error: %v", err)
	}
	if second.Changed {
		t.Fatal("expected the second run to detect no change")
	}
}

func TestGenerateViewTypes_EmptyWhenNothingFound(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	if err := os.MkdirAll(filepath.Join(resourcePath, "ui"), 0755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, resourcePath, "src/client/plain.ts", `
export class PlainService {
  doThing() {}
}
`)

	if _, _, err := rb.generateViewTypes(resourcePath); err != nil {
		t.Fatalf("generateViewTypes returned error: %v", err)
	}

	content := readViewGenFile(t, resourcePath)
	if !strings.Contains(content, "export type UiCanSend = Record<string, never>") {
		t.Fatalf("expected empty maps, got:\n%s", content)
	}
}
