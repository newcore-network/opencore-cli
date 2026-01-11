# Configuration

OpenCore CLI uses `opencore.config.ts` for project configuration.

## Basic Example

```typescript
import { defineConfig } from '@open-core/cli'

export default defineConfig({
  name: 'my-server',
  destination: 'C:/FXServer/server-data/resources/[my-server]',

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
    port: 3847,
    txAdminUrl: 'http://localhost:40120',
    txAdminUser: 'admin',
    txAdminPassword: '',
  },
})
```

## Configuration Reference

### Root Options

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `name` | `string` | Yes | Project name |
| `destination` | `string` | Yes | FiveM server resources path |
| `core` | `CoreConfig` | Yes | Core resource configuration |
| `resources` | `ResourcesConfig` | No | Satellite resources |
| `standalone` | `StandaloneConfig` | No | Standalone resources |
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

See [FiveM Runtime](./fivem-runtime.md) for platform details.