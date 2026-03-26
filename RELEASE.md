## OpenCore CLI v1.2.0

### Added
- Added template manifest support with `oc.manifest.json`, including runtime compatibility (`fivem`, `redm`, `ragemp`), game profiles (`common`, `gta5`, `rdr3`), template dependencies, and the schema file at `schemas/oc-manifest.schema.json`.
- Added `opencore create manifest` to generate starter `oc.manifest.json` files for `core/`, `resources/<name>/`, and `standalones/<name>/` with inferred compatibility defaults.
- Added `opencore adapter check` to validate external adapter packages against the framework contract baseline, with `--strict` and `--json` output modes.
- Added configurable `opencore dev` restart strategies through `dev.restart.mode` with support for `auto`, `process`, `txadmin`, and `none`.
- Added structured dev settings for `dev.bridge`, `dev.txAdmin`, and `dev.process` so local executables and txAdmin restarts can be managed from config.
- Added JSX/TSX support to views builds, plus `postcss-nesting` support in the embedded Tailwind/PostCSS pipeline.

### Changed
- `opencore clone --list` now shows manifest compatibility details when a template provides `oc.manifest.json`.
- `opencore clone <template>` now validates template/runtime compatibility against the current project's `opencore.config.ts`, with `--force` available to bypass the check.
- Cloning into RageMP now removes copied `fxmanifest.lua` files automatically so CFX-specific manifests are not left inside RageMP resources.
- Templates without a manifest remain supported and are treated as compatibility `unknown` for backward compatibility.
- Views builds now support Vite-based projects with explicit framework opt-in while keeping non-Vite projects on the embedded build path.
- New starter projects and config documentation now reflect adapter-aware runtime defaults, nested dev settings, release channels, and RageMP-oriented examples.

### Improved
- RageMP support is more complete across scaffolding and builds, including better server/client output defaults and runtime-specific build targets.
- Resource builds can now copy server-side binary folders more reliably, including platform-specific `bin/<platform>` layouts.
- npm/binary installation handling is safer: existing local binaries are reused when possible, and executable permissions are enforced on Unix systems.
- The publish/update flow now supports release channels, including `stable` and `beta`, in both the CLI updater and npm publishing workflow.

### Fixed
- Fixed adapter module config parsing by stripping comments before parsing source content.
- Fixed generated views output to inject a missing `<link>` tag when esbuild emits a CSS file that is not already referenced in HTML.
- Fixed Vite views builds failing when no explicit framework was configured.
- Fixed several packaging and repository defaults, including starter `.gitignore` cleanup and release workflow metadata updates.

### Notes
- This release covers all changes currently present on `develop` relative to `master`.
- The manifest file is optional, but recommended for official and shared templates because clone-time compatibility checks now use it when available.
