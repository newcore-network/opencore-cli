package builder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readGenFile(t *testing.T, resourcePath string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(resourcePath, ".opencore", typegenFileName))
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}
	return string(content)
}

func TestGenerateTypes_ServerNetEventDropsPlayerParam(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	writeTestFile(t, resourcePath, "src/server/bank.controller.ts", `
import { Server } from '@open-core/framework/server'

@Server.Controller()
export class BankController {
  @Server.OnNet('bank:deposit', DepositSchema)
  async deposit(player: Server.Player, amount: number) {}
}
`)

	result, err := rb.generateTypes(resourcePath, TypegenOptions{})
	if err != nil {
		t.Fatalf("generateTypes returned error: %v", err)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", result.Warnings)
	}
	if !result.Changed {
		t.Fatal("expected the first generation to report a change")
	}

	content := readGenFile(t, resourcePath)

	// The Player parameter must be stripped from the emitted payload tuple.
	if !strings.Contains(content, "DropFirst<Parameters<") {
		t.Fatalf("expected DropFirst wrapper for a server handler, got:\n%s", content)
	}
	if !strings.Contains(content, "'bank:deposit'") {
		t.Fatalf("expected the event name in the output, got:\n%s", content)
	}
	if !strings.Contains(content, "'deposit'") {
		t.Fatalf("expected the method name in the output, got:\n%s", content)
	}
	if !strings.Contains(content, "serverEvents: GenServerEvents") {
		t.Fatalf("expected the Register augmentation, got:\n%s", content)
	}
	// The import must be relative to .opencore/, not to the resource root.
	if !strings.Contains(content, "'../src/server/bank.controller'") {
		t.Fatalf("expected an import path relative to .opencore, got:\n%s", content)
	}
}

func TestGenerateTypes_AugmentsTheDeclaringModuleNotThePackageRoot(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	writeTestFile(t, resourcePath, "src/server/bank.controller.ts", `
@Server.Controller()
export class BankController {
  @Server.OnNet('bank:deposit')
  deposit(player: Server.Player, amount: number) {}
}
`)

	if _, err := rb.generateTypes(resourcePath, TypegenOptions{}); err != nil {
		t.Fatalf("generateTypes returned error: %v", err)
	}

	content := readGenFile(t, resourcePath)

	if !strings.Contains(content, "declare module '@open-core/framework/register'") {
		t.Fatalf("expected the augmentation to target the declaring module, got:\n%s", content)
	}
	if strings.Contains(content, "declare module '@open-core/framework'") {
		t.Fatalf("augmenting the package root silently disables all narrowing, got:\n%s", content)
	}
	// The helper import must come from the same module for the augmentation to resolve.
	if !strings.Contains(content, "import type { DropFirst } from '@open-core/framework/register'") {
		t.Fatalf("expected DropFirst to be imported from the register module, got:\n%s", content)
	}
}

func TestGenerateTypes_FilesMapsUnderAPerResourceKey(t *testing.T) {
	projectRoot := t.TempDir()
	rb := NewResourceBuilder(projectRoot)

	resourcePath := filepath.Join(projectRoot, "resources", "bank")
	writeTestFile(t, resourcePath, "src/server/bank.controller.ts", `
@Server.Controller()
export class BankController {
  @Server.OnNet('bank:deposit')
  deposit(player: Server.Player, amount: number) {}
}
`)

	if _, err := rb.generateTypes(resourcePath, TypegenOptions{}); err != nil {
		t.Fatalf("generateTypes returned error: %v", err)
	}

	content := readGenFile(t, resourcePath)

	if !strings.Contains(content, "'resource:resources/bank': {") {
		t.Fatalf("expected the maps to be filed under a per-resource key, got:\n%s", content)
	}
	if strings.Contains(content, "\n    serverEvents:") {
		t.Fatalf("a map declared directly on Register collides with every other resource, got:\n%s", content)
	}
	if !strings.Contains(content, "      serverEvents: GenServerEvents") {
		t.Fatalf("expected serverEvents inside the resource entry, got:\n%s", content)
	}
}

func TestResourceRegisterKeyIsProjectRelative(t *testing.T) {
	if got := resourceRegisterKey("/proj", "/proj/resources/bank"); got != "resource:resources/bank" {
		t.Fatalf("expected a project-relative key, got %q", got)
	}
	// Two resources sharing a basename across groups must not collide.
	a := resourceRegisterKey("/proj", "/proj/resources/chat")
	b := resourceRegisterKey("/proj", "/proj/standalones/chat")
	if a == b {
		t.Fatalf("same-named resources in different groups collide: %q", a)
	}
	// A path outside the project falls back to something usable rather than "../..".
	if got := resourceRegisterKey("/proj", "/elsewhere/bank"); strings.Contains(got, "..") {
		t.Fatalf("expected no parent traversal in the key, got %q", got)
	}
}

func TestGenerateTypes_AttributesHandlersToTheirOwnClass(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	writeTestFile(t, resourcePath, "src/server/pair.controller.ts", `
@Server.Controller()
export class FirstController {
  @Server.OnNet('first:ping')
  onPing(player: Server.Player, a: number) {}
}

@Server.Controller()
export class SecondController {
  @Server.OnNet('second:pong')
  onPong(player: Server.Player, b: string) {}
}
`)

	if _, err := rb.generateTypes(resourcePath, TypegenOptions{}); err != nil {
		t.Fatalf("generateTypes returned error: %v", err)
	}

	content := readGenFile(t, resourcePath)

	if !strings.Contains(content, "'first:ping': DropFirst<Parameters<T0_FirstController['onPing']>>") {
		t.Fatalf("expected the first handler on its own class, got:\n%s", content)
	}
	if !strings.Contains(content, "'second:pong': DropFirst<Parameters<T1_SecondController['onPong']>>") {
		t.Fatalf("expected the second handler on SecondController, got:\n%s", content)
	}
	if strings.Contains(content, "T0_FirstController['onPong']") {
		t.Fatalf("a handler was attributed to the wrong class, got:\n%s", content)
	}
}

// A decorator left behind in a comment used to be scanned as real and bound to whatever method
// followed it, which both invents an event and retypes a real one against the wrong handler.
func TestGenerateTypes_IgnoresCommentedDecorators(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	writeTestFile(t, resourcePath, "src/server/bank.controller.ts", `
@Server.Controller()
export class BankController {
  // @Server.OnNet('bank:removed')
  /* @Server.OnNet('bank:alsoRemoved') */
  @Server.OnNet('bank:deposit')
  deposit(player: Server.Player, amount: number) {}

  /**
   * @Server.Command('docs-example')
   */
  @Server.OnNet('bank:withdraw')
  withdraw(player: Server.Player, amount: number) {}
}
`)

	result, err := rb.generateTypes(resourcePath, TypegenOptions{})
	if err != nil {
		t.Fatalf("generateTypes returned error: %v", err)
	}

	content := readGenFile(t, resourcePath)

	for _, ghost := range []string{"bank:removed", "bank:alsoRemoved", "docs-example"} {
		if strings.Contains(content, ghost) {
			t.Fatalf("a commented-out decorator was scanned (%s), got:\n%s", ghost, content)
		}
	}
	if !strings.Contains(content, "'bank:deposit'") || !strings.Contains(content, "'bank:withdraw'") {
		t.Fatalf("expected both real handlers, got:\n%s", content)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", result.Warnings)
	}
}

func TestBlankCommentsPreservesOffsetsAndStrings(t *testing.T) {
	source := "const a = 'http://not-a-comment'\n// gone\nconst b = 1 /* gone */\n"
	blanked := blankComments(source)

	if len(blanked) != len(source) {
		t.Fatalf("length changed: %d vs %d", len(blanked), len(source))
	}
	if strings.Count(blanked, "\n") != strings.Count(source, "\n") {
		t.Fatal("newlines must survive so line numbers do not shift")
	}
	if !strings.Contains(blanked, "'http://not-a-comment'") {
		t.Fatalf("a `//` inside a string is not a comment, got:\n%q", blanked)
	}
	if strings.Contains(blanked, "gone") {
		t.Fatalf("comment contents survived, got:\n%q", blanked)
	}
	if !strings.Contains(blanked, "const b = 1") {
		t.Fatalf("code before a block comment was damaged, got:\n%q", blanked)
	}
}

func TestGenerateTypesResolvesImportedAliasDefaultAndNamespace(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")
	writeTestFile(t, resourcePath, "src/events.ts", `
export const Named = { Event: 'named' } as const
const DefaultEvents = { Event: 'default' } as const
export default DefaultEvents
`)
	writeTestFile(t, resourcePath, "src/server/controller.ts", `
import { Named as Alias } from '../events'
import Defaults from '../events'
import * as Events from '../events'
export class Controller {
  @Server.OnNet(Alias.Event) named(player: unknown) {}
  @Server.OnNet(Defaults.Event) defaults(player: unknown) {}
  @Server.OnNet(Events.Named.Event) namespace(player: unknown) {}
}
`)
	if _, err := rb.generateTypes(resourcePath, TypegenOptions{}); err != nil {
		t.Fatal(err)
	}
	content := readGenFile(t, resourcePath)
	for _, expected := range []string{
		"typeof import('../src/events').Named['Event']",
		"typeof import('../src/events').default['Event']",
		"/* Events.Named.Event */",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("missing %q in generated types:\n%s", expected, content)
		}
	}
}

func TestGenerateTypes_DuplicateEventNameIsReportedAndStable(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	writeTestFile(t, resourcePath, "src/server/a.controller.ts", `
@Server.Controller()
export class AController {
  @Server.OnNet('shared:name')
  handleA(player: Server.Player, a: number) {}
}
`)
	writeTestFile(t, resourcePath, "src/server/b.controller.ts", `
@Server.Controller()
export class BController {
  @Server.OnNet('shared:name')
  handleB(player: Server.Player, b: string) {}
}
`)

	result, err := rb.generateTypes(resourcePath, TypegenOptions{})
	if err != nil {
		t.Fatalf("generateTypes returned error: %v", err)
	}

	if len(result.Warnings) != 1 {
		t.Fatalf("expected the clash to be reported once, got %v", result.Warnings)
	}
	if !strings.Contains(result.Warnings[0].Message, "shared:name") {
		t.Fatalf("the warning must name the event, got %q", result.Warnings[0].Message)
	}

	first := readGenFile(t, resourcePath)

	// Regenerating the same sources must produce the same winner, byte for byte.
	for i := 0; i < 5; i++ {
		again, regenErr := rb.generateTypes(resourcePath, TypegenOptions{})
		if regenErr != nil {
			t.Fatalf("regeneration failed: %v", regenErr)
		}
		if again.Changed {
			t.Fatal("regeneration rewrote the file without a source change")
		}
	}
	if readGenFile(t, resourcePath) != first {
		t.Fatal("the generated file is not stable across runs")
	}
}

func TestGenerateTypes_ClientNetEventKeepsAllParams(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	writeTestFile(t, resourcePath, "src/client/hud.controller.ts", `
import { Client } from '@open-core/framework/client'

@Client.Controller()
export class HudController {
  @Client.OnNet('hud:setCash')
  setCash(amount: number) {}
}
`)

	if _, err := rb.generateTypes(resourcePath, TypegenOptions{}); err != nil {
		t.Fatalf("generateTypes returned error: %v", err)
	}

	content := readGenFile(t, resourcePath)

	if strings.Contains(content, "DropFirst") {
		t.Fatalf("client handlers must not drop a parameter, got:\n%s", content)
	}
	if !strings.Contains(content, "clientEvents: GenClientEvents") {
		t.Fatalf("expected clientEvents registration, got:\n%s", content)
	}
}

func TestGenerateTypes_RpcIncludesArgsAndResult(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	writeTestFile(t, resourcePath, "src/server/bank.controller.ts", `
@Server.Controller()
export class BankController {
  @Server.OnRPC('bank:getBalance')
  async getBalance(player: Server.Player, accountId: string): Promise<number> {
    return 0
  }
}
`)

	if _, err := rb.generateTypes(resourcePath, TypegenOptions{}); err != nil {
		t.Fatalf("generateTypes returned error: %v", err)
	}

	content := readGenFile(t, resourcePath)

	if !strings.Contains(content, "args:") || !strings.Contains(content, "result:") {
		t.Fatalf("expected an RPC entry with args and result, got:\n%s", content)
	}
	if !strings.Contains(content, "Awaited<ReturnType<") {
		t.Fatalf("expected the result to unwrap the Promise, got:\n%s", content)
	}
}

func TestGenerateTypes_CommandConfigObjectForm(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	writeTestFile(t, resourcePath, "src/server/bank.controller.ts", `
@Server.Controller()
export class BankController {
  @Server.Command({ command: 'withdraw', description: 'Take money out', usage: '/withdraw <n>' })
  withdraw(player: Server.Player) {}

  @Server.Command('help')
  help(player: Server.Player) {}
}
`)

	if _, err := rb.generateTypes(resourcePath, TypegenOptions{}); err != nil {
		t.Fatalf("generateTypes returned error: %v", err)
	}

	content := readGenFile(t, resourcePath)

	for _, want := range []string{"'withdraw'", "'help'", "Take money out", "/withdraw <n>", "commands: GenCommands"} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected %q in the output, got:\n%s", want, content)
		}
	}
}

func TestGenerateTypes_MultilineDecorator(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	writeTestFile(t, resourcePath, "src/server/multi.controller.ts", `
@Server.Controller()
export class MultiController {
  @Server.OnNet(
    'multi:line',
    SomeSchema,
  )
  handle(player: Server.Player, value: number) {}
}
`)

	result, err := rb.generateTypes(resourcePath, TypegenOptions{})
	if err != nil {
		t.Fatalf("generateTypes returned error: %v", err)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings for a multiline decorator, got %v", result.Warnings)
	}

	content := readGenFile(t, resourcePath)
	if !strings.Contains(content, "'multi:line'") || !strings.Contains(content, "'handle'") {
		t.Fatalf("expected the multiline decorator to be extracted, got:\n%s", content)
	}
}

func TestGenerateTypes_NonLiteralEventNameIsSkippedWithWarning(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	writeTestFile(t, resourcePath, "src/server/dyn.controller.ts", `
@Server.Controller()
export class DynController {
  @Server.OnNet(EVENTS.SOMETHING)
  handle(player: Server.Player) {}
}
`)

	result, err := rb.generateTypes(resourcePath, TypegenOptions{})
	if err != nil {
		t.Fatalf("generateTypes returned error: %v", err)
	}

	if len(result.Entries) != 0 {
		t.Fatalf("expected the non-literal entry to be skipped, got %v", result.Entries)
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("expected exactly 1 warning, got %v", result.Warnings)
	}
	if !strings.Contains(result.Warnings[0].Message, "could not resolve the event name") {
		t.Fatalf("unexpected warning message: %s", result.Warnings[0].Message)
	}

	// A skipped entry must never fail the build; the file is still written.
	if content := readGenFile(t, resourcePath); !strings.Contains(content, "export {};") {
		t.Fatalf("expected an empty generated file, got:\n%s", content)
	}
}

func TestGenerateTypes_ResolvesImportedConstReference(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	writeTestFile(t, resourcePath, "src/shared/character-events.ts", `
export const CharacterNetEvents = {
  UI_OPEN: 'character:ui:open',
  UI_CLOSE: 'character:ui:close',
} as const
`)
	writeTestFile(t, resourcePath, "src/client/controllers/character-ui.controller.ts", `
import { Client } from '@open-core/framework/client'
import { CharacterNetEvents } from '../../shared/character-events'

@Client.Controller()
export class CharacterUiController {
  @Client.OnNet(CharacterNetEvents.UI_OPEN)
  async onOpen(state: CharacterUiState): Promise<void> {}

  @Client.OnNet(CharacterNetEvents.UI_CLOSE)
  onClose(): void {}
}
`)

	result, err := rb.generateTypes(resourcePath, TypegenOptions{})
	if err != nil {
		t.Fatalf("generateTypes returned error: %v", err)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected constant references to resolve, got warnings: %v", result.Warnings)
	}
	if len(result.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result.Entries))
	}

	content := readGenFile(t, resourcePath)

	// The import path must be rewritten relative to .opencore/, not left as written in the
	// controller (which was relative to the controller's own directory).
	want := "typeof import('../src/shared/character-events').CharacterNetEvents['UI_OPEN']"
	if !strings.Contains(content, want) {
		t.Fatalf("expected %q in the output, got:\n%s", want, content)
	}
	if !strings.Contains(content, "__Entry<") {
		t.Fatalf("expected computed keys to use the __Entry helper, got:\n%s", content)
	}
	if !strings.Contains(content, "type __Entry<K extends PropertyKey, V>") {
		t.Fatalf("expected the __Entry helper to be declared, got:\n%s", content)
	}
}

func TestGenerateTypes_ResolvesBarePackageConstReference(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	writeTestFile(t, resourcePath, "src/client/ui.controller.ts", `
import { Client } from '@open-core/framework/client'
import { SystemUiNetEvents } from '@glint/system-ui'

@Client.Controller()
export class UiController {
  @Client.OnNet(SystemUiNetEvents.NOTIFICATION_SHOW)
  onShow(options: NotificationOptions): void {}
}
`)

	if _, err := rb.generateTypes(resourcePath, TypegenOptions{}); err != nil {
		t.Fatalf("generateTypes returned error: %v", err)
	}

	content := readGenFile(t, resourcePath)
	// Bare specifiers resolve identically from any directory and must be left untouched.
	want := "typeof import('@glint/system-ui').SystemUiNetEvents['NOTIFICATION_SHOW']"
	if !strings.Contains(content, want) {
		t.Fatalf("expected %q in the output, got:\n%s", want, content)
	}
}

func TestGenerateTypes_InlinesFileLocalStringConst(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	writeTestFile(t, resourcePath, "src/client/local.controller.ts", `
import { Client } from '@open-core/framework/client'

const CHAT_LOCK_EVENT = 'glint:chat:setBlocked'

@Client.Controller()
export class LocalController {
  @Client.OnNet(CHAT_LOCK_EVENT)
  onLock(blocked: boolean): void {}
}
`)

	result, err := rb.generateTypes(resourcePath, TypegenOptions{})
	if err != nil {
		t.Fatalf("generateTypes returned error: %v", err)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected the local const to resolve, got warnings: %v", result.Warnings)
	}

	content := readGenFile(t, resourcePath)
	// A non-exported local constant cannot be imported, so its value is inlined instead.
	if !strings.Contains(content, "'glint:chat:setBlocked'") {
		t.Fatalf("expected the local const to be inlined, got:\n%s", content)
	}
}

func TestGenerateTypes_ResolvesAliasedImport(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	writeTestFile(t, resourcePath, "src/shared/events.ts", `
export const Events = { PING: 'x:ping' } as const
`)
	writeTestFile(t, resourcePath, "src/client/alias.controller.ts", `
import { Events as NetEvents } from '../shared/events'

@Client.Controller()
export class AliasController {
  @Client.OnNet(NetEvents.PING)
  onPing(): void {}
}
`)

	result, err := rb.generateTypes(resourcePath, TypegenOptions{})
	if err != nil {
		t.Fatalf("generateTypes returned error: %v", err)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected the aliased import to resolve, got warnings: %v", result.Warnings)
	}

	content := readGenFile(t, resourcePath)
	// Type queries address exports of the imported module, not local aliases.
	if !strings.Contains(content, ".Events['PING']") || strings.Contains(content, ".NetEvents['PING']") {
		t.Fatalf("expected the original exported binding in the reference, got:\n%s", content)
	}
}

func TestGenerateTypes_NoDecoratorsProducesEmptyModule(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	writeTestFile(t, resourcePath, "src/server/plain.ts", `
export class PlainService {
  doThing() {}
}
`)

	if _, err := rb.generateTypes(resourcePath, TypegenOptions{}); err != nil {
		t.Fatalf("generateTypes returned error: %v", err)
	}

	content := readGenFile(t, resourcePath)
	if !strings.Contains(content, "export {};") {
		t.Fatalf("expected an empty module, got:\n%s", content)
	}
	if strings.Contains(content, "declare module") {
		t.Fatalf("an empty result must not augment Register, got:\n%s", content)
	}
}

func TestGenerateTypes_IsIdempotent(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	writeTestFile(t, resourcePath, "src/server/bank.controller.ts", `
@Server.Controller()
export class BankController {
  @Server.OnNet('bank:deposit')
  deposit(player: Server.Player, amount: number) {}
}
`)

	first, err := rb.generateTypes(resourcePath, TypegenOptions{})
	if err != nil {
		t.Fatalf("first generateTypes returned error: %v", err)
	}
	if !first.Changed {
		t.Fatal("expected the first run to write the file")
	}

	firstContent := readGenFile(t, resourcePath)

	second, err := rb.generateTypes(resourcePath, TypegenOptions{})
	if err != nil {
		t.Fatalf("second generateTypes returned error: %v", err)
	}
	// Re-writing an identical file would feed the dev watcher a spurious change event.
	if second.Changed {
		t.Fatal("expected the second run to detect no change")
	}
	if readGenFile(t, resourcePath) != firstContent {
		t.Fatal("expected identical output across runs")
	}
}

func TestGenerateTypes_StrictModeMarker(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	writeTestFile(t, resourcePath, "src/server/bank.controller.ts", `
@Server.Controller()
export class BankController {
  @Server.OnNet('bank:deposit')
  deposit(player: Server.Player) {}
}
`)

	if _, err := rb.generateTypes(resourcePath, TypegenOptions{Strict: true}); err != nil {
		t.Fatalf("generateTypes returned error: %v", err)
	}

	if content := readGenFile(t, resourcePath); !strings.Contains(content, "strict: true") {
		t.Fatalf("expected the strict marker, got:\n%s", content)
	}
}

func TestGenerateTypes_SkipsViewsAndNodeModules(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	writeTestFile(t, resourcePath, "src/server/real.controller.ts", `
@Server.Controller()
export class RealController {
  @Server.OnNet('real:event')
  handle(player: Server.Player) {}
}
`)
	writeTestFile(t, resourcePath, "ui/fake.controller.ts", `
@Server.Controller()
export class UiController {
  @Server.OnNet('ui:should-not-appear')
  handle(player: Server.Player) {}
}
`)
	writeTestFile(t, resourcePath, "node_modules/pkg/dep.controller.ts", `
@Server.Controller()
export class DepController {
  @Server.OnNet('dep:should-not-appear')
  handle(player: Server.Player) {}
}
`)

	if _, err := rb.generateTypes(resourcePath, TypegenOptions{}); err != nil {
		t.Fatalf("generateTypes returned error: %v", err)
	}

	content := readGenFile(t, resourcePath)
	if !strings.Contains(content, "'real:event'") {
		t.Fatalf("expected the real event, got:\n%s", content)
	}
	if strings.Contains(content, "should-not-appear") {
		t.Fatalf("ui/ and node_modules/ must be skipped, got:\n%s", content)
	}
}

func TestGenerateTypes_DuplicateClassNamesGetDistinctAliases(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	writeTestFile(t, resourcePath, "src/server/a/main.controller.ts", `
@Server.Controller()
export class MainController {
  @Server.OnNet('a:event')
  handle(player: Server.Player) {}
}
`)
	writeTestFile(t, resourcePath, "src/server/b/main.controller.ts", `
@Server.Controller()
export class MainController {
  @Server.OnNet('b:event')
  handle(player: Server.Player) {}
}
`)

	if _, err := rb.generateTypes(resourcePath, TypegenOptions{}); err != nil {
		t.Fatalf("generateTypes returned error: %v", err)
	}

	content := readGenFile(t, resourcePath)
	if !strings.Contains(content, "T0_MainController") || !strings.Contains(content, "T1_MainController") {
		t.Fatalf("expected distinct aliases for same-named classes, got:\n%s", content)
	}
}

func TestRemoveGeneratedTypes(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	writeTestFile(t, resourcePath, "src/server/bank.controller.ts", `
@Server.Controller()
export class BankController {
  @Server.OnNet('bank:deposit')
  deposit(player: Server.Player) {}
}
`)

	if _, err := rb.generateTypes(resourcePath, TypegenOptions{}); err != nil {
		t.Fatalf("generateTypes returned error: %v", err)
	}
	if err := removeGeneratedTypes(resourcePath); err != nil {
		t.Fatalf("removeGeneratedTypes returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(resourcePath, ".opencore", typegenFileName)); !os.IsNotExist(err) {
		t.Fatal("expected the generated file to be removed")
	}
	// Removing a file that is already gone must be a no-op.
	if err := removeGeneratedTypes(resourcePath); err != nil {
		t.Fatalf("expected removing a missing file to be a no-op, got: %v", err)
	}
}
