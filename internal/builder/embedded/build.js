const esbuild = require('esbuild')
const { swcPlugin } = require('esbuild-plugin-swc')
const path = require('path')
const fs = require('fs')

// =============================================================================
// SWC Plugin Configuration for TypeScript Decorators
// =============================================================================

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
        keepClassNames: true,
    },
})

// =============================================================================
// Plugins
// =============================================================================

// Excludes Node.js adapters from the bundle (FiveM compatibility)
const excludeNodeAdaptersPlugin = {
    name: 'exclude-node-adapters',
    setup(build) {
        const nodePath = require('path')
        build.onLoad({ filter: /node-.*\.(js|ts)$/ }, (args) => {
            if (args.path.includes('@open-core') &&
                (args.path.includes(nodePath.sep + 'node' + nodePath.sep) ||
                 args.path.includes('/node/'))) {
                return { contents: '', loader: 'js' }
            }
            return null
        })
    },
}

// Patches exports() for FiveM compatibility
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
        minifyIdentifiers: false,
        keepNames: true,
        logLevel: 'info',
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
            entryPoints: [serverEntry],
            outfile: path.join(resourceOutDir, 'server.js'),
            platform: 'neutral',
            target: options.target || 'es2020',
            format: 'iife',
            mainFields: ['main', 'module'],
            plugins,
        }))
    }

    // Client build
    if (options.client !== false && fs.existsSync(clientEntry)) {
        builds.push(esbuild.build({
            ...shared,
            entryPoints: [clientEntry],
            outfile: path.join(resourceOutDir, 'client.js'),
            platform: 'neutral',
            target: options.target || 'es2020',
            format: 'iife',
            mainFields: ['main', 'module'],
            plugins,
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
    const plugins = getResourcePlugins()

    const resourceName = path.basename(resourcePath)
    const resourceOutDir = path.join(outDir, resourceName)

    await fs.promises.mkdir(resourceOutDir, { recursive: true })

    const builds = []

    // Server build
    const serverEntry = path.join(resourcePath, 'src/server/main.ts')
    if (options.server !== false && fs.existsSync(serverEntry)) {
        builds.push(esbuild.build({
            ...shared,
            entryPoints: [serverEntry],
            outfile: path.join(resourceOutDir, 'server.js'),
            platform: 'neutral',
            target: options.target || 'es2020',
            format: 'iife',
            mainFields: ['main', 'module'],
            plugins,
            external: ['@open-core/framework'],
        }))
    }

    // Client build
    const clientEntry = path.join(resourcePath, 'src/client/main.ts')
    if (options.client !== false && fs.existsSync(clientEntry)) {
        builds.push(esbuild.build({
            ...shared,
            entryPoints: [clientEntry],
            outfile: path.join(resourceOutDir, 'client.js'),
            platform: 'neutral',
            target: options.target || 'es2020',
            format: 'iife',
            mainFields: ['main', 'module'],
            plugins,
            external: ['@open-core/framework'],
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
    const plugins = getStandalonePlugins()

    const resourceName = path.basename(resourcePath)
    const resourceOutDir = path.join(outDir, resourceName)

    await fs.promises.mkdir(resourceOutDir, { recursive: true })

    const builds = []

    // Server build
    const serverEntry = path.join(resourcePath, 'src/server/main.ts')
    if (options.server !== false && fs.existsSync(serverEntry)) {
        builds.push(esbuild.build({
            ...shared,
            entryPoints: [serverEntry],
            outfile: path.join(resourceOutDir, 'server.js'),
            platform: 'neutral',
            target: options.target || 'es2020',
            format: 'iife',
            mainFields: ['main', 'module'],
            plugins,
        }))
    }

    // Client build
    const clientEntry = path.join(resourcePath, 'src/client/main.ts')
    if (options.client !== false && fs.existsSync(clientEntry)) {
        builds.push(esbuild.build({
            ...shared,
            entryPoints: [clientEntry],
            outfile: path.join(resourceOutDir, 'client.js'),
            platform: 'neutral',
            target: options.target || 'es2020',
            format: 'iife',
            mainFields: ['main', 'module'],
            plugins,
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
 * Build views/NUI (React, Vue, Svelte, etc.)
 */
async function buildViews(viewPath, outDir, options = {}) {
    const shared = getSharedConfig(options)

    await fs.promises.mkdir(outDir, { recursive: true })

    // Detect entry point
    const possibleEntries = [
        path.join(viewPath, 'index.tsx'),
        path.join(viewPath, 'index.jsx'),
        path.join(viewPath, 'index.ts'),
        path.join(viewPath, 'index.js'),
        path.join(viewPath, 'main.tsx'),
        path.join(viewPath, 'main.jsx'),
        path.join(viewPath, 'src/index.tsx'),
        path.join(viewPath, 'src/main.tsx'),
    ]

    let entryPoint = null
    for (const entry of possibleEntries) {
        if (fs.existsSync(entry)) {
            entryPoint = entry
            break
        }
    }

    if (!entryPoint) {
        console.log(`[views] No entry point found in ${viewPath}`)
        return
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

    // Copy index.html if exists
    const htmlSrc = path.join(viewPath, 'index.html')
    const htmlDst = path.join(outDir, 'index.html')
    if (fs.existsSync(htmlSrc)) {
        await fs.promises.copyFile(htmlSrc, htmlDst)
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
