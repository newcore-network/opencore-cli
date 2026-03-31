# Configuration

OpenCore CLI uses `opencore.config.ts` for project configuration.

## Basic Example

```typescript
import { defineConfig } from '@open-core/cli'
import { FiveMClientAdapter } from '@open-core/fivem-adapter/client'
import { FiveMServerAdapter } from '@open-core/fivem-adapter/server'

export default defineConfig({
  name: 'my-server',
  destination: 'C:/FXServer/server-data/resources/[my-server]',
  adapter: {
    server: FiveMServerAdapter(),
    client: FiveMClientAdapter(),
  },

  core: {
    path: './core',
    resourceName: 'core',
  },

  resources: {
    include: ['./resources/*'],
  },

  build: {
    minify: true,
    parallel: true,
  },
})
```

## Adapter-Based Runtime

OpenCore uses the configured adapter as the runtime source of truth.

### FiveM Example

```typescript
import { defineConfig } from '@open-core/cli'
import { FiveMClientAdapter } from '@open-core/fivem-adapter/client'
import { FiveMServerAdapter } from '@open-core/fivem-adapter/server'

export default defineConfig({
  name: 'my-fivem-server',
  destination: 'C:/FXServer/server-data/resources',
  adapter: {
    server: FiveMServerAdapter(),
    client: FiveMClientAdapter(),
  },
})
```

### RageMP Example

```typescript
import { defineConfig } from '@open-core/cli'
import { RageMPClientAdapter } from '@open-core/ragemp-adapter/client'
import { RageMPServerAdapter } from '@open-core/ragemp-adapter/server'

export default defineConfig({
  name: 'my-ragemp-server',
  destination: 'C:/ragemp-server',
  adapter: {
    server: RageMPServerAdapter(),
    client: RageMPClientAdapter(),
  },
  build: {
    target: 'node14',
  },
})
```

Runtime behavior:

- FiveM: standard resource layout with `fxmanifest.lua`
- RageMP: server output in `packages/`, client output in `client_packages/`
- The compiler injects the configured adapter into built bundles automatically

## Full Example

```typescript
import { defineConfig } from '@open-core/cli'

export default defineConfig({
  name: 'my-server',
  destination: 'C:/FXServer/server-data/resources/[my-server]',

  core: {
    path: './core',
    resourceName: 'core',
    entryPoints: {
      server: './core/src/server.ts',
      client: './core/src/client.ts',
    },
  },

  resources: {
    include: ['./resources/*'],
    explicit: [
      {
        path: './resources/admin',
        resourceName: 'admin',
        build: {
          server: { external: ['typeorm'] },
          client: false,  // Server-only resource
        },
      },
    ],
  },

  standalone: {
    include: ['./standalone/*'],
  },

  build: {
    minify: true,
    sourceMaps: false,
    parallel: true,
    maxWorkers: 8,

    server: {
      platform: 'node',
      format: 'cjs',
      target: 'es2020',
    },

    client: {
      platform: 'neutral',
      format: 'iife',
      target: 'es2020',
    },
  },

  dev: {
    bridge: {
      port: 3847,
    },
    restart: {
      mode: 'auto',
    },
    txAdmin: {
      url: 'http://localhost:40120',
      user: 'admin',
      password: '',
    },
    process: {
      command: './server',
      args: [],
      cwd: '../server',
    },
  },
})
```

Notes:

- `dev.bridge.port` is the CLI/framework bridge port used for development logs and tooling.
- `dev.txAdmin` is optional and intended for txAdmin-managed FiveM restarts.
- `dev.process` is the simplest cross-runtime option for RageMP or custom servers: build, stop process, start process again.

## Configuration Reference

### Root Options

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `name` | `string` | Yes | Project name |
| `destination` | `string` | Yes | Deployment root path for the selected runtime |
| `core` | `CoreConfig` | Yes | Core resource configuration |
| `resources` | `ResourcesConfig` | No | Satellite resources |
| `standalone` | `StandaloneConfig` | No | Standalone resources |
| `adapter` | `OpenCoreAdapterConfig` | No | Central server/client runtime adapters |
| `build` | `BuildConfig` | No | Global build settings |
| `dev` | `DevConfig` | No | Development settings |

### Build Options

| Property | Type | Default | Description |
|----------|------|---------|-------------|
| `minify` | `boolean` | `false` | Minify output |
| `sourceMaps` | `boolean` | `false` | Generate source maps |
| `parallel` | `boolean` | `false` | Parallel compilation |
| `maxWorkers` | `number` | CPU cores | Max parallel workers |
| `server` | `SideBuildConfig` | - | Server build config |
| `client` | `SideBuildConfig` | - | Client build config |

### Side Build Options

| Property | Type | Default | Description |
|----------|------|---------|-------------|
| `platform` | `string` | `node`/`neutral` | Build platform |
| `format` | `string` | `cjs`/`iife` | Output format |
| `target` | `string` | `es2020` | JS target |
| `external` | `string[]` | `[]` | External packages (server only) |

See [FiveM Runtime](./fivem-runtime.md) for FiveM platform details.

## Views Vite Configuration

OpenCore now supports only two view build modes:

- `vite`: recommended for React, Vue, Svelte, Astro, Tailwind, PostCSS, Sass, and any modern frontend stack
- `vanilla`: minimal HTML/CSS/JS/TS views built directly by the CLI

Resolution order:

- `views.framework: 'vite'` forces Vite
- `views.framework: 'vanilla'` forces the minimal CLI runner
- Without an explicit framework, OpenCore uses Vite when it finds `vite.config.*` in the view directory or in the project root next to `opencore.config.ts`
- Otherwise, OpenCore falls back to `vanilla`

Recommended setup:

- Keep a shared root `vite.config.ts` next to `opencore.config.ts`
- Let each project configure its own framework plugins, CSS pipeline, and browser targets in Vite
- Add PostCSS only when your frontend needs it, especially for older runtimes such as RageMP CEF

Helper:

- OpenCore exposes `createOpenCoreViteConfig` from `@open-core/cli/vite` for shared root configs
- The helper auto-resolves `OPENCORE_VIEW_ROOT`, `OPENCORE_VIEW_OUTDIR`, and `postcss.config.*` from the OpenCore project root

Example:

```ts
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { createOpenCoreViteConfig } from '@open-core/cli/vite'

export default createOpenCoreViteConfig({
  plugins: [react(), tailwindcss()],
  build: {
    target: 'chrome97',
  },
})
```

Per-view `package.json` scripts are optional. They are useful for local development, but `opencore build` does not require them.

Removed support:

- The CLI no longer provides dedicated React, Vue, Svelte, or Astro builders
- The CLI no longer auto-manages Tailwind/PostCSS/Sass for views

If you need any of those features, switch the view to `framework: 'vite'` and configure them in your Vite project.
