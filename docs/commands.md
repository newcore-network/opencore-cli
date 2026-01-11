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
| `opencore version` | Display CLI version |

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
opencore create feature chat -r myserver
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

# Clone a template
opencore clone chat

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
