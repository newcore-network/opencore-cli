# OpenCore CLI

<div align="center">

**Official command-line tool for the OpenCore Framework**

[![License](https://img.shields.io/badge/license-MPL--2.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://go.dev/)
[![NPM Version](https://img.shields.io/npm/v/@open-core/cli.svg)](https://www.npmjs.com/package/@open-core/cli)

[Quick Start](QUICKSTART.md) • [Architecture](ARCHITECTURE.md) • [Contributing](CONTRIBUTING.md)

</div>

---

Official command-line tool for creating, managing, and building FiveM servers with the [OpenCore Framework](https://github.com/newcore-network/opencore).

## ✨ Features

- 🎨 **Beautiful CLI** - Animated UI with Charmbracelet ecosystem
- 📦 **Project Scaffolding** - Generate projects with best practices
- 🔧 **Generators** - Create features and resources instantly
- 🏗️ **Build Orchestration** - TypeScript → JavaScript compilation
- 🔥 **Hot Reload** - Development mode with file watching
- 🩺 **Health Checks** - Validate project configuration
- 📥 **Template Cloning** - Official templates from GitHub
- 🚀 **Fast** - Written in Go for maximum performance

## 📦 Installation

### Via NPM (Recommended)

```bash
npm install -g @open-core/cli
# or
pnpm add -g @open-core/cli
```

### Via Go

```bash
go install github.com/newcore-network/opencore-cli/cmd/opencore@latest
```

### From Source

```bash
git clone https://github.com/newcore-network/opencore-cli
cd opencore-cli
go build -o opencore ./cmd/opencore
```

## 🎯 Commands

| Command | Description |
|---------|-------------|
| `opencore init [name]` | Create a new OpenCore project |
| `opencore create feature [name]` | Create a new feature in core |
| `opencore create resource [name]` | Create a new independent resource |
| `opencore build` | Build all resources for production |
| `opencore dev` | Start development mode with hot-reload |
| `opencore doctor` | Validate project configuration |
| `opencore clone [template]` | Clone an official template |
| `opencore --version` | Show CLI version |

## 🏁 Quick Start

```bash
# Create a new project
opencore init my-server
cd my-server

# Install dependencies
pnpm install

# Create a feature in core
opencore create feature banking

# Create an independent resource
opencore create resource chat --with-client

# Start development mode
opencore dev

# Build for production
opencore build
```

**Want more details?** Check out the [Quick Start Guide](QUICKSTART.md).

## 📸 Screenshots

```
╭──────────────────────────────────────────╮
│                                          │
│               ◆ OpenCore CLI             │
│             By Newcore Network           │
│                                          │
╰──────────────────────────────────────────╯

Building Resources

✓ [core] compiled (1.2s)
✓ [oc-chat] compiled (0.8s)

┌─────────────────────────────────────────┐
│ ✓ Build completed successfully! 🚀      │
│                                         │
│ Resources: 2                            │
│ Time: 2.0s                              │
│ Output: ./dist/resources/               │
└─────────────────────────────────────────┘
```

## ⚙️ Configuration

Projects use an `opencore.config.ts` file:

```typescript
import { defineConfig } from '@open-core/cli'

export default defineConfig({
  name: 'my-server',
  outDir: './dist/resources',
  core: {
    path: './core',
    resourceName: '[core]',
  },
  resources: {
    include: ['./resources/*'],
  },
  modules: ['@open-core/identity'],
  build: {
    minify: true,
    sourceMaps: true,
  }
})
```

## 🛠️ Development

```bash
# Install dependencies
go mod download

# Run locally
go run ./cmd/opencore

# Build
go build -o opencore ./cmd/opencore

# Run tests
go test ./...
```

## 📚 Documentation

- [Quick Start Guide](QUICKSTART.md) - Get started in 5 minutes
- [Architecture](ARCHITECTURE.md) - Internal design and structure
- [Contributing](CONTRIBUTING.md) - Development guidelines
- [OpenCore Framework](https://github.com/newcore-network/opencore) - The main framework

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for:
- Development setup
- Code style guidelines
- How to submit PRs
- Reporting issues

## 📄 License

MPL-2.0 - See [LICENSE](LICENSE) for details.

## 🌟 Show Your Support

If you find OpenCore CLI useful, please consider:
- ⭐ Starring the repo on GitHub
- 📢 Sharing it with other FiveM developers
- 🐛 Reporting bugs and issues
- 💡 Suggesting new features
- 🤝 Contributing code

## 🔗 Links

- [OpenCore Framework](https://github.com/newcore-network/opencore)
- [OpenCore Identity Module](https://github.com/newcore-network/opencore-identity)
- [NPM Package](https://www.npmjs.com/package/@open-core/cli)
- [GitHub Releases](https://github.com/newcore-network/opencore-cli/releases)

---

<div align="center">

**Built with ❤️ by [Newcore Network](https://github.com/newcore-network)**

*Stop scripting. Start engineering.*

</div>

