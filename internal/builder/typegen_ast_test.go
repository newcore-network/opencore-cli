package builder

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/newcore-network/opencore-cli/internal/config"
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
			name: "library listeners are filed per library",
			files: map[string]string{
				"src/server/events.ts": `export const RP = 'rp' as const
export const SPAWNED = 'character:spawned'`,
				"src/server/characters.controller.ts": `
import { Controller, OnLibraryEvent } from '@open-core/framework/server'
import { RP, SPAWNED } from './events'

export interface Spawned { accountId: string; firstTime: boolean }

@Controller()
export class CharactersController {
  @OnLibraryEvent(RP, SPAWNED)
  onSpawned(payload: Spawned) {}

  @OnLibraryEvent('bank', 'bank:opened')
  onOpened() {}
}
`,
				// A second listener of the same event: not a clash, and ignoring the payload
				// must not untype the event.
				"src/server/audit.controller.ts": `
import { Server } from '@open-core/framework/server'

@Server.Controller()
export class AuditController {
  @Server.OnLibraryEvent('rp', 'character:spawned')
  log() {}

  @Server.OnLibraryEvent('rp', 'character:left')
  onLeft(payload: { accountId: string }, meta: unknown) {}
}
`,
				"src/client/hud.controller.ts": `
import { Client } from '@open-core/framework/client'

@Client.Controller()
export class HudController {
  @Client.OnLibraryEvent('hud', 'hud:shown')
  onShown(visible: boolean) {}
}
`,
			},
			want: []string{
				"type __Payload<P extends unknown[]>",
				"export type GenServerLibraryEvents = {\n" +
					"  'bank': {\n" +
					// Its only listener ignores the payload, which then constrains nothing.
					"    'bank:opened': unknown\n" +
					"  }\n" +
					"  'rp': {\n" +
					"    'character:left': __Payload<Parameters<T1_AuditController['onLeft']>>\n" +
					"    'character:spawned': __Payload<Parameters<T2_CharactersController['onSpawned']>>\n" +
					"  }\n" +
					"}",
				"export type GenClientLibraryEvents = {\n" +
					"  'hud': {\n" +
					"    'hud:shown': __Payload<Parameters<T0_HudController['onShown']>>\n" +
					"  }\n" +
					"}",
				"serverLibraryEvents: GenServerLibraryEvents",
				"clientLibraryEvents: GenClientLibraryEvents",
			},
			notWant: []string{"['log']"},
		},
		{
			name: "library listener with an optional, defaulted or rest payload keeps its type",
			files: map[string]string{
				"src/server/rp.controller.ts": `
import { Server } from '@open-core/framework/server'

export interface Spawned { id: string }

@Server.Controller()
export class RpController {
  @Server.OnLibraryEvent('rp', 'optional')
  onOptional(payload?: Spawned) {}

  @Server.OnLibraryEvent('rp', 'defaulted')
  onDefaulted(payload = { id: 'x' }) {}

  @Server.OnLibraryEvent('rp', 'rest')
  onRest(...ids: string[]) {}

  @Server.OnLibraryEvent('rp', 'receiver')
  onReceiver(this: RpController, count: number) {}
}
`,
				// Compiled with the generated file: each payload must accept a real value.
				"src/server/check.ts": `
import type { GenServerLibraryEvents as E } from '../../.opencore/opencore.gen'
export const optional: E['rp']['optional'] = { id: 'a' }
export const defaulted: E['rp']['defaulted'] = { id: 'a' }
export const rest: E['rp']['rest'] = 'a'
export const receiver: E['rp']['receiver'] = 1
`,
			},
			want: []string{
				"type __Payload<P extends unknown[]> = P extends [] ? undefined : Required<P>[0]",
				"    'optional': __Payload<Parameters<T0_RpController['onOptional']>>\n",
				"    'receiver': __Payload<Parameters<T0_RpController['onReceiver']>>\n",
			},
		},
		{
			// The payload reaches every listener, so it must satisfy each of their parameters.
			name: "library listeners of one event intersect their payloads",
			files: map[string]string{
				"src/server/a.controller.ts": `
import { Server } from '@open-core/framework/server'

@Server.Controller()
export class AController {
  @Server.OnLibraryEvent('rp', 'joined')
  onJoined(payload: { id: string }) {}

  @Server.OnLibraryEvent('rp', 'joined')
  ignored() {}
}
`,
				"src/server/b.controller.ts": `
import { Server } from '@open-core/framework/server'

@Server.Controller()
export class BController {
  @Server.OnLibraryEvent('rp', 'joined')
  onJoined(payload: { name: string }) {}
}
`,
				"src/server/check.ts": `
import type { GenServerLibraryEvents as E } from '../../.opencore/opencore.gen'
export const joined: E['rp']['joined'] = { id: 'a', name: 'b' }
`,
			},
			want: []string{
				"    'joined': __Payload<Parameters<T0_AController['onJoined']>> & __Payload<Parameters<T1_BController['onJoined']>>\n",
			},
			notWant: []string{"['ignored']"},
		},
		{
			name: "library emits type the events nobody listens to",
			files: map[string]string{
				"src/server/rp.library.ts": `
import { createServerLibrary } from '@open-core/framework/server'
export const rpLibrary = createServerLibrary('rp')
`,
				"src/server/market.service.ts": `
import { rpLibrary } from './rp.library'

export type Reason = 'sold' | 'owner-deleted'

export class MarketService {
  release(propertyId: string, reason: Reason) {
    rpLibrary.emit('property:released', { propertyId, reason })
  }

  jail(characterId: string) {
    rpLibrary.emit('police:jailed', { characterId, releaseAt: 1 })
  }

  jailById(characterId: number) {
    rpLibrary.emit('police:jailed', { characterId, releaseAt: 2 })
  }

  revive(payload: { characterId: string }) {
    rpLibrary.emit('vitals:revived', payload)
  }

  tick() {
    rpLibrary.emit('market:tick')
  }

  leave(accountId: string) {
    rpLibrary.emit('character:left', { accountId, extra: true })
  }

  notALibrary(emitter: { emit(name: string, payload: unknown): void }) {
    emitter.emit('not:library', 1)
  }
}
`,
				"src/server/session.controller.ts": `
import { Server } from '@open-core/framework/server'

@Server.Controller()
export class SessionController {
  @Server.OnLibraryEvent('rp', 'character:left')
  onLeft(payload: { accountId: string }) {}

  @Server.OnLibraryEvent('rp', 'property:released')
  onReleased() {}
}
`,
				// The generated file cannot import a class that is not exported.
				"src/server/notifier.ts": `
import { rpLibrary } from './rp.library'

interface Fired { at: number }

class Notifier {
  fire(payload: Fired) {
    rpLibrary.emit('notifier:fired', payload)
  }
}

export const notifier = new Notifier()
`,
				"src/client/hud.ts": `
import { createClientLibrary } from '@open-core/framework/client'
const hud = createClientLibrary('hud')
export function show() {
  hud.emit('hud:shown', { visible: true as boolean })
}
`,
			},
			want: []string{
				// A listener types its event, however its emit() calls pass it.
				"    'character:left': __Payload<Parameters<T1_SessionController['onLeft']>>\n",
				"    'market:tick': undefined\n",
				// Different payloads at different calls make a union.
				"    'police:jailed': { characterId: number; releaseAt: number } | { characterId: string; releaseAt: number }\n",
				// A listener ignoring the payload leaves the event to its emit() calls. The order of a
				// literal union is the checker's, which differs between TypeScript versions.
				"    'property:released': { propertyId: string; reason: \"",
				"    'notifier:fired': { at: number }\n",
				"    'vitals:revived': Parameters<T0_MarketService['revive']>[0]\n",
				"export type GenClientLibraryEvents = {\n" +
					"  'hud': {\n" +
					"    'hud:shown': { visible: boolean }\n" +
					"  }\n" +
					"}",
			},
			notWant: []string{"not:library", "extra", "Notifier"},
		},
		{
			// An unused helper fails projects compiled with `noUnusedLocals`.
			name: "library emits alone do not declare the payload helper",
			files: map[string]string{"src/client/hud.ts": `
import { createClientLibrary } from '@open-core/framework/client'
const hud = createClientLibrary('hud')
export function show() {
  hud.emit('hud:shown', { visible: true as boolean })
}
`},
			want:    []string{"    'hud:shown': { visible: boolean }\n"},
			notWant: []string{"__Payload"},
		},
		{
			name: "library emit on a library without a literal name warns",
			files: map[string]string{"src/server/dynamic.ts": `
import { createServerLibrary } from '@open-core/framework/server'

const name: string = 'r' + 'p'
const library = createServerLibrary(name)

export function publish() {
  library.emit('character:spawned', 1)
}
`},
			warning: "src/server/dynamic.ts:8 could not resolve the library name of library of a library emit()",
			notWant: []string{"character:spawned"},
		},
		{
			name: "library listener with a plain string library name warns",
			files: map[string]string{"src/server/rp.controller.ts": `
import { Server } from '@open-core/framework/server'

const library: string = 'r' + 'p'

@Server.Controller()
export class RpController {
  @Server.OnLibraryEvent(library, 'character:spawned')
  onSpawned(payload: number) {}
}
`},
			warning: "src/server/rp.controller.ts:8 could not resolve the name library: its type is `string`",
			notWant: []string{"character:spawned", "LibraryEvents"},
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

func TestTypegen_WarnsWhenAnotherResourceCreatesALibraryWithTheSameName(t *testing.T) {
	forEachTypeScript(t, func(t *testing.T, typeScriptDir string) {
		root := t.TempDir()
		writeTestFile(t, root, "tsconfig.json", typegenTestTSConfig)
		characters := filepath.Join(root, "resources", "characters")
		garage := filepath.Join(root, "resources", "garage")
		session := filepath.Join(root, "resources", "session")

		writeTestFile(t, characters, "src/server/core.library.ts", `
import { createServerLibrary } from '@open-core/framework/server'
export const core = createServerLibrary('core')
`)
		writeTestFile(t, characters, "src/client/core.library.ts", `
import { createClientLibrary } from '@open-core/framework/client'
export const core = createClientLibrary('core')
`)
		writeTestFile(t, garage, "src/server/core.library.ts", `
import { Server } from '@open-core/framework/server'
export const core = Server.createServerLibrary('core')
export const named = (name: string) => Server.createServerLibrary(name)
`)
		// Listening to a library does not create one, so it never collides.
		writeTestFile(t, session, "src/server/session.controller.ts", `
import { Server } from '@open-core/framework/server'
@Server.Controller()
export class SessionController {
  @Server.OnLibraryEvent('core', 'ready')
  onReady(payload: { id: number }) {}
}
`)
		rb := newTypegenBuilderWith(t, root, typeScriptDir, true)

		warningsOf := func(resource string) []string {
			t.Helper()
			result, err := rb.generateTypes(resource, TypegenOptions{})
			if err != nil {
				t.Fatalf("generateTypes returned error: %v", err)
			}
			var warnings []string
			for _, w := range result.Warnings {
				warnings = append(warnings, w.String())
			}
			return warnings
		}

		if warnings := warningsOf(characters); len(warnings) != 0 {
			t.Fatalf("no other resource is generated yet, got %q", warnings)
		}
		if content := readGenFile(t, characters); !strings.Contains(content,
			`// opencore:libraries [{"kind":"clientLibraryEvents","name":"core"},{"kind":"serverLibraryEvents","name":"core"}]`) {
			t.Fatalf("expected the libraries the resource creates in its header, got:\n%s", content)
		}
		if warnings := warningsOf(session); len(warnings) != 0 {
			t.Fatalf("a resource only listening to a library collides with nothing, got %q", warnings)
		}

		want := `server library "core" is also created by resource:resources/`
		warnings := warningsOf(garage)
		if len(warnings) != 1 || !strings.HasPrefix(warnings[0], "src/server/core.library.ts:3") ||
			!strings.Contains(warnings[0], want+"characters;") {
			t.Fatalf("expected one collision with resources/characters, got %q", warnings)
		}
		warnings = warningsOf(characters)
		if len(warnings) != 1 || !strings.HasPrefix(warnings[0], "src/server/core.library.ts:3") ||
			!strings.Contains(warnings[0], want+"garage;") {
			t.Fatalf("expected one collision with resources/garage, got %q", warnings)
		}

		tsc := exec.Command("node", filepath.Join(typeScriptDir, "bin", "tsc"), "-p", filepath.Join(root, "tsconfig.json"))
		if output, err := tsc.CombinedOutput(); err != nil {
			t.Fatalf("the generated types do not compile: %v\n%s", err, output)
		}
	})
}

// A build generates every resource before bundling runs typegen again per task, so the resource
// generated first also reports a collision it could not see yet, even without type-checking.
func TestBuilder_BothResourcesReportALibraryCollision(t *testing.T) {
	forEachTypeScript(t, func(t *testing.T, typeScriptDir string) {
		root := t.TempDir()
		writeTestFile(t, root, "tsconfig.json", typegenTestTSConfig)
		characters := filepath.Join(root, "resources", "characters")
		garage := filepath.Join(root, "resources", "garage")

		// Found through its symbol, whatever it is imported as.
		writeTestFile(t, characters, "src/server/core.library.ts", `
import { createServerLibrary as makeLibrary } from '@open-core/framework/server'
export const core = makeLibrary('core')
`)
		writeTestFile(t, garage, "src/server/core.library.ts", `
import { createServerLibrary } from '@open-core/framework/server'
export const core = createServerLibrary('core')
`)
		// A function of the project that only shares the name creates no library.
		writeTestFile(t, garage, "src/server/local.ts", `
function createServerLibrary(name: string) { return { name, side: 'server' as const } }
export const local = createServerLibrary('core')
`)

		disabled := false
		b := New(&config.Config{Build: config.BuildConfig{Typecheck: &disabled}})
		rb := newTypegenBuilderWith(t, root, typeScriptDir, true)
		rb.ConfigureTypegen(true, TypegenOptions{})
		b.resourceBuilder = rb

		tasks := []BuildTask{
			{Path: characters, ResourceName: "characters", Type: TypeResource},
			{Path: garage, ResourceName: "garage", Type: TypeResource},
		}
		if err := b.prepareTypes(context.Background(), tasks, true); err != nil {
			t.Fatalf("prepareTypes returned error: %v", err)
		}
		for _, task := range tasks {
			rb.RunTypegen(task.Path)
		}

		for _, resource := range []string{characters, garage} {
			result, err := rb.generateTypes(resource, TypegenOptions{})
			if err != nil {
				t.Fatalf("generateTypes returned error: %v", err)
			}
			if len(result.Warnings) != 1 || result.Warnings[0].File != "src/server/core.library.ts" {
				t.Fatalf("expected one collision in %s, got %v", resource, result.Warnings)
			}
			if !rb.typegenWarned[resource] {
				t.Fatalf("the collision in %s was never reported", resource)
			}
		}
	})
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
