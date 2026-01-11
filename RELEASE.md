## OpenCore CLI v0.4.10

### Highlights
- New Sass/CSS support with integrated alias resolution for cleaner and more flexible imports
- Improved developer experience for `create` commands with unified prompts and consistent output

### Changes
- Added support for Sass and CSS, including a new alias resolver plugin for build-time path resolution
- Refactored all `create` commands to use shared prompt logic and standardized output formatting
- Introduced reusable helpers for name resolution, prompt configuration, and result rendering

### Fixes
- Removed duplicated validation and prompt logic across create flows
- Eliminated inconsistent success messages between feature, resource, and standalone creation
- Test `resource_builder_test` fixed, to add dependencies
- CI: fix in push with `Trusted Publisher` and tests added
- Updated all CI actions

### Notes
- No breaking changes
