const esbuild = require('esbuild')
const { swcPlugin } = require('esbuild-plugin-swc')
const path = require('path')
const fs = require('fs')

// =============================================================================
// SWC Plugin Configuration for TypeScript Decorators
// =============================================================================

// Node.js built-ins that must be external (FiveM doesn't have these)
const nodeBuiltins = [
    'node:events', 'node:fs', 'node:path', 'node:util', 'node:crypto',
    'node:stream', 'node:buffer', 'node:http', 'node:https', 'node:net',
    'node:os', 'node:child_process', 'node:worker_threads', 'node:url',
    'events', 'fs', 'path', 'util', 'crypto', 'stream', 'buffer',
    'http', 'https', 'net', 'os', 'child_process', 'worker_threads', 'url'
]

// SWC configuration for TypeScript decorators and metadata
const swcConfig = swcPlugin({
    jsc: {
        parser: {
            syntax: 'typescript',
            decorators: true,
            dynamicImport: true,
        },
        transform: {
            legacyDecorator: true,
            decoratorMetadata: true,
        },
        target: 'es2020',
        keepClassNames: true,
        preserveAllComments: false,
    },
    minify: false, // Let esbuild handle minification
})

// =============================================================================
// Plugins
// =============================================================================

// Excludes Node.js adapters and builtins from the bundle
const excludeNodeAdaptersPlugin = {
    name: 'exclude-node-adapters',
    setup(build) {
        // Mark node: prefixed imports as external
        build.onResolve({ filter: /^node:/ }, args => ({
            path: args.path,
            external: true
        }))

        // Mark bare node builtins as external  
        build.onResolve({ filter: /^(events|fs|path|util|crypto|stream|buffer|http|https|net|os|child_process|worker_threads|url)$/ }, args => ({
            path: args.path,
            external: true
        }))

        // Exclude node adapter files from framework (return empty module)
        build.onLoad({ filter: /[/\\]adapters[/\\]node[/\\]/ }, () => ({
            contents: 'module.exports = {};',
            loader: 'js'
        }))
    },
}

// Patches exports() calls for FiveM compatibility
const preserveFiveMExportsPlugin = {
    name: 'preserve-fivem-exports',
    setup(build) {
        build.onEnd(async (result) => {
            if (result.errors.length > 0) return
            const outfile = build.initialOptions.outfile
            if (!outfile) return

            try {
                let contents = await fs.promises.readFile(outfile, 'utf8')
                const originalContents = contents
                // Replace exports() with globalThis.exports() for FiveM
                contents = contents.replace(/\bexports\s*\(\s*["'`]/g, 'globalThis.exports("')
                contents = contents.replace(/\bexports\s*\(\s*([a-zA-Z_$][a-zA-Z0-9_$]*)\s*,/g, 'globalThis.exports($1,')
                if (contents !== originalContents) {
                    await fs.promises.writeFile(outfile, contents, 'utf8')
                }
            } catch (err) {
                console.error(`[preserve-fivem-exports] Error:`, err)
            }
        })
    },
}

// =============================================================================
// Build Configurations
// =============================================================================

function getSharedConfig(options = {}) {
    return {
        bundle: true,
        sourcemap: options.sourceMaps ? 'inline' : false,
        minifyWhitespace: options.minify !== false,
        minifySyntax: options.minify !== false,
        minifyIdentifiers: false, // Keep identifiers for DI/reflection
        keepNames: true, // Critical for tsyringe class name resolution
        treeShaking: true,
        logLevel: 'info',
        legalComments: 'none',
    }
}

// FiveM-specific esbuild options
function getFiveMBuildOptions() {
    return {
        platform: 'neutral', // FiveM is NOT Node.js - use browser platform
        target: 'es2020',
        format: 'iife',
        mainFields: ['module', 'main'], // Prefer ES modules
        conditions: ['import', 'default'],
        supported: {
            'dynamic-import': true,
        },
    }
}

function getCorePlugins() {
    return [swcConfig, excludeNodeAdaptersPlugin, preserveFiveMExportsPlugin]
}

function getResourcePlugins() {
    return [swcConfig, excludeNodeAdaptersPlugin, preserveFiveMExportsPlugin]
}

function getStandalonePlugins() {
    return [swcConfig, excludeNodeAdaptersPlugin]
}

// =============================================================================
// Build Functions
// =============================================================================

/**
 * Build core resource (full framework with DI, remotes, etc.)
 */
async function buildCore(resourcePath, outDir, options = {}) {
    const shared = getSharedConfig(options)
    const fivemOptions = getFiveMBuildOptions()
    const plugins = getCorePlugins()

    const serverEntry = options.entryPoints?.server || path.join(resourcePath, 'src/server.ts')
    const clientEntry = options.entryPoints?.client || path.join(resourcePath, 'src/client.ts')

    const resourceName = path.basename(resourcePath)
    const resourceOutDir = path.join(outDir, resourceName)

    await fs.promises.mkdir(resourceOutDir, { recursive: true })

    const builds = []

    // Server build
    if (options.server !== false && fs.existsSync(serverEntry)) {
        builds.push(esbuild.build({
            ...shared,
            ...fivemOptions,
            entryPoints: [serverEntry],
            outfile: path.join(resourceOutDir, 'server.js'),
            plugins,
            external: nodeBuiltins,
        }))
    }

    // Client build
    if (options.client !== false && fs.existsSync(clientEntry)) {
        builds.push(esbuild.build({
            ...shared,
            ...fivemOptions,
            entryPoints: [clientEntry],
            outfile: path.join(resourceOutDir, 'client.js'),
            plugins,
            external: nodeBuiltins,
        }))
    }

    // Copy fxmanifest.lua
    const manifestSrc = path.join(resourcePath, 'fxmanifest.lua')
    const manifestDst = path.join(resourceOutDir, 'fxmanifest.lua')
    if (fs.existsSync(manifestSrc)) {
        await fs.promises.copyFile(manifestSrc, manifestDst)
    }

    await Promise.all(builds)
    console.log(`[core] Built ${resourceName}`)
}

/**
 * Build satellite resource (depends on core exports at runtime)
 */
async function buildResource(resourcePath, outDir, options = {}) {
    const shared = getSharedConfig(options)
    const fivemOptions = getFiveMBuildOptions()
    const plugins = getResourcePlugins()

    const resourceName = path.basename(resourcePath)
    const resourceOutDir = path.join(outDir, resourceName)

    await fs.promises.mkdir(resourceOutDir, { recursive: true })

    const builds = []

    // Server build
    const serverEntry = options.entryPoints?.server || path.join(resourcePath, 'src/server/main.ts')
    if (options.server !== false && fs.existsSync(serverEntry)) {
        builds.push(esbuild.build({
            ...shared,
            ...fivemOptions,
            entryPoints: [serverEntry],
            outfile: path.join(resourceOutDir, 'server.js'),
            plugins,
            external: nodeBuiltins,
        }))
    }

    // Client build
    const clientEntry = options.entryPoints?.client || path.join(resourcePath, 'src/client/main.ts')
    if (options.client !== false && fs.existsSync(clientEntry)) {
        builds.push(esbuild.build({
            ...shared,
            ...fivemOptions,
            entryPoints: [clientEntry],
            outfile: path.join(resourceOutDir, 'client.js'),
            plugins,
            external: nodeBuiltins,
        }))
    }

    // Copy fxmanifest.lua
    const manifestSrc = path.join(resourcePath, 'fxmanifest.lua')
    const manifestDst = path.join(resourceOutDir, 'fxmanifest.lua')
    if (fs.existsSync(manifestSrc)) {
        await fs.promises.copyFile(manifestSrc, manifestDst)
    }

    if (builds.length > 0) {
        await Promise.all(builds)
    }
    console.log(`[resource] Built ${resourceName}`)
}

/**
 * Build standalone resource (no core dependency, basic decorator support)
 */
async function buildStandalone(resourcePath, outDir, options = {}) {
    const shared = getSharedConfig(options)
    const fivemOptions = getFiveMBuildOptions()
    const plugins = getStandalonePlugins()

    const resourceName = path.basename(resourcePath)
    const resourceOutDir = path.join(outDir, resourceName)

    await fs.promises.mkdir(resourceOutDir, { recursive: true })

    const builds = []

    // Server build
    const serverEntry = options.entryPoints?.server || path.join(resourcePath, 'src/server/main.ts')
    if (options.server !== false && fs.existsSync(serverEntry)) {
        builds.push(esbuild.build({
            ...shared,
            ...fivemOptions,
            entryPoints: [serverEntry],
            outfile: path.join(resourceOutDir, 'server.js'),
            plugins,
            external: nodeBuiltins,
        }))
    }

    // Client build
    const clientEntry = options.entryPoints?.client || path.join(resourcePath, 'src/client/main.ts')
    if (options.client !== false && fs.existsSync(clientEntry)) {
        builds.push(esbuild.build({
            ...shared,
            ...fivemOptions,
            entryPoints: [clientEntry],
            outfile: path.join(resourceOutDir, 'client.js'),
            plugins,
            external: nodeBuiltins,
        }))
    }

    // Copy fxmanifest.lua
    const manifestSrc = path.join(resourcePath, 'fxmanifest.lua')
    const manifestDst = path.join(resourceOutDir, 'fxmanifest.lua')
    if (fs.existsSync(manifestSrc)) {
        await fs.promises.copyFile(manifestSrc, manifestDst)
    }

    if (builds.length > 0) {
        await Promise.all(builds)
    }
    console.log(`[standalone] Built ${resourceName}`)
}

/**
 * Read .ocignore file and return patterns
 */
function readOcIgnore(viewPath) {
    const ocignorePath = path.join(viewPath, '.ocignore')
    if (!fs.existsSync(ocignorePath)) {
        return []
    }

    try {
        const content = fs.readFileSync(ocignorePath, 'utf8')
        return content
            .split('\n')
            .map(line => line.trim())
            .filter(line => line && !line.startsWith('#')) // Remove empty lines and comments
    } catch (error) {
        console.warn(`[views] Failed to read .ocignore: ${error.message}`)
        return []
    }
}

/**
 * Build views/NUI (React, Vue, Svelte, etc.)
 */
async function buildViews(viewPath, outDir, options = {}) {
    const shared = getSharedConfig(options)

    await fs.promises.mkdir(outDir, { recursive: true })

    // Collect ignore patterns from config and .ocignore
    const ignorePatterns = [
        ...(options.ignore || []),
        ...readOcIgnore(viewPath),
        // Default ignores
        'node_modules',
        '.git',
        '.ocignore',
    ]

    console.log(`[views] Ignore patterns:`, ignorePatterns.length > 0 ? ignorePatterns : 'none')

    let entryPoint = null

    // 1. Check if explicit entry point is configured
    if (options.viewEntry) {
        const explicitEntry = path.join(viewPath, options.viewEntry)
        if (fs.existsSync(explicitEntry)) {
            entryPoint = explicitEntry
            console.log(`[views] Using explicit entry point: ${options.viewEntry}`)
        } else {
            throw new Error(`[views] Configured entry point not found: ${options.viewEntry}`)
        }
    }

    // 2. Auto-detect entry point if not explicitly configured
    if (!entryPoint) {
        const possibleEntries = [
            path.join(viewPath, 'index.tsx'),
            path.join(viewPath, 'index.jsx'),
            path.join(viewPath, 'index.ts'),
            path.join(viewPath, 'index.js'),
            path.join(viewPath, 'main.tsx'),
            path.join(viewPath, 'main.jsx'),
            path.join(viewPath, 'main.ts'),
            path.join(viewPath, 'main.js'),
            path.join(viewPath, 'app.tsx'),
            path.join(viewPath, 'app.jsx'),
            path.join(viewPath, 'app.ts'),
            path.join(viewPath, 'app.js'),
            path.join(viewPath, 'src/index.tsx'),
            path.join(viewPath, 'src/index.ts'),
            path.join(viewPath, 'src/main.tsx'),
            path.join(viewPath, 'src/main.ts'),
            path.join(viewPath, 'src/app.tsx'),
            path.join(viewPath, 'src/app.ts'),
        ]

        for (const entry of possibleEntries) {
            if (fs.existsSync(entry)) {
                entryPoint = entry
                break
            }
        }

        if (!entryPoint) {
            const errorMsg = `[views] No entry point found in ${viewPath}\nSearched for: ${possibleEntries.map(p => path.basename(p)).join(', ')}\nTip: Set 'entryPoint' in views config or create one of the above files.`
            console.error(errorMsg)
            throw new Error(errorMsg)
        }

        console.log(`[views] Auto-detected entry point: ${path.relative(viewPath, entryPoint)}`)
    }

    await esbuild.build({
        ...shared,
        entryPoints: [entryPoint],
        outdir: outDir,
        platform: 'browser',
        target: options.target || 'es2020',
        format: 'esm',
        splitting: true,
        loader: {
            '.tsx': 'tsx',
            '.jsx': 'jsx',
            '.css': 'css',
            '.svg': 'file',
            '.png': 'file',
            '.jpg': 'file',
            '.gif': 'file',
            '.woff': 'file',
            '.woff2': 'file',
        },
        define: {
            'process.env.NODE_ENV': options.minify ? '"production"' : '"development"',
        },
    })

    // Process and copy HTML file if exists
    const htmlSrc = path.join(viewPath, 'index.html')
    const htmlDst = path.join(outDir, 'index.html')
    if (fs.existsSync(htmlSrc)) {
        let html = await fs.promises.readFile(htmlSrc, 'utf8')

        // Replace TypeScript/JSX references with compiled JS
        // Match: <script ... src="app.ts"> or <script ... src="app.tsx">
        html = html.replace(
            /(<script[^>]*\ssrc=["'])([^"']+\.(ts|tsx|jsx))(['"][^>]*>)/gi,
            (match, prefix, src, ext, suffix) => {
                const jsFile = src.replace(/\.(ts|tsx|jsx)$/, '.js')
                return prefix + jsFile + suffix
            }
        )

        // Extract and copy referenced CSS files
        const cssMatches = html.matchAll(/<link[^>]*href=["']([^"']+\.css)["'][^>]*>/gi)
        for (const match of cssMatches) {
            const cssFile = match[1]
            const cssSrc = path.join(viewPath, cssFile)
            const cssDst = path.join(outDir, cssFile)

            if (fs.existsSync(cssSrc)) {
                await fs.promises.copyFile(cssSrc, cssDst)
                console.log(`[views] Copied ${cssFile}`)
            }
        }

        await fs.promises.writeFile(htmlDst, html, 'utf8')
        console.log(`[views] Processed and copied index.html`)
    }

    console.log(`[views] Built ${path.basename(viewPath)}`)
}

/**
 * Build a single resource by type (called from Go CLI)
 */
async function buildSingle(type, resourcePath, outDir, options = {}) {
    switch (type) {
        case 'core':
            return buildCore(resourcePath, outDir, options)
        case 'resource':
            return buildResource(resourcePath, outDir, options)
        case 'standalone':
            return buildStandalone(resourcePath, outDir, options)
        case 'views':
            return buildViews(resourcePath, outDir, options)
        default:
            throw new Error(`Unknown resource type: ${type}`)
    }
}

// =============================================================================
// CLI Entry Point (called from Go CLI)
// =============================================================================

async function main() {
    const args = process.argv.slice(2)
    const mode = args[0] || 'single'

    if (mode === 'single') {
        // Called from Go CLI: node build.js single <type> <path> <outDir> <options-json>
        const type = args[1]
        const resourcePath = args[2]
        const outDir = args[3]
        const options = args[4] ? JSON.parse(args[4]) : {}

        try {
            await buildSingle(type, resourcePath, outDir, options)
            console.log(JSON.stringify({ success: true }))
        } catch (error) {
            console.error(error.message)
            process.exit(1)
        }
    } else {
        console.error('Usage: node build.js single <type> <path> <outDir> [options-json]')
        process.exit(1)
    }
}

// Run if called directly
if (require.main === module) {
    main()
}
