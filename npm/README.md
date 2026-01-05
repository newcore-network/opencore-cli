# @open-core/cli

Command-line interface for the OpenCore Framework. Build and manage FiveM TypeScript projects.

## Installation

```bash
npm install -g @open-core/cli
```

## Commands

```bash
opencore init [name]    # Initialize a new project
opencore build          # Build all resources
opencore dev            # Development mode with hot-reload
opencore doctor         # Validate configuration
```

## Quick Start

```bash
opencore init my-server
cd my-server
pnpm install
opencore dev
```

## FiveM Runtime Environments

| | Server | Client | Views |
|--|--------|--------|-------|
| Runtime | Node.js | Neutral | Browser |
| FiveM APIs | Yes | Yes | Callbacks |
| GTA Natives | No | Yes | No |
| External pkgs | Yes | No | N/A |

**Server**: Node.js with all APIs. **Client**: Neutral JS + GTA natives. **Views**: Chromium browser.

## Documentation

Full documentation: https://github.com/newcore-network/opencore-cli

## License

MPL-2.0
