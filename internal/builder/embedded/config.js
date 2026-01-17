const path = require('path')
const fs = require('fs')

// FiveM runtime limitations warning
let clientExternalsWarningShown = false

function getSharedConfig(options = {}) {
    const config = {
        bundle: true,
        sourcemap: options.sourceMaps ? 'inline' : false,
        minifyWhitespace: options.minify !== false,
        minifySyntax: options.minify !== false,
        minifyIdentifiers: false,
        keepNames: true,
        treeShaking: true,
        logLevel: 'info',
        legalComments: 'none',
        define: {
            'process.env.NODE_ENV': options.minify ? '"production"' : '"development"',
        },
        supported: {
            'class-static-blocks': false,
        },
        alias: {}
    }

    return config
}

function mergeOptions(side, sideOptions, globalOptions, defaults) {
    if (sideOptions === false) {
        return null
    }

    const merged = { ...defaults }

    if (globalOptions.minify !== undefined) merged.minify = globalOptions.minify
    if (globalOptions.sourceMaps !== undefined) merged.sourceMaps = globalOptions.sourceMaps

    if (sideOptions && typeof sideOptions === 'object') {
        if (sideOptions.platform !== undefined) merged.platform = sideOptions.platform
        if (sideOptions.format !== undefined) merged.format = sideOptions.format
        if (sideOptions.target !== undefined) merged.target = sideOptions.target
        if (sideOptions.external !== undefined) merged.external = sideOptions.external
        if (sideOptions.minify !== undefined) merged.minify = sideOptions.minify
        if (sideOptions.sourceMaps !== undefined) merged.sourceMaps = sideOptions.sourceMaps
    }

    return merged
}

function getBuildOptions(side, options = {}) {
    // FiveM runtime:
    // Server: Node.js runtime with full Node APIs
    // Client: Neutral JS runtime - no Node.js APIs, no Web APIs
    const defaults = {
        platform: side === 'server' ? 'node' : 'neutral',
        target: side === 'server' ? 'es2023' : 'es2020',
        format: side === 'server' ? 'cjs' : 'iife',
        external: [],
        minify: false,
        sourceMaps: false,
    }

    const sideOptions = options[side]
    const merged = mergeOptions(side, sideOptions, options, defaults)

    if (merged === null) {
        return null
    }

    return {
        platform: merged.platform,
        target: merged.target,
        format: merged.format,
        mainFields: ['module', 'main'],
        conditions: ['import', 'default'],
        supported: {
            'dynamic-import': true,
            'class-static-blocks': false,
        },
    }
}

function getExternals(side, options = {}) {
    const sideOptions = options[side]

    // Client cannot use externals - FiveM client has no filesystem access
    // All dependencies MUST be bundled into the final JS file
    if (side === 'client') {
        if (sideOptions?.external?.length > 0 && !clientExternalsWarningShown) {
            clientExternalsWarningShown = true
            console.log('')
            console.log('\x1b[31m┌──────────────────────────────────────────────────────────────────┐\x1b[0m')
            console.log('\x1b[31m│  ERROR: externals not supported for FiveM client                 │\x1b[0m')
            console.log('\x1b[31m└──────────────────────────────────────────────────────────────────┘\x1b[0m')
            console.log('\x1b[33m  FiveM client has no access to node_modules or filesystem.\x1b[0m')
            console.log('\x1b[33m  All dependencies must be bundled into the final .js file.\x1b[0m')
            console.log('\x1b[90m  Ignoring client.external configuration...\x1b[0m')
            console.log('')
        }
        return []
    }

    if (sideOptions && typeof sideOptions === 'object' && Array.isArray(sideOptions.external)) {
        return sideOptions.external
    }
    return []
}

module.exports = {
    getSharedConfig,
    mergeOptions,
    getBuildOptions,
    getExternals
}
