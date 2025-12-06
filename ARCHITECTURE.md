# OpenCore CLI Architecture

This document explains the internal architecture of the OpenCore CLI.

## Overview

The OpenCore CLI is written in Go and uses the Charmbracelet ecosystem for beautiful terminal UI. It orchestrates TypeScript compilation and FiveM resource management.

## Project Structure

```
opencore-cli/
├── cmd/
│   └── opencore/
│       └── main.go              # CLI entry point
├── internal/
│   ├── commands/                # Command implementations
│   │   ├── init.go              # Project initialization
│   │   ├── create.go            # Feature/resource creation
│   │   ├── create_feature.go
│   │   ├── create_resource.go
│   │   ├── build.go             # Build orchestration
│   │   ├── dev.go               # Development mode
│   │   ├── doctor.go            # Health checks
│   │   └── clone.go             # Template cloning
│   ├── config/
│   │   └── config.go            # Config loading & parsing
│   ├── builder/
│   │   └── builder.go           # Build orchestration
│   ├── watcher/
│   │   └── watcher.go           # File watching for dev mode
│   ├── templates/
│   │   ├── embed.go             # Template embedding
│   │   ├── starter-project/     # Project templates
│   │   ├── resource/            # Resource templates
│   │   └── feature/             # Feature templates
│   └── ui/
│       └── styles.go            # Terminal UI styles
├── npm/                         # NPM wrapper package
│   ├── package.json
│   ├── index.js                 # Binary executor
│   ├── binary.js                # Platform detection
│   └── install.js               # Post-install downloader
└── .github/
    └── workflows/
        ├── release.yml          # Release automation
        └── test.yml             # CI testing

```

## Key Components

### 1. Commands (`internal/commands/`)

Each command is implemented as a Cobra command:

- **init**: Creates new projects using embedded templates
- **create**: Generates features or resources from templates
- **build**: Orchestrates TypeScript compilation via pnpm
- **dev**: File watching with hot-reload
- **doctor**: Health checks and validation
- **clone**: Downloads official templates from GitHub

### 2. Config Loader (`internal/config/`)

Loads and parses `opencore.config.ts`:

1. Creates a temporary Node.js script
2. Executes it to transpile TypeScript to JSON
3. Parses JSON into Go structs

This allows users to write config in TypeScript with type safety.

### 3. Builder (`internal/builder/`)

Orchestrates builds using Bubbletea TUI:

1. Reads config to find all resources
2. For each resource:
   - Runs `pnpm build` in the resource directory
   - Shows animated spinner during build
   - Collects timing and error information
3. Displays final results with styled output

### 4. Watcher (`internal/watcher/`)

Implements hot-reload for development:

1. Uses `fsnotify` to watch `src/` directories
2. Debounces changes (300ms)
3. Triggers rebuild on file modifications
4. Shows live feedback in terminal

### 5. Templates (`internal/templates/`)

Embeds template files using `//go:embed`:

- **starter-project**: Full project structure
- **resource**: Independent resource template
- **feature**: Feature module template

Uses Go's `text/template` for variable substitution.

### 6. UI (`internal/ui/`)

Beautiful terminal output using Charmbracelet:

- **lipgloss**: Styling, colors, boxes, borders
- **bubbles**: Spinners, progress bars
- **bubbletea**: Interactive TUI framework
- **huh**: Interactive forms

## Data Flow

### Init Command Flow

```
User runs: opencore init my-server
    ↓
main.go → commands/init.go
    ↓
Show interactive form (huh)
    ↓
templates.GenerateStarterProject()
    ↓
Read embedded templates
    ↓
Execute text/template with config
    ↓
Write files to disk
    ↓
Show success message (lipgloss)
```

### Build Command Flow

```
User runs: opencore build
    ↓
main.go → commands/build.go
    ↓
config.Load() → Parse opencore.config.ts
    ↓
builder.New(config).Build()
    ↓
Create Bubbletea model
    ↓
For each resource:
    ↓
    buildResource() → exec.Command("pnpm", "build")
    ↓
    Update TUI with progress
    ↓
Show final results
```

### Dev Command Flow

```
User runs: opencore dev
    ↓
main.go → commands/dev.go
    ↓
config.Load()
    ↓
watcher.New(config)
    ↓
Initial build
    ↓
fsnotify.NewWatcher()
    ↓
Watch src/ directories
    ↓
On file change:
    ↓
    Debounce (300ms)
    ↓
    builder.Build()
    ↓
    Show results
    ↓
Loop until Ctrl+C
```

## Distribution

### Go Binary

Compiled for multiple platforms:

- `windows-amd64`
- `darwin-amd64`
- `darwin-arm64`
- `linux-amd64`

### NPM Package

The NPM package (`@open-core/cli`) is a wrapper:

1. **Post-install**: Downloads correct binary from GitHub Releases
2. **Bin script**: Spawns Go binary with user's arguments
3. **Platform detection**: Automatically selects correct binary

This gives users the convenience of `npm install` while using a fast Go binary.

## CI/CD

### GitHub Actions

**test.yml**: Runs on every push/PR

- Tests on Linux, macOS, Windows
- Runs `go test ./...`
- Runs `go build`

**release.yml**: Runs on tag push

1. Build binaries for all platforms
2. Create GitHub Release
3. Upload binaries as release assets
4. Publish NPM package

## Design Decisions

### Why Go?

- **Fast compilation**: Near-instant builds
- **Single binary**: Easy distribution
- **Cross-platform**: Build for all platforms from any OS
- **Concurrency**: Easy parallel builds

### Why Charmbracelet?

- **Beautiful**: Professional-looking TUI
- **Modern**: Actively maintained
- **Comprehensive**: Covers all UI needs
- **Go-native**: No external dependencies

### Why Embed Templates?

- **Self-contained**: No network required for init/create
- **Versioned**: Templates match CLI version
- **Fast**: Instant access

### Why TypeScript Config?

- **Type safety**: Catch errors at write-time
- **IntelliSense**: IDE autocomplete
- **Familiar**: Same language as user's code

## Performance

- **Build time**: ~500ms for typical CLI build
- **Binary size**: ~15-20MB (includes templates)
- **Memory usage**: ~10-30MB during operation
- **Startup time**: <100ms

## Future Enhancements

1. **Plugin system**: Allow community plugins
2. **Remote templates**: Download templates from URLs
3. **Build caching**: Speed up incremental builds
4. **Parallel builds**: Build multiple resources simultaneously
5. **Config validation**: Schema validation for config files

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines.

---

For questions or discussions about the architecture, open an issue on GitHub.
