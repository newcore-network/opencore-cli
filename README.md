# OpenCore CLI

Command-line interface for the OpenCore Framework. Build, manage, and deploy FiveM TypeScript projects with a modern toolchain.

[![License](https://img.shields.io/badge/license-MPL--2.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://go.dev/)
[![NPM Version](https://img.shields.io/npm/v/@open-core/cli.svg)](https://www.npmjs.com/package/@open-core/cli)

## Overview

OpenCore CLI provides a complete build system for FiveM TypeScript projects. It handles project scaffolding, parallel compilation, resource management, and deployment to FiveM servers.

**Key capabilities:**

- Project initialization with multiple architecture patterns
- Parallel build system with esbuild and SWC for decorator support
- Three resource types: Core, Resource (satellite), and Standalone
- Hot-reload development mode with file watching
- Automatic deployment to FiveM server directories
- Embedded build toolchain (no project-level build scripts required)

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
| `opencore dev` | Start development mode with file watching |
| `opencore doctor` | Validate project configuration |
| `opencore version` | Display CLI version |

## Quick Start

```bash
opencore init my-server
cd my-server
pnpm install
opencore dev
```

## Configuration

Projects are configured via `opencore.config.ts`:

```typescript
import { defineConfig } from '@open-core/cli'

export default defineConfig({
  name: 'my-server',
  outDir: './dist/resources',

  // Deploy builds directly to FiveM server
  destination: 'C:/FXServer/server-data/resources/[my-server]',

  core: {
    path: './core',
    resourceName: '[core]',
    entryPoints: {
      server: './core/src/server.ts',
      client: './core/src/client.ts',
    },
    // Optional: custom build script
    // customCompiler: './scripts/core-build.js',
  },

  resources: {
    include: ['./resources/*'],
    explicit: [
      {
        path: './resources/admin',
        resourceName: 'admin-panel',
        views: {
          path: './resources/admin/ui',
          framework: 'react',
        },
      },
    ],
  },

  standalone: {
    include: ['./standalone/*'],
    explicit: [
      { path: './standalone/utils', compile: true },
      { path: './standalone/legacy', compile: false },  // Copy without compilation
    ],
  },

  modules: ['@open-core/identity'],

  build: {
    minify: true,
    sourceMaps: true,
    target: 'ES2020',
    parallel: true,
    maxWorkers: 8,
  },
})
```

## Resource Types

### Core

The central resource containing the framework runtime, dependency injection container, and shared services. All other resources depend on core at runtime.

### Resource (Satellite)

Resources that extend core functionality. They import from `@open-core/framework` which resolves to core exports at runtime via FiveM's resource system.

### Standalone

Independent resources with no core dependency. Can use basic decorators via SWC. Set `compile: false` for pure Lua/JS resources that should be copied without transformation.

## Build System

The CLI embeds a complete build toolchain based on esbuild with SWC for TypeScript decorator support. No build scripts are required in individual projects.

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

## Project Architectures

The CLI supports four project structures:

**Domain-Driven**: Organize by business domains with client/server/shared separation. Recommended for large projects.

**Layer-Based**: Traditional separation by technical layers. Suitable for teams with specialized frontend/backend roles.

**Feature-Based**: Flat feature structure for rapid development. Good for small to medium projects.

**Hybrid**: Mix domain modules for critical systems with simple feature folders for lightweight functionality.

## Development

```bash
go mod download
go test ./...
go build -o opencore .
```

### Running Tests

```bash
go test ./... -v
```

## Requirements

- Go 1.21+ (for building the CLI)
- Node.js 18+ (for project compilation)
- pnpm (recommended) or npm

## License

MPL-2.0. See [LICENSE](LICENSE) for details.

## Links

- [OpenCore Framework](https://github.com/newcore-network/opencore)
- [NPM Package](https://www.npmjs.com/package/@open-core/cli)
- [GitHub Releases](https://github.com/newcore-network/opencore-cli/releases)
