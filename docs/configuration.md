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

## Views PostCSS

OpenCore auto-detects PostCSS config for NUI/views builds from the project root.

Behavior:

- The CLI walks up from the views directory until it finds `opencore.config.ts`
- In that same directory it looks for `postcss.config.js`, `postcss.config.cjs`, `postcss.config.mjs`, or `postcss.config.ts`
- If one exists, that PostCSS config is used for CSS processing in the views build
- If none exists, the current built-in Tailwind fallback remains active

Example with Tailwind 4:

```typescript
import { defineConfig } from '@open-core/cli'

export default defineConfig({
  name: 'my-server',
  core: {
    path: './core',
    resourceName: 'core',
    views: {
      path: './web',
    },
  },
})
```

`postcss.config.mjs` at the project root:

```js
import tailwindcss from '@tailwindcss/postcss'
import autoprefixer from 'autoprefixer'

export default {
  plugins: [tailwindcss(), autoprefixer()],
}
```

## Views Vite Configuration

OpenCore treats Vite as the primary frontend build system for views:

- The recommended setup is a shared `vite.config.ts` in the same root where `opencore.config.ts` exists.
- View folders (for example `resources/chat/view`) do not need a local `vite.config.ts` by default.
- During build, OpenCore first checks for a local `vite.config.*` in the view directory.
- If local config is missing, OpenCore falls back to the root `vite.config.*` and builds the target view with that shared config.

For advanced cases, keep local config files as override layers that extend shared root defaults with `mergeConfig`.
