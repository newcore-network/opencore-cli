# OpenCore CLI

> Modern build system for FiveM TypeScript projects with full decorator support

[![License](https://img.shields.io/badge/license-MPL--2.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://go.dev/)
[![NPM Version](https://img.shields.io/npm/v/@open-core/cli.svg)](https://www.npmjs.com/package/@open-core/cli)

## Overview

OpenCore CLI is the official build toolchain for the OpenCore Framework, providing enterprise-grade TypeScript compilation for FiveM servers. It combines esbuild's speed with SWC's decorator support to deliver fast, reliable builds with modern JavaScript features.

### Architecture

The CLI implements a three-tier build architecture:

- Core Runtime: Framework initialization, dependency injection, and shared services
- Satellite Resources: Domain-specific modules that depend on core at runtime
- Standalone Resources: Independent modules with bundled dependencies

### Key Features

- Parallel Compilation: Multi-core builds with configurable worker pools
- Full TypeScript Support: Decorators, metadata, and modern ES features
- Zero Configuration: Embedded build toolchain, no project setup required
- Hot Reload: File watching with incremental compilation
- Build Analytics: Bundle size tracking and performance metrics
- FiveM Optimization: Node.js module exclusion and FiveM-specific transforms
- Auto-Deployment: Direct deployment to FiveM server directories

## Installation

### NPM (Recommended)

```bash
npm install -g @open-core/cli
```

### Go

```bash
go install github.com/newcore-network/opencore-cli@latest
```

### From Source

```bash
git clone https://github.com/newcore-network/opencore-cli
cd opencore-cli
go build -o opencore .
```

## Commands

| Command | Description |
|---------|-------------|
| `opencore init [name]` | Initialize a new project with interactive wizard |
| `opencore build` | Build all resources for production |
| `opencore create <type>` | Create scaffolding (feature, resource, standalone) |
| `opencore clone <template>` | Clone an official template (chat, admin, racing) |
| `opencore dev` | Start development mode with file watching |
| `opencore doctor` | Validate project configuration |
| `opencore version` | Display CLI version |

### Create Command

The `create` command generates scaffolding for different project components:

```bash
# Create a feature in the core
opencore create feature banking

# Create a feature inside a specific resource
opencore create feature chat -r myserver

# Create a satellite resource (depends on core)
opencore create resource admin --with-client --with-nui

# Create a standalone resource (no dependencies)
opencore create standalone utils --with-client
```

| Subcommand | Flags | Description |
|------------|-------|-------------|
| `feature [name]` | `-r, --resource <name>` | Create feature in core (default) or inside a resource |
| `resource [name]` | `--with-client`, `--with-nui` | Create satellite resource in `resources/` |
| `standalone [name]` | `--with-client`, `--with-nui` | Create standalone resource in `standalone/` |

**Feature flag `-r`:** When specified, creates the feature inside an existing resource instead of the core:
```bash
opencore create feature banking              # → core/src/features/banking/
opencore create feature banking -r admin     # → resources/admin/src/server/features/banking/
```

### Clone Command

Download official templates from the [opencore-templates](https://github.com/newcore-network/opencore-templates) repository:

```bash
# List all available templates
opencore clone --list

# Clone a template to resources/
opencore clone chat

# Force using GitHub API instead of git sparse-checkout
opencore clone admin --api
```

| Flag | Description |
|------|-------------|
| `-l, --list` | List all available templates from the repository |
| `--api` | Force download via GitHub API (skips git sparse-checkout) |

**How it works:**
1. If git >= 2.25 is available, uses sparse-checkout (faster, downloads only the template folder)
2. Falls back to GitHub API if git is unavailable or older version

## Quick Start

```bash
opencore init my-server
cd my-server
pnpm install
opencore dev
```

## Configuration

Projects are configured via `opencore.config.ts` with full TypeScript support and IDE autocompletion:

```typescript
import { defineConfig } from '@open-core/cli'

export default defineConfig({
  name: 'newcore',
  outDir: './build',

  destination: 'C:/Users/alexf/Projects/Newcore Software/newcore/server/resources',

  core: {
    path: './core',
    resourceName: 'core',
    entryPoints: {
      server: './core/src/server.ts',
      client: './core/src/client.ts',
    },
  },

  // Satellite resources (depend on core at runtime)
  resources: {
    include: ['./resources/*'],
    explicit: [
      {
        path: './resources/admin',
        resourceName: 'admin',
        build: {minify: false, server: true, client: false, nui: false, sourceMaps: false},

        // Whether to compile this resource. Set to false to just copy files without compilation.
        // Useful for legacy Lua resources or pre-compiled code.
        compile: true,

        // Path to a custom build script for this resource. if you need
        customCompiler: undefined,

        entryPoints: {
          client: './resources/admin/src/client/main.ts',
          server: './resources/admin/src/server/main.ts'
        },

        // Resource type identifier (for internal use).
        type: 'administrator',

        // Views / UI, of course, recommended in build `nui: true`
        views: {
          path: './resources/admin/ui',
          framework: 'react'
        }
      },
    ],
  },

  // Standalone resources (no core dependency)
  standalone: {
     include: ['./standalone/*'],
     explicit: [
       { path: './standalone/utils', compile: true },
       { path: './standalone/legacy', compile: false },  // Just copy, no build
       { path: './standalone/custom', customCompiler: './scripts/custom-build.js' /* Yes, you can create your own compiler. */ },
     ],
   },

  build: {
  minify: true,
  sourceMaps: false,
  target: 'ES2020',
  parallel: true,
  maxWorkers: 8,
  },

  dev: {
    port: 3847,
    // or you can use enviroment variables

    // VAR: OPENCORE_TXADMIN_USER
    txAdminUser: '',
    // VAR: OPENCORE_TXADMIN_PASSWORD
    txAdminPassword: '',
    // VAR: OPENCORE_TXADMIN_URL
    txAdminUrl: ''
  }
})

```

### Configuration Reference

| Property | Type | Default | Description |
|----------|------|---------|-------------|
| `name` | `string` | - | Project identifier |
| `outDir` | `string` | `"./build"` | Output directory (cleaned before each build) |
| `destination` | `string` | - | FiveM server deployment path |
| `core` | `CoreConfig` | - | **Required**. Core resource configuration |
| `resources` | `ResourcesConfig` | - | Satellite resources configuration |
| `standalone` | `StandaloneConfig` | - | Independent resources configuration |
| `modules` | `string[]` | - | OpenCore modules to include |
| `build` | `BuildConfig` | - | Global build settings |

## Resource Types

### Core Resource

The central resource containing the framework runtime, dependency injection container, and shared services. All other resources depend on core at runtime.

**Characteristics:**
- Initializes OpenCore Framework
- Provides dependency injection container
- Exports shared services and utilities
- Bundles all dependencies (no externals)

### Satellite Resources

Domain-specific modules that extend core functionality. They import from `@open-core/framework` which resolves to core exports at runtime via FiveM's resource system.

**Characteristics:**
- External dependency on `@open-core/framework`
- Cannot access Node.js built-ins
- Optimized for smaller bundle sizes
- Runtime dependency on core resource

### Standalone Resources

Independent modules with no core dependency. Can use basic decorators via SWC. Set `compile: false` for pure Lua/JS resources that should be copied without transformation.

**Characteristics:**
- No external dependencies
- All dependencies bundled
- Can use Node.js built-ins (if not excluded)
- Independent deployment

## Build System

### Embedded Toolchain

The CLI embeds a complete build toolchain based on esbuild with SWC for TypeScript decorator support. No build scripts are required in individual projects.

**Build Pipeline:**
1. **SWC Transformation**: TypeScript → JavaScript with decorators
2. **esbuild Bundling**: JavaScript → Optimized bundles
3. **FiveM Optimization**: Node.js module exclusion, export patching
4. **Size Analysis**: Bundle size tracking and reporting

### Performance Optimizations

- **Parallel Compilation**: Multi-core builds with configurable worker pools
- **Incremental Builds**: File watching with hot reload in development
- **Bundle Splitting**: Separate server/client bundles
- **Tree Shaking**: Dead code elimination
- **Minification**: Optional code compression

### Custom Build Scripts

For advanced use cases, specify a custom compiler per resource:

```typescript
core: {
  path: './core',
  customCompiler: './scripts/custom-build.js',
}
```

Custom compilers receive the same interface as the embedded script:

```bash
node custom-build.js single <type> <path> <outDir> '<options-json>'
```

**Interface:**
- `<type>`: `core`, `resource`, `standalone`, or `views`
- `<path>`: Source directory path
- `<outDir>`: Output directory path
- `<options-json>`: Build options as JSON string

## Project Architectures

The CLI supports four project structures tailored to different team sizes and complexity:

### Domain-Driven (Recommended for Large Projects)

```
project/
├── core/                    # Framework runtime
├── domains/
│   ├── authentication/
│   │   ├── src/
│   │   │   ├── server/
│   │   │   ├── client/
│   │   │   └── shared/
│   │   └── views/
│   ├── inventory/
│   └── vehicles/
└── shared/                   # Cross-domain utilities
```

**Benefits:** Clear domain boundaries, scalable team organization, reduced coupling.

### Layer-Based (Traditional Teams)

```
project/
├── core/
├── layers/
│   ├── controllers/         # Server logic
│   ├── services/           # Business logic
│   ├── repositories/      # Data access
│   └── ui/                # Client interfaces
└── shared/
```

**Benefits:** Familiar structure, specialized frontend/backend roles.

### Feature-Based (Rapid Development)

```
project/
├── core/
├── features/
│   ├── player-spawn/
│   ├── vehicle-shop/
│   └── admin-panel/
└── shared/
```

**Benefits:** Fast iteration, feature isolation, easy onboarding.

### Hybrid (Mixed Approach)

Combine domain modules for critical systems with simple feature folders for lightweight functionality.

## Development

### Building from Source

```bash
git clone https://github.com/newcore-network/opencore-cli
cd opencore-cli
go mod download
go test ./...
go build -o opencore .
```

### Running Tests

```bash
go test ./... -v
```

### Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Architecture

The CLI is built with Go and consists of:

- Commands: CLI command definitions and handlers
- Builder: Build system with parallel compilation
- Config: TypeScript configuration parser
- UI: Terminal interfaces and progress displays
- Templates: Project scaffolding templates

## Requirements

- Go 1.21+ (for building the CLI)
- Node.js 18+ (for TypeScript compilation)
- pnpm (recommended) or npm: Package management
- Windows/Linux/macOS: Cross-platform support

## Performance Benchmarks

Build performance on a typical project with 10 resources:

| Configuration | Build Time | Memory Usage |
|---------------|------------|--------------|
| Sequential    | 2.3s       | 45MB         |
| Parallel (4 cores) | 0.8s   | 120MB        |
| Parallel (8 cores) | 0.5s   | 200MB        |

## License

MPL-2.0. See [LICENSE](LICENSE) for details.

## Links

- [OpenCore Framework](https://github.com/newcore-network/opencore)
- [NPM Package](https://www.npmjs.com/package/@open-core/cli)
- [GitHub Releases](https://github.com/newcore-network/opencore-cli/releases)
- [Documentation](https://opencore.dev/docs/cli)
