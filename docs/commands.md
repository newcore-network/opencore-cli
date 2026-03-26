# Commands

## Overview

| Command | Description |
|---------|-------------|
| `opencore init [name]` | Initialize a new project |
| `opencore build` | Build all resources |
| `opencore dev` | Development mode with hot-reload |
| `opencore create <type>` | Create scaffolding |
| `opencore clone <template>` | Clone official template |
| `opencore doctor` | Validate configuration |
| `opencore update` | Update the CLI |
| `opencore --version` | Display CLI version |

## init

Initialize a new OpenCore project with interactive wizard.

```bash
opencore init my-server
```

## build

Build all resources for production.

```bash
opencore build
```

Options:
- Uses configuration from `opencore.config.ts`
- Outputs to `destination` path
- Runs parallel if `build.parallel: true`
- `--output auto|tui|plain` controls output mode (default: `auto`)

CI usage:

```bash
opencore build --output=plain
```

## dev

Start development mode with file watching and hot-reload.

```bash
opencore dev
```

Features:
- Watches for file changes
- Incremental compilation
- Hot-reload via framework HTTP server
- Optional txAdmin integration for core reload

## create

Generate scaffolding for project components.

### feature

```bash
# Create feature in core
opencore create feature banking

# Create feature in specific resource
opencore create feature chat -r myresource
```

### resource

```bash
# Create satellite resource
opencore create resource admin --with-client --with-nui
```

### standalone

```bash
# Create standalone resource
opencore create standalone utils --with-client
```

## clone

Download official templates from the repository.

```bash
# List available templates
opencore clone --list

# List templates from a development branch
opencore clone --list --branch develop

# Clone a template
opencore clone chat

# Clone from a development branch
opencore clone chat --branch develop

# Force GitHub API (skip git sparse-checkout)
opencore clone admin --api
```

## doctor

Validate project configuration and check for issues.

```bash
opencore doctor
```

Checks:
- Configuration file exists and is valid
- Required paths exist
- Dependencies are compatible

## update

Update the CLI from the selected release channel.

```bash
opencore update
opencore update --channel beta
```

Options:
- `--channel stable|beta` selects which release stream to check
- `OPENCORE_UPDATE_CHANNEL=beta` changes the default channel for update checks
