## OpenCore CLI v1.0.0

### Highlights

- **Automatic controller autoloading**: automatic controller discovery and autoload import generation.
- **More robust CI/non-TTY builds**: new simple output mode for workflows (for example GitHub Actions).
- **Improved source/controller validation**: less ambiguity with generic decorators and framework imports.
- **Configurable package manager support**: npm, yarn, or pnpm across init/build.
- **Template and DX upgrades**: Node 22 + ES2022 target for generated templates.

### New Features

- **Controller autoload generation**
  - Added build-time autoload generation with automatic controller discovery.
  - Generates files in `.opencore`/`src/.opencore` depending on project context.
  - Skips autoload generation for view-only builds to avoid unnecessary work.
- **Autoload split server/client**
  - Autoload generation is now split by side (server/client) instead of a single combined file.
  - Improved dynamic import resolution plugin for autoload imports.
- **Package manager selection**
  - Added a unified flow to resolve and use `npm`, `yarn`, or `pnpm` consistently for install/run commands.
  - Integrated package manager selection with scaffolding and build commands.
- **Clone command enhancements**
  - Added `--branch` flag to `opencore clone` to list/clone templates from a specific branch.
- **No-TTY / workflow mode**
  - Added non-interactive environment detection and simple output mode.
  - Added explicit `build --output auto|tui|plain` support.
  - Improved CI behavior by avoiding TUI/spinner output in logs.

### Changes

- **Templates/runtime target**
  - Updated templates to Node.js 22 (`fxmanifest`).
  - Updated TypeScript target to `ES2022` and module mode to `preserve` for new projects.
- **Build validation**
  - Added validation to prevent invalid mixed server/client build configurations.
  - Strengthened controller detection to prevent ambiguous decorator usage.
- **Starter/project defaults**
  - Simplified starter project default configuration.
  - Updated `.gitignore` defaults to exclude generated `.opencore` files.

### Breaking Changes

- New projects now default to **Node 22** and **TypeScript ES2022** in templates.
- Autoload flow changed (server/client split), which may affect internal customizations relying on the previous combined layout.
- Decorator/controller validation is stricter and may flag previously tolerated ambiguous cases.

### Notes

- This release establishes the `v1.0` CLI foundation: reliable autoloading, safer builds, and better CI compatibility.
- Recommended for CI:
  - `opencore build --output=plain`
  - `OPENCORE_DISABLE_UPDATE_CHECK=1` for cleaner logs.
