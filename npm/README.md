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

## FiveM Runtime Limitations

FiveM uses a **neutral JavaScript runtime**:

- **NO Node.js APIs**: `fs`, `path`, `http`, etc. not available
- **NO Web APIs**: `DOM`, `fetch`, `localStorage`, etc. not available
- **NO native C++ packages**: Use pure JS alternatives (`bcryptjs`, `jimp`, `sql.js`)

**Client**: All dependencies bundled into single `.js` file (no `external` support)

**Server**: Can use `external` packages, but bundling everything is recommended

## Documentation

Full documentation: https://github.com/newcore-network/opencore-cli

## License

MPL-2.0
