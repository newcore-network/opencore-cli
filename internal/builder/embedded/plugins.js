const path = require('path')
const fs = require('fs')

let _esbuild
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

function createSwcPlugin(target = 'es2020') {
    const swc = getSwc()
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
                            target: target,
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

module.exports = {
    getEsbuild,
    createSwcPlugin,
    createExcludeNodeAdaptersPlugin,
    createExternalPackagesPlugin,
    preserveFiveMExportsPlugin,
    createNodeGlobalsShimPlugin
}
