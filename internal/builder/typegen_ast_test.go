package builder

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// typeScriptInstall returns the repository's `typescript` (5.x) or `typescript-7` devDependency.
func typeScriptInstall(t *testing.T, name string) string {
	t.Helper()
	dir, _ := filepath.Abs(filepath.Join("..", "..", "node_modules", name))
	_, nodeErr := exec.LookPath("node")
	if _, err := os.Stat(filepath.Join(dir, "package.json")); err != nil || nodeErr != nil {
		reason := "typegen tests need node and " + name + " (run `npm install` in the repository root)"
		if os.Getenv("CI") != "" {
			t.Fatal(reason)
		}
		t.Skip(reason)
	}
	return dir
}

// newTypegenBuilder returns a builder for a project at projectPath with the framework fixture installed.
func newTypegenBuilder(t *testing.T, projectPath string) *ResourceBuilder {
	t.Helper()
	return newTypegenBuilderWith(t, projectPath, typeScriptInstall(t, "typescript"), true)
}

func newTypegenBuilderWith(t *testing.T, projectPath string, typeScriptDir string, framework bool) *ResourceBuilder {
	t.Helper()
	if framework {
		target := filepath.Join(projectPath, "node_modules", "@open-core", "framework")
		if err := os.CopyFS(target, os.DirFS(filepath.Join("testdata", "framework"))); err != nil {
			t.Fatal(err)
		}
	}
	rb := NewResourceBuilder(projectPath)
	rb.SetTypegenTypeScript(typeScriptDir)
	t.Cleanup(rb.Cleanup)
	return rb
}

func forEachTypeScript(t *testing.T, run func(t *testing.T, typeScriptDir string)) {
	for _, name := range []string{"typescript", "typescript-7"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			run(t, typeScriptInstall(t, name))
		})
	}
}

const typegenTestTSConfig = `{
  "compilerOptions": {
    "target": "es2022", "module": "esnext", "moduleResolution": "bundler", "strict": true,
    "experimentalDecorators": true, "skipLibCheck": true, "noEmit": true, "types": []
  },
  "include": ["resources/**/*.ts", "resources/**/.opencore/*.ts"]
}`

func TestTypegen_Listeners(t *testing.T) {
	cases := []struct {
		name        string
		files       map[string]string
		want        []string
		wantView    []string
		notWant     []string
		warning     string
		noFramework bool
	}{
		{
			name: "named and aliased imports",
			files: map[string]string{"src/server/bank.controller.ts": `
import { Controller, OnRPC, OnNet, Command, type Player } from '@open-core/framework/server'
import { OnRPC as Rpc } from '@open-core/framework/server'

@Controller()
export class BankController {
  @OnRPC('bank:balance')
  balance(player: Player): number { return 0 }

  @OnNet('bank:deposit')
  deposit(player: Player, amount: number) {}

  @Rpc('bank:history')
  history(player: Player, page: number): string[] { return [] }

  @Command({ command: 'bank', description: 'Opens the bank' })
  open(player: Player) {}
}
`},
			want: []string{
				"'bank:balance': { args: DropFirst<Parameters<T0_BankController['balance']>>; result: Awaited<ReturnType<T0_BankController['balance']>> }",
				"'bank:deposit': DropFirst<Parameters<T0_BankController['deposit']>>",
				"'bank:history': { args: DropFirst<Parameters<T0_BankController['history']>>",
				"'bank': { description: 'Opens the bank' }",
			},
		},
		{
			name: "namespace import and barrel re-exports",
			files: map[string]string{
				"src/server/decorators.ts": `export { OnNet as Listen, OnRPC } from '@open-core/framework/server'`,
				"src/server/index.ts":      `export * from './decorators'`,
				"src/server/shop.controller.ts": `
import * as Framework from '@open-core/framework/server'
import { Listen } from './decorators'
import { OnRPC } from './index'

@Framework.Controller()
export class ShopController {
  @Framework.OnNet('shop:buy')
  buy(player: Framework.Player, item: string) {}

  @Listen('shop:sell')
  sell(player: Framework.Player, item: string, price: number) {}

  @OnRPC('shop:stock')
  stock(player: Framework.Player): number { return 0 }
}
`,
			},
			want: []string{
				"'shop:buy': DropFirst<Parameters<T0_ShopController['buy']>>",
				"'shop:sell': DropFirst<Parameters<T0_ShopController['sell']>>",
				"'shop:stock': { args: DropFirst<Parameters<T0_ShopController['stock']>>",
			},
		},
		{
			name: "named client imports",
			files: map[string]string{
				"ui/index.html": "<!doctype html>",
				"src/client/hud.controller.ts": `
import { Controller, OnNet, OnRPC, OnView as View } from '@open-core/framework/client'

@Controller()
export class HudController {
  @OnNet('hud:setCash')
  setCash(amount: number) {}

  @OnRPC('hud:confirm')
  confirm(message: string): boolean { return true }

  @View('hud:close')
  close(data: { reason: string }) {}
}
`,
			},
			want: []string{
				"'hud:setCash': Parameters<T0_HudController['setCash']>",
				"'hud:confirm': { args: Parameters<T0_HudController['confirm']>",
				"'hud:close': __Payload<Parameters<V0_HudController['close']>>",
			},
		},
		{
			name: "constant names",
			files: map[string]string{
				"src/shared/events.ts": `
export const BankEvents = { deposit: 'bank:deposit' } as const
export const PAYDAY = 'bank:payday'
`,
				"src/server/bank.controller.ts": `
import { Server } from '@open-core/framework/server'
import { BankEvents, PAYDAY } from '../shared/events'

const LOCAL_EVENT = 'bank:local'

@Server.Controller()
export class BankController {
  @Server.OnNet(BankEvents.deposit)
  deposit(player: Server.Player, amount: number) {}

  @Server.OnNet(PAYDAY)
  payday(player: Server.Player) {}

  @Server.OnNet(LOCAL_EVENT)
  local(player: Server.Player) {}
}
`,
				"src/client/chat.controller.ts": `
import { Client } from '@open-core/framework/client'
import { SYSTEM_EVENTS } from '@open-core/framework'

@Client.Controller()
export class ChatController {
  @Client.OnNet(SYSTEM_EVENTS.chat.message)
  onMessage(author: string, text: string) {}
}
`,
			},
			want: []string{
				"'bank:deposit': DropFirst<Parameters<T1_BankController['deposit']>>",
				"'bank:payday': DropFirst<Parameters<T1_BankController['payday']>>",
				"'bank:local': DropFirst<Parameters<T1_BankController['local']>>",
				"'opencore:chat:message': Parameters<T0_ChatController['onMessage']>",
			},
		},
		{
			name: "plain string name warns",
			files: map[string]string{"src/server/bank.controller.ts": `
import { Server } from '@open-core/framework/server'

const dynamicName: string = 'bank:' + 'dynamic'

@Server.Controller()
export class BankController {
  @Server.OnNet(dynamicName)
  dynamic(player: Server.Player) {}
}
`},
			warning: "src/server/bank.controller.ts:8 could not resolve the name dynamicName: its type is `string`",
			notWant: []string{"dynamic"},
		},
		{
			name: "multi-line decorator arguments",
			files: map[string]string{"src/server/pay.controller.ts": `
import { Server } from '@open-core/framework/server'

const TargetAmountArgs = { target: 'number', amount: 'number' }

@Server.Controller()
export class PayController {
  @Server.Command({ command: 'pay', description: 'Pay someone' }, TargetAmountArgs)
  @Server.RequiresState({
    has: ['playing'],
    missing: ['downed'],
    errorMessage: 'You cannot pay now.',
  })
  @Server.Throttle(2, 3000)
  pay(player: Server.Player, target: number, amount: number) {}

  @Server.Command({
    command: 'me',
    description: 'Describe an action',
    usage: '/me <text>',
  }, TargetAmountArgs)
  me(player: Server.Player, text: string) {}
}
`},
			want: []string{
				"'pay': { description: 'Pay someone' }",
				"'me': { description: 'Describe an action'; usage: '/me <text>' }",
			},
		},
		{
			name: "reserved-word method names",
			files: map[string]string{"src/server/scene.controller.ts": `
import { Server } from '@open-core/framework/server'

@Server.Controller()
export class SceneController {
  @Server.Command({ command: 'do', description: 'Describe the scene' })
  do(player: Server.Player, text: string) {}

  @Server.OnNet('scene:delete')
  delete(player: Server.Player, id: number) {}

  @Server.OnRPC('scene:new')
  new(player: Server.Player): number { return 1 }
}
`},
			want: []string{
				"'do': { description: 'Describe the scene' }",
				"'scene:delete': DropFirst<Parameters<T0_SceneController['delete']>>",
				"'scene:new': { args: DropFirst<Parameters<T0_SceneController['new']>>",
			},
		},
		{
			name: "decorators in comments and strings are ignored",
			files: map[string]string{"src/shared/net.contract.ts": `
/** Events the server sends, one per ` + "`@Client.OnNet`" + ` listener; RPCs per ` + "`@Server.OnRPC`" + `. */
export type X = 1
export const docs = "@Server.OnNet('not:real')"
`},
			want:    []string{"export {};"},
			notWant: []string{"not:real"},
		},
		{
			name: "lookalike decorators from other modules are ignored",
			files: map[string]string{
				"src/server/local-decorators.ts": `export function OnNet(name: string): MethodDecorator { return () => {} }`,
				"src/server/audit.controller.ts": `
import { OnNet } from './local-decorators'

export class AuditController {
  @OnNet('audit:log')
  log(entry: string) {}
}
`,
			},
			notWant: []string{"audit:log"},
		},
		{
			name: "WebView payloads come from the checker",
			files: map[string]string{
				"ui/index.html": "<!doctype html>",
				"src/client/clock.controller.ts": `
import { Client, WebView } from '@open-core/framework/client'

export interface ClockSync { hour: number; minute: number }
interface Private { label: string; tags?: string[] }
type Messages = { 'clock:tick': ClockSync; 'clock:label': Private }

@Client.Controller()
export class ClockController {
  private sync: ClockSync = { hour: 0, minute: 0 }
  private hidden: Private = { label: '' }

  @Client.OnNet('clock:notice')
  notice(level: 'info' | 'warn', message: string) {
    WebView.send('notice', { level, message })
    WebView.send('clock', this.sync)
    const local = this.hidden
    WebView.send('label', local)
  }

  send<K extends keyof Messages>(action: K, data: Messages[K]) {
    WebView.send(action, data)
  }

  refresh() {
    this.send('clock:tick', this.sync)
  }
}
`,
			},
			want: []string{
				`'notice': { level: "info" | "warn"; message: string }`,
				`'clock': import('../src/client/clock.controller').ClockSync`,
				`'label': { label: string; tags?: string[] }`,
				`'clock:tick': import('../src/client/clock.controller').ClockSync`,
				`'clock:label': { label: string; tags?: string[] }`,
			},
			wantView: []string{`'clock': import('../../src/client/clock.controller').ClockSync`},
		},
		{
			name:        "imports resolve without the framework installed",
			noFramework: true,
			files: map[string]string{
				"src/server/decorators.ts": `export { OnRPC } from '@open-core/framework/server'`,
				"src/server/bank.controller.ts": `
import { Controller, OnNet as Listen } from '@open-core/framework/server'
import * as Server from '@open-core/framework/server'
import { OnRPC } from './decorators'

@Controller()
export class BankController {
  @Listen('bank:deposit')
  deposit(player: unknown, amount: number) {}

  @Server.OnNet('bank:withdraw')
  withdraw(player: unknown, amount: number) {}

  @OnRPC('bank:balance')
  balance(player: unknown): number { return 0 }
}
`,
			},
			want: []string{"'bank:deposit': DropFirst<", "'bank:withdraw': DropFirst<", "'bank:balance': { args: DropFirst<"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			forEachTypeScript(t, func(t *testing.T, typeScriptDir string) {
				root := t.TempDir()
				writeTestFile(t, root, "tsconfig.json", typegenTestTSConfig)
				resource := filepath.Join(root, "resources", "demo")
				for rel, content := range tc.files {
					writeTestFile(t, resource, rel, content)
				}
				rb := newTypegenBuilderWith(t, root, typeScriptDir, !tc.noFramework)

				result, err := rb.generateTypes(resource, TypegenOptions{Strict: true})
				if err != nil {
					t.Fatalf("generateTypes returned error: %v", err)
				}
				if _, _, err := rb.generateViewTypes(resource); err != nil {
					t.Fatalf("generateViewTypes returned error: %v", err)
				}

				var warnings []string
				for _, w := range result.Warnings {
					warnings = append(warnings, w.String())
				}
				if tc.warning == "" && len(warnings) > 0 || tc.warning != "" && (len(warnings) != 1 || !strings.HasPrefix(warnings[0], tc.warning)) {
					t.Fatalf("unexpected warnings %q", warnings)
				}

				content := readGenFile(t, resource)
				for _, want := range tc.want {
					if !strings.Contains(content, want) {
						t.Fatalf("expected %q in:\n%s", want, content)
					}
				}
				for _, notWant := range tc.notWant {
					if strings.Contains(content, notWant) {
						t.Fatalf("unexpected %q in:\n%s", notWant, content)
					}
				}
				for _, want := range tc.wantView {
					if view := readViewGenFile(t, resource); !strings.Contains(view, want) {
						t.Fatalf("expected %q in the view file:\n%s", want, view)
					}
				}

				if !tc.noFramework {
					tsc := exec.Command("node", filepath.Join(typeScriptDir, "bin", "tsc"), "-p", filepath.Join(root, "tsconfig.json"))
					if output, err := tsc.CombinedOutput(); err != nil {
						t.Fatalf("the generated types do not compile: %v\n%s", err, output)
					}
				}
			})
		})
	}
}

func TestTypegen_NamespaceLiteralFormIsUnchanged(t *testing.T) {
	forEachTypeScript(t, func(t *testing.T, typeScriptDir string) {
		root := t.TempDir()
		resource := filepath.Join(root, "resource")
		fixture := filepath.Join("testdata", "typegen-snapshot")
		if err := os.CopyFS(resource, os.DirFS(filepath.Join(fixture, "resource"))); err != nil {
			t.Fatal(err)
		}
		rb := newTypegenBuilderWith(t, root, typeScriptDir, true)

		if _, err := rb.generateTypes(resource, TypegenOptions{Strict: true}); err != nil {
			t.Fatal(err)
		}
		if _, _, err := rb.generateViewTypes(resource); err != nil {
			t.Fatal(err)
		}
		for generated, golden := range map[string]string{
			filepath.Join(resource, ".opencore", typegenFileName):       "resource.opencore.gen.ts.golden",
			filepath.Join(resource, "ui", ".opencore", typegenFileName): "ui.opencore.gen.ts.golden",
		} {
			want, _ := os.ReadFile(filepath.Join(fixture, golden))
			got, _ := os.ReadFile(generated)
			if string(got) != string(want) {
				t.Fatalf("%s differs from %s\n--- got\n%s\n--- want\n%s", generated, golden, got, want)
			}
		}
	})
}

func TestTypegen_ReusesTheAnalysisUntilASourceChanges(t *testing.T) {
	resource := t.TempDir()
	rb := newTypegenBuilder(t, resource)
	controller := `
@Server.Controller()
export class BankController {
  @Server.OnNet('bank:%s')
  handle(player: unknown) {}
}
`
	writeTestFile(t, resource, "src/server/bank.controller.ts", strings.Replace(controller, "%s", "deposit", 1))

	first, _ := rb.analyzeResource(resource)
	second, _ := rb.analyzeResource(resource)
	if first == nil || first != second {
		t.Fatal("expected an unchanged resource to reuse its analysis")
	}

	writeTestFile(t, resource, "src/server/bank.controller.ts", strings.Replace(controller, "%s", "withdraw", 1))
	if _, err := rb.generateTypes(resource, TypegenOptions{}); err != nil {
		t.Fatal(err)
	}
	if content := readGenFile(t, resource); !strings.Contains(content, "'bank:withdraw'") {
		t.Fatalf("expected the edit to be picked up, got:\n%s", content)
	}
}

// The analyzer's stdout is a pipe, which Node writes asynchronously: the response must survive past 64 KB.
func TestTypegen_AnalyzerResponseLargerThanAPipeBuffer(t *testing.T) {
	resource := t.TempDir()
	rb := newTypegenBuilder(t, resource)
	const controllers = 400
	for i := range controllers {
		writeTestFile(t, resource, fmt.Sprintf("src/server/c%03d.controller.ts", i), fmt.Sprintf(`
@Server.Controller()
export class C%03dController {
  @Server.OnNet('c%03d:event')
  handle(player: unknown) {}
}
`, i, i))
	}

	var files []string
	for i := range controllers {
		file, _ := filepath.Abs(filepath.Join(resource, fmt.Sprintf("src/server/c%03d.controller.ts", i)))
		files = append(files, file)
	}
	response, err := rb.runAnalyzer(files, filepath.Join(resource, typecheckConfigFile))
	if err != nil {
		t.Fatal(err)
	}
	if encoded, _ := json.Marshal(response); len(encoded) <= 64*1024 {
		t.Fatalf("fixture too small to cross the pipe buffer: %d bytes", len(encoded))
	}
	for i, file := range files {
		handlers := response.Files[file].Handlers
		if len(handlers) != 1 || handlers[0].Event != fmt.Sprintf("c%03d:event", i) {
			t.Fatalf("%s: unexpected handlers %+v", file, handlers)
		}
	}
}
