## OpenCore CLI v0.5.2

### Highlights
- **Platform-Specific Binary Support**: Introduced `serverBinaryPlatform` and `serverBinaries` configuration for automated, platform-aware management of server-side binaries (Windows, Linux, Darwin).
- **Astro Framework Support**: Added support for Astro in views with static-only output mode, including custom build commands and automatic framework detection.

### Changes
- **Server Binaries Management**: 
    - Added `serverBinaryPlatform` option to support platform-specific binary selection from `bin/<platform>` folders.
    - Added `serverBinaries` configuration to explicitly list paths for copying server-side binaries.
    - Implemented automatic detection and copying of the `bin/` directory if present and not otherwise specified.
- **Astro Integration**:
    - Added 'astro' to framework options in `ViewsConfig`.
    - Added `buildCommand` and `outputDir` fields to `ViewsConfig` for advanced static framework customization.
- **Static Asset Control**:
    - Added `forceInclude` option to `ViewsConfig` to explicitly include static files by filename pattern, even if they aren't imported in JS/CSS.

### Fixes
- Improved platform normalization and auto-detection logic for binary resolution.
- Enhanced reliability of static asset inclusion during the build process.