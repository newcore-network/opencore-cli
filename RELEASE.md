## OpenCore CLI v1.2.0

### Added
- Added template manifest support through `oc.manifest.json` for official templates.
- Added compatibility metadata support for template runtimes (`fivem`, `redm`, `ragemp`) and game profiles (`common`, `gta5`, `rdr3`).
- Added `requires.templates` metadata to declare template dependencies.
- Added a JSON Schema reference at `schemas/oc-manifest.schema.json` for editor validation and contributor guidance.

### Clone Command
- `opencore clone --list` now shows compatibility information when a template provides `oc.manifest.json`.
- `opencore clone <template>` now validates manifest runtime compatibility against the current project's `opencore.config.ts` when available.
- Added `opencore clone --force` to bypass compatibility validation for advanced or experimental use cases.
- Templates without a manifest remain supported and are treated as compatibility `unknown` for backward compatibility.

### Create Command
- Added `opencore create manifest` to generate an example `oc.manifest.json` for existing `core/`, `resources/<name>/`, or `standalones/<name>/` directories.
- The generated manifest includes the public schema URL, a starter compatibility block inferred from the current project runtime when available, and a default `core` dependency for framework-connected resources.

### Manifest Format
Example `oc.manifest.json`:

```json
{
  "schemaVersion": 1,
  "name": "example-template",
  "displayName": "Example Template",
  "kind": "resource",
  "description": "Example OpenCore resource",
  "compatibility": {
    "runtimes": ["fivem", "redm"],
    "gameProfiles": ["common"]
  },
  "requires": {
    "templates": ["core"]
  }
}
```

### Notes
- The manifest is optional but recommended for all published templates.
- Runtime compatibility is currently enforced during clone; additional manifest-driven behaviors can build on this later without changing the file name or schema family.
