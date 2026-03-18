const path = require('path')
const fs = require('fs')

let _esbuild
let _tsconfigPaths
function getEsbuild() {
    if (!_esbuild) {
        _esbuild = require('esbuild')
    }
    return _esbuild
}

let _swc
function getSwc() {
    if (!_swc) {
        _swc = require('@swc/core')
    }
    return _swc
}

function getTsconfigPaths() {
    if (!_tsconfigPaths) {
        try {
            _tsconfigPaths = require('tsconfig-paths')
        } catch (e) {
            _tsconfigPaths = null
        }
    }
    return _tsconfigPaths
}

function normalizeSwcTarget(target = 'es2020') {
    const value = String(target).toLowerCase()

    if (value.startsWith('node')) {
        switch (value) {
            case 'node14':
                return 'es2020'
            case 'node16':
                return 'es2021'
            case 'node18':
                return 'es2022'
            case 'node20':
            case 'node22':
            case 'node24':
                return 'es2023'
            default:
                return 'es2020'
        }
    }

    return value
}

function createSwcPlugin(target = 'es2020') {
    const swc = getSwc()
    const swcTarget = normalizeSwcTarget(target)
    return {
        name: 'swc-custom',
        setup(build) {
            build.onLoad({ filter: /\.(ts|tsx)$/ }, async (args) => {
                if (args.path.includes('node_modules')) {
                    return null
                }

                const source = await fs.promises.readFile(args.path, 'utf8')
                const isTsx = args.path.endsWith('.tsx')

                try {
                    const result = await swc.transform(source, {
                        jsc: {
                            parser: {
                                syntax: 'typescript',
                                tsx: isTsx,
                                decorators: true,
                                dynamicImport: true,
                            },
                            transform: {
                                legacyDecorator: true,
                                decoratorMetadata: true,
                            },
                            target: swcTarget,
                            keepClassNames: true,
                        },
                        filename: args.path,
                        sourceMaps: false,
                    })

                    return {
                        contents: result.code,
                        loader: 'js',
                    }
                } catch (e) {
                    console.error(`[SWC] Error compiling ${args.path}:`, e)
                    return { errors: [{ text: e.message }] }
                }
            })
        },
    }
}

function createExcludeNodeAdaptersPlugin(isServerBuild) {
    return {
        name: 'exclude-node-adapters',
        setup(build) {
            if (isServerBuild) {
                return
            }
            build.onResolve({ filter: /^node:/ }, args => ({
                path: args.path,
                external: true
            }))
            build.onLoad({ filter: /[/\\]adapters[/\\]node[/\\]/ }, () => ({
                contents: 'module.exports = {};',
                loader: 'js'
            }))
            build.onLoad({ filter: /[/\\]runtime[/\\]client[/\\]adapter[/\\]node-[^/\\]+/ }, () => ({
                contents: 'module.exports = {};',
                loader: 'js'
            }))
        },
    }
}

function createExternalPackagesPlugin(externals = []) {
    if (!externals || externals.length === 0) {
        return {
            name: 'external-packages',
            setup() {}
        }
    }

    console.log(`[external-packages] Marking as external:`, externals)

    return {
        name: 'external-packages',
        setup(build) {
            build.onResolve({ filter: /.*/ }, (args) => {
                const importPath = args.path
                for (const pkg of externals) {
                    if (importPath === pkg || importPath.startsWith(pkg + '/')) {
                        console.log(`[external-packages] Marking as external: ${importPath}`)
                        return {
                            path: importPath,
                            external: true
                        }
                    }
                }
                return null
            })
        },
    }
}

const preserveFiveMExportsPlugin = {
    name: 'preserve-fivem-exports',
    setup(build) {
        // Replace exports() with globalThis.exports() DURING load phase
        // This prevents esbuild from renaming exports to exports2
        // Only apply to .js files in node_modules (already compiled)
        build.onLoad({ filter: /\.js$/ }, async (args) => {
            // Only transform files in node_modules that might use FiveM exports
            if (!args.path.includes('node_modules')) {
                return null
            }
            if (!args.path.includes('@open-core') && !args.path.includes('fivem')) {
                return null
            }

            try {
                let contents = await fs.promises.readFile(args.path, 'utf8')
                const originalContents = contents

                // Replace FiveM exports() calls with globalThis.exports()
                // Pattern 1: exports("name", handler) or exports('name', handler)
                contents = contents.replace(/\bexports\s*\(\s*["'`]/g, 'globalThis.exports("')
                // Pattern 2: exports(variableName, handler)
                contents = contents.replace(/\bexports\s*\(\s*([a-zA-Z_$][a-zA-Z0-9_$]*)\s*,/g, 'globalThis.exports($1,')

                if (contents !== originalContents) {
                    return { contents, loader: 'js' }
                }
            } catch (err) {
                // Ignore read errors, let esbuild handle them
            }
            return null
        })
    },
}

/**
 * Adds shims for Node.js globals like __dirname and __filename.
 * This is implemented as an onLoad plugin to ensure replacements happen
 * during the transformation phase, which is more robust for nested scopes.
 */
function createNodeGlobalsShimPlugin(format) {
    return {
        name: 'node-globals-shim',
        setup(build) {
            // For ESM, we still use the post-build shim because it needs imports
            if (format === 'esm') {
                build.onEnd(async (result) => {
                    if (result.errors.length > 0) return
                    const outfile = build.initialOptions.outfile
                    if (!outfile) return
                    try {
                        let contents = await fs.promises.readFile(outfile, 'utf8')
                        if (!contents.includes('fileURLToPath(import.meta.url)')) {
                            const shim = `
import { fileURLToPath as __fileURLToPath } from 'url';
import { dirname as __pathDirname } from 'path';
const __filename = __fileURLToPath(import.meta.url);
const __dirname = __pathDirname(__filename);
`
                            await fs.promises.writeFile(outfile, shim + contents, 'utf8')
                        }
                    } catch (err) {
                        console.error(`[node-globals-shim] Error:`, err)
                    }
                })
                return
            }

            // For CJS/IIFE, we use a simple text replacement during load
            // this handles cases where bcrypt or other libs use __dirname
            build.onLoad({ filter: /\.(js|ts|tsx|jsx)$/ }, async (args) => {
                if (args.path.includes('node_modules')) {
                    // We don't want to re-transform node_modules with SWC here,
                    // but we DO want to fix the __dirname references.
                    // However, esbuild's define is the "right" way if it wasn't for the syntax error.
                    return null 
                }
                return null
            })
        }
    }
}

/**
 * Alternative approach: use esbuild 'define' but with valid identifiers.
 * We will define a proxy that we inject via banner.
 */
function getNodeGlobalsDefine() {
    return {
        '__dirname': 'globalThis.__dirname',
        '__filename': 'globalThis.__filename'
    }
}

/**
 * Creates an esbuild plugin that resolves TypeScript path aliases from tsconfig.json.
 * Uses the tsconfig-paths library to handle path resolution including wildcards.
 * @param {string} resourcePath - Path to the resource directory containing tsconfig.json
 * @returns {object|null} esbuild plugin or null if tsconfig-paths is not available
 */
function createTsconfigPathsPlugin(resourcePath) {
    const tsconfigPaths = getTsconfigPaths()
    if (!tsconfigPaths) {
        return null
    }

    // Load tsconfig from the resource directory
    const configLoaderResult = tsconfigPaths.loadConfig(resourcePath)

    if (configLoaderResult.resultType === 'failed') {
        // No tsconfig.json found or no paths configured - this is fine, just skip
        return null
    }

    const { absoluteBaseUrl, paths } = configLoaderResult

    // If no paths are configured, skip plugin
    if (!paths || Object.keys(paths).length === 0) {
        return null
    }

    const matchPath = tsconfigPaths.createMatchPath(absoluteBaseUrl, paths)
    const extensions = ['.ts', '.tsx', '.js', '.jsx', '.json']
    // Also check for index files in directories
    const indexExtensions = ['/index.ts', '/index.tsx', '/index.js', '/index.jsx']

    console.log(`[tsconfig-paths] Loaded path aliases from ${resourcePath}`)

    /**
     * Find the actual file path with extension
     * matchPath returns base path without extension, we need to find the real file
     */
    function resolveWithExtension(basePath) {
        // First check if the path already has an extension and exists
        if (fs.existsSync(basePath)) {
            const stat = fs.statSync(basePath)
            if (stat.isFile()) {
                return basePath
            }
            // If it's a directory, try index files
            if (stat.isDirectory()) {
                for (const ext of indexExtensions) {
                    const indexPath = basePath + ext.slice(1) // Remove leading /
                    if (fs.existsSync(indexPath)) {
                        return indexPath
                    }
                }
            }
        }

        // Try adding extensions
        for (const ext of extensions) {
            const fullPath = basePath + ext
            if (fs.existsSync(fullPath)) {
                return fullPath
            }
        }

        // Try index files (for directory imports like '@/components')
        for (const ext of indexExtensions) {
            const fullPath = basePath + ext
            if (fs.existsSync(fullPath)) {
                return fullPath
            }
        }

        return null
    }

    return {
        name: 'tsconfig-paths',
        setup(build) {
            // Only intercept imports that look like aliases (start with @ or are not relative/absolute)
            build.onResolve({ filter: /^[^./]/ }, (args) => {
                // Skip node_modules resolution
                if (args.resolveDir.includes('node_modules')) {
                    return null
                }

                // Try to match the path using tsconfig paths
                const basePath = matchPath(args.path, undefined, undefined, extensions)

                if (basePath) {
                    // Resolve the actual file with extension
                    const resolvedPath = resolveWithExtension(basePath)
                    if (resolvedPath) {
                        return { path: resolvedPath }
                    }
                }

                return null
            })
        }
    }

}

function findProjectConfigPath(resourcePath) {
    if (!resourcePath) return null

    let current = path.resolve(resourcePath)
    while (true) {
        const candidate = path.join(current, 'opencore.config.ts')
        if (fs.existsSync(candidate)) {
            return candidate
        }

        const parent = path.dirname(current)
        if (parent === current) {
            return null
        }
        current = parent
    }
}

function getPackageManagerValue(packageManager) {
    const pm = String(packageManager || '').toLowerCase()
    if (pm === 'pnpm' || pm === 'yarn' || pm === 'npm') return pm
    return 'pnpm'
}

function installCmd(pm, pkg, isDev) {
    const mgr = getPackageManagerValue(pm)
    if (mgr === 'yarn') return isDev ? `yarn add -D ${pkg}` : `yarn add ${pkg}`
    if (mgr === 'npm') return isDev ? `npm install -D ${pkg}` : `npm install ${pkg}`
    return isDev ? `pnpm add -D ${pkg}` : `pnpm add ${pkg}`
}

function createReflectMetadataPlugin(optionsOrPackageManager) {
    const options = typeof optionsOrPackageManager === 'object' && optionsOrPackageManager !== null
        ? optionsOrPackageManager
        : { packageManager: optionsOrPackageManager }
    const packageManager = options.packageManager
    const resourcePath = options.resourcePath || null
    const target = options.target || null
    const projectConfigPath = findProjectConfigPath(resourcePath)
    const adapterModulePath = target ? `opencore:project-adapter:${target}` : null
    const cliConfigHelperPath = 'opencore:config-helper'

    return {
        name: 'reflect-metadata-injector',
        setup(build) {
            if (adapterModulePath) {
                function buildSideAdapterModule(configSource, side) {
                    const sideRegex = new RegExp(`${side}\\s*:\\s*([^,\\n]+)`, 'm')
                    const sideMatch = configSource.match(sideRegex)
                    if (!sideMatch) {
                        return 'export const __openCoreProjectAdapter = undefined\n'
                    }

                    const expression = sideMatch[1].trim()
                    const identifiers = Array.from(new Set((expression.match(/[A-Za-z_$][\w$]*/g) || [])
                        .filter((name) => !['true', 'false', 'null', 'undefined'].includes(name))))

                    const importLines = []
                    for (const identifier of identifiers) {
                        const importRegex = new RegExp(`^\\s*import\\s+[^\\n]*\\b${identifier}\\b[^\\n]*$`, 'm')
                        const importMatch = configSource.match(importRegex)
                        if (importMatch) {
                            importLines.push(importMatch[0])
                        }
                    }

                    if (identifiers.length > 0 && importLines.length === 0) {
                        return null
                    }

                    return `${Array.from(new Set(importLines)).join('\n')}\nexport const __openCoreProjectAdapter = ${expression}\n`
                }

                build.onResolve({ filter: /^opencore:project-adapter:(server|client)$/ }, (args) => ({
                    path: args.path,
                    namespace: 'opencore-project-adapter',
                }))

                build.onLoad({ filter: /^opencore:project-adapter:(server|client)$/, namespace: 'opencore-project-adapter' }, async (args) => {
                    const side = args.path.endsWith(':server') ? 'server' : 'client'
                    if (!projectConfigPath) {
                        return {
                            contents: 'export const __openCoreProjectAdapter = undefined\n',
                            loader: 'js',
                        }
                    }

                    const configSource = await fs.promises.readFile(projectConfigPath, 'utf8')
                    const sideModule = buildSideAdapterModule(configSource, side)
                    if (sideModule) {
                        return {
                            contents: sideModule,
                            loader: 'ts',
                            resolveDir: path.dirname(projectConfigPath),
                        }
                    }

                    return {
                        contents: `import projectConfig from ${JSON.stringify('./' + path.basename(projectConfigPath))}\nexport const __openCoreProjectAdapter = projectConfig?.adapter?.${side}\n`,
                        loader: 'js',
                        resolveDir: path.dirname(projectConfigPath),
                    }
                })

                build.onResolve({ filter: /^@open-core\/cli$/ }, (args) => {
                    if (!projectConfigPath) return null
                    if (path.resolve(args.importer) !== path.resolve(projectConfigPath)) {
                        return null
                    }

                    return {
                        path: cliConfigHelperPath,
                        namespace: 'opencore-config-helper',
                    }
                })

                build.onLoad({ filter: /^opencore:config-helper$/, namespace: 'opencore-config-helper' }, async () => ({
                    contents: 'export function defineConfig(config) { return config }\n',
                    loader: 'js',
                }))
            }

            // Force reflect-metadata to be bundled even if marked as external
            build.onResolve({ filter: /^reflect-metadata$/ }, () => {
                try {
                    return { path: require.resolve('reflect-metadata'), external: false }
                } catch (e) {
                    const cmd = installCmd(packageManager, 'reflect-metadata', false)
                    return { errors: [{ text: `reflect-metadata not found. Please install it with: ${cmd}` }] }
                }
            })

            // Inject import at the top of the entry point
            build.onLoad({ filter: /\.(ts|tsx|js|jsx)$/ }, async (args) => {
                // Skip node_modules
                if (args.path.includes('node_modules')) return null

                // Only inject into the main entry points (client/server main files)
                const isEntry = build.initialOptions.entryPoints.some(e => 
                    path.resolve(e) === path.resolve(args.path)
                )

                if (!isEntry) return null

                try {
                    const contents = await fs.promises.readFile(args.path, 'utf8')
                    const lines = []
                    let transformedContents = contents
                    if (!contents.includes('reflect-metadata')) {
                        lines.push(`import 'reflect-metadata';`)
                    }

                    if (adapterModulePath && !contents.includes(adapterModulePath)) {
                        lines.push(`import { __openCoreProjectAdapter } from ${JSON.stringify(adapterModulePath)};`)
                        lines.push(`const __openCoreInitWithAdapter = (runtimeApi, adapter, options = {}) => {
  if (!adapter) {
    return runtimeApi.init(options);
  }
  if (options && typeof options === 'object' && !Array.isArray(options) && 'adapter' in options && options.adapter) {
    return runtimeApi.init(options);
  }
  const normalizedOptions = options && typeof options === 'object' && !Array.isArray(options)
    ? { ...options, adapter }
    : { adapter };
  return runtimeApi.init(normalizedOptions);
};`)

                        if (target === 'server') {
                            transformedContents = transformedContents.replace(
                                /\bServer\.init\s*\(/g,
                                '__openCoreInitWithAdapter(Server, __openCoreProjectAdapter, '
                            )
                        } else {
                            transformedContents = transformedContents.replace(
                                /\bClient\.init\s*\(/g,
                                '__openCoreInitWithAdapter(Client, __openCoreProjectAdapter, '
                            )
                        }
                    }

                    if (lines.length === 0) return null

                    const ext = path.extname(args.path).slice(1)
                    // If it's TS, use 'ts' or 'tsx' loader, otherwise esbuild will fail
                    const loader = ext === 'ts' || ext === 'tsx' ? ext : 'js'

                    return {
                        contents: `${lines.join('\n')}\n${transformedContents}`,
                        loader: loader,
                    }
                } catch (e) {
                    return null
                }
            })
        },
    }
}

function createAutoloadDynamicImportShimPlugin() {
    return {
        name: 'opencore-autoload-dynamic-import-shim',
        setup(build) {
            build.onLoad({ filter: /\.js$/ }, async (args) => {
                if (!args.path.includes('node_modules')) {
                    return null
                }
                if (!args.path.includes(`${path.sep}@open-core${path.sep}framework${path.sep}dist${path.sep}runtime${path.sep}`)) {
                    return null
                }

                let contents
                try {
                    contents = await fs.promises.readFile(args.path, 'utf8')
                } catch (e) {
                    return null
                }

                const original = contents
                contents = contents.replace(
                    /\bimport\s*\(\s*(['"`])([^'"`]*autoload\.(?:server|client)\.controllers?[^'"`]*)\1\s*\)/g,
                    (m, q, spec) => `Promise.resolve().then(() => require(${q}${spec}${q}))`
                )

                if (contents === original) {
                    return null
                }

                return { contents, loader: 'js' }
            })
        },
    }
}

function createAutoloadControllersRedirectPlugin(resourcePath) {
    return {
        name: 'opencore-autoload-controllers-redirect',
        setup(build) {
            if (!resourcePath) return

            const serverCandidates = [
                path.resolve(resourcePath, '.opencore', 'autoload.server.controllers.ts'),
                path.resolve(resourcePath, '.opencore', 'autoload.server.controller.ts'),
                path.resolve(resourcePath, 'src', '.opencore', 'autoload.server.controllers.ts'),
                path.resolve(resourcePath, 'src', '.opencore', 'autoload.server.controller.ts'),
            ]

            const clientCandidates = [
                path.resolve(resourcePath, '.opencore', 'autoload.client.controllers.ts'),
                path.resolve(resourcePath, '.opencore', 'autoload.client.controller.ts'),
                path.resolve(resourcePath, 'src', '.opencore', 'autoload.client.controllers.ts'),
                path.resolve(resourcePath, 'src', '.opencore', 'autoload.client.controller.ts'),
            ]

            const pickFirstExisting = (candidates) => {
                for (const c of candidates) {
                    if (fs.existsSync(c)) return c
                }
                return null
            }

            build.onResolve({ filter: /autoload\.server\.controllers?(\.(ts|js))?$/ }, (args) => {
                // Only redirect when the framework runtime is trying to load its own stub
                // Example bundled path:
                // @open-core/framework/dist/runtime/server/.opencore/autoload.server.controllers.js
                if (!args.resolveDir.includes(`${path.sep}@open-core${path.sep}framework${path.sep}dist${path.sep}runtime${path.sep}server`)) {
                    return null
                }

                const target = pickFirstExisting(serverCandidates)
                if (!target) return null
                return { path: target }
            })

            build.onResolve({ filter: /autoload\.client\.controllers?(\.(ts|js))?$/ }, (args) => {
                if (!args.resolveDir.includes(`${path.sep}@open-core${path.sep}framework${path.sep}dist${path.sep}runtime${path.sep}client`)) {
                    return null
                }

                const target = pickFirstExisting(clientCandidates)
                if (!target) return null
                return { path: target }
            })
        },
    }
}

module.exports = {
    getEsbuild,
    createSwcPlugin,
    createExcludeNodeAdaptersPlugin,
    createExternalPackagesPlugin,
    preserveFiveMExportsPlugin,
    createNodeGlobalsShimPlugin,
    createTsconfigPathsPlugin,
    createReflectMetadataPlugin,
    createAutoloadDynamicImportShimPlugin,
    createAutoloadControllersRedirectPlugin
}
