## OpenCore CLI v1.5.0

### Dependency Resolver

Added `build.dependencyResolution` for server resources that use `build.server.external`.

This is mainly for FiveM/RedM resources that need runtime npm packages such as `pg`, `typeorm`, database clients, or Prisma adapters while staying inside the FXServer Node.js 22 filesystem sandbox.

### Modes

- `auto`: uses OpenCore's runtime default. For FiveM/RedM this currently resolves to `isolated`.
- `isolated`: installs physical resource-local dependencies. This is the recommended production mode.
- `symlink`: legacy opt-in mode for linking `node_modules`; not recommended for FXServer sandboxed runtimes.
- `shared-resource`: experimental shared dependency resource.
- `bundle`: experimental bundling mode for pure JavaScript packages.

### Isolated Installs

`isolated` mode now writes a minimal `package.json` into each built resource and installs only the external packages that resource actually uses.

The resolver refuses unsafe deploy specs such as `latest`, `*`, `workspace:`, `file:`, `link:`, `portal:`, and local paths.

Sandbox validation is enabled by default and rejects symlinks that resolve outside the generated resource.

### pnpm Runtime Layout Fix

pnpm installs now use a hoisted physical layout for generated dependency installs:

```sh
pnpm install --prod --ignore-workspace --no-lockfile --reporter=append-only --config.node-linker=hoisted --package-import-method=copy
```

This fixes transitive runtime imports under FXServer, for example `typeorm` resolving `tslib` from the built resource instead of climbing to the workspace root and failing the sandbox check.

### Dependency Cache

Generated dependency trees are cached under:

```text
node_modules/.cache/opencore/dependencies/<hash>
```

Matching resources reuse the cached physical dependency tree when `.opencore-deps.json` matches the dependency set, package manager, install-script setting, pnpm linker mode, platform, architecture, and Node major version.

### Native Package Checks

The resolver detects known native packages and native package indicators such as `binding.gyp`, `prebuilds`, `.node` files, and native package metadata.

Native packages are rejected in `bundle` mode and reported clearly for dependency-resolution flows.

### Docs

Added dependency resolver documentation and FiveM sandbox guidance:

- `docs/cli/dependency-resolution.md`
- `docs/adapters/fivem.md`

Use `isolated` for production FiveM/RedM server resources that require runtime npm dependencies.

### Bug Fixes

**Dev Mode UI Hot-Reload (#16):** Fixed an issue where `server.js` and `client.js` would disappear from the resource output directory when editing UI files (`.tsx`, `.jsx`) in dev mode. The incremental build now preserves sibling artifacts when rebuilding only views, instead of wiping the entire resource directory before the partial rebuild.
