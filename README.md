# OpenCore CLI

> Modern build system for FiveM TypeScript projects with full decorator support

[![License](https://img.shields.io/badge/license-MPL--2.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://go.dev/)
[![NPM Version](https://img.shields.io/npm/v/@open-core/cli.svg)](https://www.npmjs.com/package/@open-core/cli)

---

## Documentation

This may be outdated, the latest information is recommended at [opencorejs.dev](https://opencorejs.dev/docs/compiler/about)


| Topic | Description |
|-------|-------------|
| [FiveM Runtime](./docs/fivem-runtime.md) | Server vs Client, platform limitations, incompatible packages |
| [Configuration](./docs/configuration.md) | Full configuration reference and examples |
| [Commands](./docs/commands.md) | CLI commands and usage |

---

## Key Features

| Feature | Description |
|---------|-------------|
| **Parallel Compilation** | Multi-core builds with configurable worker pools for maximum performance |
| **Full TypeScript Support** | Decorators, metadata reflection, and modern ES2020+ features |
| **Simple Configuration** | Embedded build toolchain with sensible defaults, no setup required, a single typed config file |
| **Hot Reload** | File watching with incremental compilation for rapid development |
| **Three-Tier Architecture** | Core, satellite resources, and standalone scripts with proper dependency management |
| **FiveM Optimization** | Node.js module exclusion, FiveM-specific transforms, and native API support |
| **Official Templates** | Clone production-ready templates directly from the OpenCore repository |
| **Project Scaffolding** | Generate features, resources, and standalone scripts with a single command |

---

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

---

## Commands

| Command | Description |
|---------|-------------|
| `opencore init [name]` | Initialize a new project with interactive wizard |
| `opencore build` | Build all resources for production |
| `opencore completion` | Completion files config  to set in your zsh, bash etc |
| `opencore create <type>` | Create scaffolding (feature, resource, standalone) |
| `opencore clone <template>` | Clone an official template |
| `opencore dev` | Start development mode with file watching |
| `opencore doctor` | Validate project configuration |
| `opencore update` | self-update CLI |
| `opencore --v` | Display CLI version |
| `opencore --h` | Help |

---

## Quick Start

```bash
opencore init my-server
cd my-server
pnpm install
opencore dev
```

If you don't have pnpm installed, you can use:

```bash
npm install
```

Or yarn (modern/berry):

```bash
yarn install
```

You can also force a package manager:

```bash
opencore init my-server --usePackageManager=pnpm
opencore init my-server --usePackageManager=yarn
opencore init my-server --usePackageManager=npm
```

---

## Create Command

Generate scaffolding for different project components:

```bash
# Create a feature in the core
opencore create feature banking

# Create a feature inside a specific resource
opencore create feature chat -r myresource

# Create a satellite resource (depends on core)
opencore create resource admin --with-client --with-nui

# Create a standalone resource (no dependencies)
opencore create standalone utils --with-client
```

### Subcommands

| Subcommand | Flags | Description |
|------------|-------|-------------|
| `feature [name]` | `-r, --resource <name>` | Create feature in core (default) or inside a resource |
| `resource [name]` | `--with-client`, `--with-nui` | Create satellite resource in `resources/` |
| `standalone [name]` | `--with-client`, `--with-nui` | Create standalone resource in `standalone/` |

The `-r` flag allows creating features inside existing resources:

```bash
opencore create feature banking              # creates in core/src/features/banking/
opencore create feature banking -r admin     # creates in resources/admin/src/server/features/banking/
```

---

## Clone Command

Download official templates from the [opencore-templates](https://github.com/newcore-network/opencore-templates) repository:

```bash
# List all available templates
opencore clone --list

# List templates from a development branch
opencore clone --list --branch develop

# Clone a template to resources/
opencore clone chat

# Clone a template from a development branch
opencore clone chat --branch develop

# Force using GitHub API instead of git sparse-checkout
opencore clone admin --api
```

### Flags

| Flag | Description |
|------|-------------|
| `-l, --list` | List all available templates from the repository |
| `-b, --branch <name>` | Repository branch to use when listing or cloning templates (default: `master`) |
| `--api` | Force download via GitHub API (skips git sparse-checkout) |

The clone command automatically selects the best download method:
1. Uses git sparse-checkout if git >= 2.25 is available (faster)
2. Falls back to GitHub API for older git versions or when git is unavailable

---

## Configuration

> Full documentation: [docs/configuration.md](./docs/configuration.md)

Projects are configured via `opencore.config.ts`:

```typescript
import { defineConfig } from '@open-core/cli'

export default defineConfig({
  name: 'my-server',
  destination: '/path/to/fxserver/resources',

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

### Configuration Reference

| Property | Type | Default | Description |
|----------|------|---------|-------------|
| `name` | `string` | - | Project identifier |
| `destination` | `string` | - | FiveM server deployment path (mandatory) |
| `core` | `CoreConfig` | - | Core resource configuration (required) |
| `resources` | `ResourcesConfig` | - | Satellite resources configuration |
| `standalone` | `StandaloneConfig` | - | Independent resources configuration |
| `modules` | `string[]` | - | OpenCore modules to include |
| `build` | `BuildConfig` | - | Global build settings |
| `dev` | `DevConfig` | - | Development server settings |

---

## Resource Types

### Core Resource

The central resource containing the framework runtime, dependency injection container, and shared services.

- Initializes OpenCore Framework
- Provides dependency injection container
- Exports shared services and utilities
- Bundles all dependencies

### Satellite Resources

Domain-specific modules that extend core functionality. They import from `@open-core/framework` which resolves to core exports at runtime.

- External dependency on `@open-core/framework`
- Optimized for smaller bundle sizes
- Runtime dependency on core resource

### Standalone Resources

Independent modules with no core dependency. Set `compile: false` for Lua/JS resources that should be copied without transformation.

- No external dependencies
- All dependencies bundled
- Independent deployment

---

## Build System

### Embedded Toolchain

The CLI embeds a complete build toolchain based on esbuild with SWC for TypeScript decorator support.

**Build Pipeline:**
1. SWC Transformation: TypeScript to JavaScript with decorators
2. esbuild Bundling: JavaScript to optimized bundles
3. FiveM Optimization: Neutral platform targeting, export patching
4. Size Analysis: Bundle size tracking and reporting

---

## FiveM Runtime Environments

> Full documentation: [docs/fivem-runtime.md](./docs/fivem-runtime.md)

FiveM has **three distinct runtime environments**:

| Feature | Server | Client | Views (NUI) |
|---------|--------|--------|-------------|
| Runtime | Node.js | Neutral JS | Web Browser |
| Platform | `node` | `neutral` | `browser` |
| Node.js APIs | Available | NOT available | NOT available |
| Web APIs | NOT available | NOT available | Available |
| FiveM APIs | Available | Available | Via callbacks |
| GTA Natives | NOT available | Available | NOT available |
| External pkgs | Supported | NOT supported | N/A |

**Server**: Full Node.js runtime with all APIs.

**Client**: Neutral JS - FiveM APIs + GTA natives only. All deps MUST be bundled.

**Views**: Embedded Chromium browser with some version limitations.

### Custom Build Scripts

For advanced use cases, specify a custom compiler per resource:

```typescript
core: {
  path: './core',
  customCompiler: './scripts/custom-build.js',
}
```

Custom compilers receive the following interface:

```bash
node custom-build.js single <type> <path> <outDir> '<options-json>'
```

---

## Project Architectures

The CLI supports four project structures:

### Domain-Driven

Recommended for large projects with complex business logic.

```
project/
├── core/
├── domains/
│   ├── authentication/
│   │   ├── src/
│   │   │   ├── server/
│   │   │   ├── client/
│   │   │   └── shared/
│   │   └── views/
│   └── inventory/
└── shared/
```

### Layer-Based

Traditional separation by technical layers.

```
project/
├── core/
├── layers/
│   ├── controllers/
│   ├── services/
│   ├── repositories/
│   └── ui/
└── shared/
```

### Feature-Based

Simple structure for rapid development.

```
project/
├── core/
├── features/
│   ├── player-spawn/
│   ├── vehicle-shop/
│   └── admin-panel/
└── shared/
```

### Hybrid

Combine domain modules for critical systems with feature folders for lightweight functionality.

---

## Development

### Building from Source

```bash
git clone https://github.com/newcore-network/opencore-cli
cd opencore-cli
go mod download
go test ./...
go build -o opencore .
```

### Project Structure

```
opencore-cli/
├── internal/
│   ├── commands/     # CLI command handlers
│   ├── builder/      # Build system with parallel compilation
│   ├── config/       # TypeScript configuration parser
│   ├── templates/    # Project scaffolding templates
│   └── ui/           # Terminal interfaces
├── npm/              # NPM package wrapper
└── main.go           # Entry point
```

---

## Requirements

| Requirement | Version |
|-------------|---------|
| Go | 1.21+ (for building the CLI) |
| Node.js | 18+ (for TypeScript compilation) |
| Package Manager | pnpm (recommended) or npm |
| Platform | Windows, Linux, macOS |

---

## Performance

Build performance on a typical project with 10 resources:

| Configuration | Build Time | Memory Usage |
|---------------|------------|--------------|
| Sequential | 2.3s | 45MB |
| Parallel (4 cores) | 0.8s | 120MB |
| Parallel (8 cores) | 0.5s | 200MB |

---

## License

MPL-2.0. See [LICENSE](LICENSE) for details.

---

## Links

- [OpenCore Framework](https://github.com/newcore-network/opencore)
- [NPM Package](https://www.npmjs.com/package/@open-core/cli)
- [GitHub Releases](https://github.com/newcore-network/opencore-cli/releases)
