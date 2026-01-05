const path = require('path')
const fs = require('fs')

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
            '__dirname': 'globalThis.__dirname',
            '__filename': 'globalThis.__filename',
        }
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
    const defaults = {
        platform: side === 'server' ? 'node' : 'browser',
        target: 'es2020',
        format: 'iife',
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
        },
    }
}

function getExternals(side, options = {}) {
    const sideOptions = options[side]
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
