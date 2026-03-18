## OpenCore CLI v1.1.0

### Added

- Add central adapter and runtime inspection to config loading, doctor output, and build defaults for FiveM and RageMP environments.
- Add runtime-aware scaffolding for `create resource` and `create standalone`, including manifest generation rules, runtime-specific TypeScript targets, and correct type packages.
- Add RedM manifest defaults in generated `fxmanifest.lua` files with `game 'rdr3'` and the required `rdr3_warning` directive.
- Add automated coverage for adapter detection, simplified feature scaffolding, and runtime-specific template generation.

### Changed

- Standardize new project generation on a single default layout with `core/src/server.ts`, `core/src/client.ts`, and `core/src/features/`.
- Remove architecture selection and all legacy architecture-specific generators and templates from `init` and `create feature` flows.
- Update documentation to reflect adapter-driven runtime behavior and the simplified default project structure.

### Improved

- Improve embedded adapter injection and runtime bootstrap handling during builds.
- Skip redundant binary downloads when `opencore-cli` is already available during installation.