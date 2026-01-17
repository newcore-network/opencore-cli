## OpenCore CLI v0.5.0

### Highlights
- **New Architecture Support**: Added "No-Architecture" mode for minimal projects (simple `server.ts` and `client.ts` in `core/src`).
- **Improved Scaffolding**: Refactored layer-based templates for better server/client separation and consistent dependency injection.
- **Enhanced UI Support**: Added auto-detection and plugin support for Svelte and Vue frameworks.
- **Styling Improvements**: Added support for SASS/CSS and custom alias resolver plugins.

### Changes
- **No-Architecture Mode**: New CLI option to generate projects without complex folder structures.
- **Layer-Based Refactor**: Controllers now correctly inject services by default for better architectural separation.
- **Bootstrap Updates**: Updated `server.ts` and `client.ts` templates with clearer examples and better initialization logic.
- **NPM & Go Updates**: Updated Go version to 1.24.5 and optimized NPM package structure.
- **Documentation**: Comprehensive CLI documentation added and README updated.

### Fixes
- **Update Cache Removed**: Removed `update.json` internal cache to prevent versioning issues and ensure real-time update checks.
- **Template Typos**: Fixed critical typos in templates (e.g., `source` vs `player.source` in controllers).
- **Framework Detection**: Improved framework detection logic for UI builds.

### Notes
- Version v0.4.11 was skipped in favor of v0.5.0 to reflect major architectural improvements.
- No breaking changes for existing projects.
