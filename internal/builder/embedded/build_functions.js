const path = require('path')
const fs = require('fs')
const { getEsbuild, createSwcPlugin, createExcludeNodeAdaptersPlugin, createExternalPackagesPlugin, preserveFiveMExportsPlugin, createNodeGlobalsShimPlugin, createTsconfigPathsPlugin } = require('./plugins')
const { getSharedConfig, getBuildOptions, getExternals } = require('./config')
const { handleDependencies, shouldHandleDependencies, detectNativePackages, printNativePackageWarnings } = require('./dependencies')

function getCorePlugins(isServerBuild = false, externals = [], target = 'es2020', format = 'iife', resourcePath = null) {
    const plugins = [
        createExternalPackagesPlugin(externals),
        createSwcPlugin(target),
        createExcludeNodeAdaptersPlugin(isServerBuild),
        preserveFiveMExportsPlugin,
        createNodeGlobalsShimPlugin(format)
    ]

    // Add tsconfig-paths plugin if resourcePath is provided
    if (resourcePath) {
        const tsconfigPlugin = createTsconfigPathsPlugin(resourcePath)
        if (tsconfigPlugin) {
            plugins.unshift(tsconfigPlugin) // Add at the beginning for early resolution
        }
    }

    return plugins
}

function getResourcePlugins(isServerBuild = false, externals = [], target = 'es2020', format = 'iife', resourcePath = null) {
    const plugins = [
        createExternalPackagesPlugin(externals),
        createSwcPlugin(target),
        createExcludeNodeAdaptersPlugin(isServerBuild),
        preserveFiveMExportsPlugin,
        createNodeGlobalsShimPlugin(format)
    ]

    if (resourcePath) {
        const tsconfigPlugin = createTsconfigPathsPlugin(resourcePath)
        if (tsconfigPlugin) {
            plugins.unshift(tsconfigPlugin)
        }
    }

    return plugins
}

function getStandalonePlugins(isServerBuild = false, externals = [], target = 'es2020', format = 'iife', resourcePath = null) {
    const plugins = [
        createExternalPackagesPlugin(externals),
        createSwcPlugin(target),
        createExcludeNodeAdaptersPlugin(isServerBuild),
        createNodeGlobalsShimPlugin(format)
    ]

    if (resourcePath) {
        const tsconfigPlugin = createTsconfigPathsPlugin(resourcePath)
        if (tsconfigPlugin) {
            plugins.unshift(tsconfigPlugin)
        }
    }

    return plugins
}

/**
 * Check for native packages and warn the user
 */
async function checkNativePackages(resourcePath, options = {}) {
    const nodeModulesPath = path.join(resourcePath, 'node_modules')
    const serverExternals = getExternals('server', options)
    const clientExternals = getExternals('client', options)
    const allExternals = [...new Set([...serverExternals, ...clientExternals])]

    const { warnings, errors } = await detectNativePackages(nodeModulesPath, allExternals)
    printNativePackageWarnings(warnings, errors)
}

/**
 * Resolves entry points based on multiple possible patterns
 */
function resolveEntry(resourcePath, side, explicitEntry = null) {
    if (explicitEntry) return explicitEntry;

    const patterns = side === 'server' ? [
        'src/server.ts',       // Root src (no-architecture)
        'src/server/main.ts',  // Layer-based / Standard
        'src/server/index.ts'  // Standard index
    ] : [
        'src/client.ts',       // Root src (no-architecture)
        'src/client/main.ts',  // Layer-based / Standard
        'src/client/index.ts'  // Standard index
    ];

    for (const pattern of patterns) {
        const fullPath = path.join(resourcePath, pattern);
        if (fs.existsSync(fullPath)) {
            return fullPath;
        }
    }

    return null;
}

async function buildCore(resourcePath, outDir, options = {}) {
    const esbuild = getEsbuild()
    const shared = getSharedConfig(options)
    const serverEntry = resolveEntry(resourcePath, 'server', options.entryPoints?.server)
    const clientEntry = resolveEntry(resourcePath, 'client', options.entryPoints?.client)

    await fs.promises.mkdir(outDir, { recursive: true })
    await checkNativePackages(resourcePath, options)
    const builds = []

    const serverBuildOptions = getBuildOptions('server', options)
    if (serverBuildOptions !== null && serverEntry) {
        const serverExternals = getExternals('server', options)
        const serverTarget = (serverBuildOptions.target || 'es2020').toLowerCase()
        const serverFormat = serverBuildOptions.format || 'cjs'
        builds.push(esbuild.build({
            ...shared,
            ...serverBuildOptions,
            target: serverTarget,
            entryPoints: [serverEntry],
            outfile: path.join(outDir, 'server.js'),
            plugins: getCorePlugins(true, serverExternals, serverTarget, serverFormat, resourcePath),
            external: serverExternals,
            define: {
                '__OPENCORE_LOG_LEVEL__': JSON.stringify(options.logLevel || 'INFO'),
                '__OPENCORE_TARGET__': '"server"'
            }
        }))
    }

    const clientBuildOptions = getBuildOptions('client', options)
    if (clientBuildOptions !== null && clientEntry) {
        const clientExternals = getExternals('client', options)
        const clientTarget = (clientBuildOptions.target || 'es2020').toLowerCase()
        const clientFormat = clientBuildOptions.format || 'iife'
        builds.push(esbuild.build({
            ...shared,
            ...clientBuildOptions,
            target: clientTarget,
            entryPoints: [clientEntry],
            outfile: path.join(outDir, 'client.js'),
            plugins: getCorePlugins(false, clientExternals, clientTarget, clientFormat, resourcePath),
            external: clientExternals,
            define: {
                '__OPENCORE_LOG_LEVEL__': JSON.stringify(options.logLevel || 'INFO'),
                '__OPENCORE_TARGET__': '"client"'
            }
        }))
    }

    const manifestSrc = path.join(resourcePath, 'fxmanifest.lua')
    const manifestDst = path.join(outDir, 'fxmanifest.lua')
    if (fs.existsSync(manifestSrc)) {
        await fs.promises.copyFile(manifestSrc, manifestDst)
    }

    if (shouldHandleDependencies(options)) {
        await handleDependencies(resourcePath, outDir)
    }

    await Promise.all(builds)
    console.log(`[core] Built ${path.basename(outDir)}`)
}

async function buildResource(resourcePath, outDir, options = {}) {
    const esbuild = getEsbuild()
    const shared = getSharedConfig(options)
    await fs.promises.mkdir(outDir, { recursive: true })
    await checkNativePackages(resourcePath, options)
    const builds = []

    const serverEntry = resolveEntry(resourcePath, 'server', options.entryPoints?.server)
    const serverBuildOptions = getBuildOptions('server', options)
    if (serverBuildOptions !== null && serverEntry) {
        const serverExternals = getExternals('server', options)
        const serverTarget = (serverBuildOptions.target || 'es2020').toLowerCase()
        const serverFormat = serverBuildOptions.format || 'cjs'
        builds.push(esbuild.build({
            ...shared, ...serverBuildOptions,
            target: serverTarget,
            entryPoints: [serverEntry],
            outfile: path.join(outDir, 'server.js'),
            plugins: getResourcePlugins(true, serverExternals, serverTarget, serverFormat, resourcePath),
            external: serverExternals,
        }))
    }

    const clientEntry = resolveEntry(resourcePath, 'client', options.entryPoints?.client)
    const clientBuildOptions = getBuildOptions('client', options)
    if (clientBuildOptions !== null && clientEntry) {
        const clientExternals = getExternals('client', options)
        const clientTarget = (clientBuildOptions.target || 'es2020').toLowerCase()
        const clientFormat = clientBuildOptions.format || 'iife'
        builds.push(esbuild.build({
            ...shared, ...clientBuildOptions,
            target: clientTarget,
            entryPoints: [clientEntry],
            outfile: path.join(outDir, 'client.js'),
            plugins: getResourcePlugins(false, clientExternals, clientTarget, clientFormat, resourcePath),
            external: clientExternals,
        }))
    }

    const manifestSrc = path.join(resourcePath, 'fxmanifest.lua')
    const manifestDst = path.join(outDir, 'fxmanifest.lua')
    if (fs.existsSync(manifestSrc)) {
        await fs.promises.copyFile(manifestSrc, manifestDst)
    }

    if (shouldHandleDependencies(options)) {
        await handleDependencies(resourcePath, outDir)
    }

    if (builds.length > 0) await Promise.all(builds)
    console.log(`[resource] Built ${path.basename(outDir)}`)
}

async function buildStandalone(resourcePath, outDir, options = {}) {
    const esbuild = getEsbuild()
    const shared = getSharedConfig(options)
    await fs.promises.mkdir(outDir, { recursive: true })
    await checkNativePackages(resourcePath, options)
    const builds = []

    const serverEntry = resolveEntry(resourcePath, 'server', options.entryPoints?.server)
    const serverBuildOptions = getBuildOptions('server', options)
    if (serverBuildOptions !== null && serverEntry) {
        const serverExternals = getExternals('server', options)
        const serverTarget = (serverBuildOptions.target || 'es2020').toLowerCase()
        const serverFormat = serverBuildOptions.format || 'cjs'
        builds.push(esbuild.build({
            ...shared, ...serverBuildOptions,
            target: serverTarget,
            entryPoints: [serverEntry],
            outfile: path.join(outDir, 'server.js'),
            plugins: getStandalonePlugins(true, serverExternals, serverTarget, serverFormat, resourcePath),
            external: serverExternals,
        }))
    }

    const clientEntry = resolveEntry(resourcePath, 'client', options.entryPoints?.client)
    const clientBuildOptions = getBuildOptions('client', options)
    if (clientBuildOptions !== null && clientEntry) {
        const clientExternals = getExternals('client', options)
        const clientTarget = (clientBuildOptions.target || 'es2020').toLowerCase()
        const clientFormat = clientBuildOptions.format || 'iife'
        builds.push(esbuild.build({
            ...shared, ...clientBuildOptions,
            target: clientTarget,
            entryPoints: [clientEntry],
            outfile: path.join(outDir, 'client.js'),
            plugins: getStandalonePlugins(false, clientExternals, clientTarget, clientFormat, resourcePath),
            external: clientExternals,
        }))
    }

    const manifestSrc = path.join(resourcePath, 'fxmanifest.lua')
    const manifestDst = path.join(outDir, 'fxmanifest.lua')
    if (fs.existsSync(manifestSrc)) {
        await fs.promises.copyFile(manifestSrc, manifestDst)
    }

    if (shouldHandleDependencies(options)) {
        await handleDependencies(resourcePath, outDir)
    }

    if (builds.length > 0) await Promise.all(builds)
    console.log(`[standalone] Built ${path.basename(outDir)}`)
}

async function copyResource(resourcePath, outDir, options = {}) {
    const absSrcPath = path.resolve(resourcePath)
    const absOutDir = path.resolve(outDir)

    if (absSrcPath === absOutDir) {
        if (shouldHandleDependencies(options)) {
            await handleDependencies(resourcePath, outDir)
        }
        return
    }

    if (!fs.existsSync(absSrcPath)) {
        throw new Error(`Source path does not exist: ${absSrcPath}`)
    }

    await fs.promises.mkdir(absOutDir, { recursive: true })

    const entries = await fs.promises.readdir(absSrcPath, { withFileTypes: true })
    
    for (const entry of entries) {
        const src = path.join(absSrcPath, entry.name)
        const dst = path.join(absOutDir, entry.name)
        
        if (entry.isDirectory()) {
            if (entry.name === 'node_modules' || entry.name === 'dist' || entry.name === '.git' || entry.name === '.ocignore') {
                continue
            }
            await copyDirRecursive(src, dst)
        } else {
            if (entry.name === 'package.json') continue 
            await fs.promises.copyFile(src, dst)
        }
    }

    if (shouldHandleDependencies(options)) {
        await handleDependencies(resourcePath, outDir)
    }
    console.log(`[copy] Copied ${path.basename(outDir)}`)
}

async function copyDirRecursive(src, dst) {
    await fs.promises.mkdir(dst, { recursive: true })
    const entries = await fs.promises.readdir(src, { withFileTypes: true })
    
    for (const entry of entries) {
        const srcPath = path.join(src, entry.name)
        const dstPath = path.join(dst, entry.name)
        
        if (entry.isDirectory()) {
            await copyDirRecursive(srcPath, dstPath)
        } else {
            await fs.promises.copyFile(srcPath, dstPath)
        }
    }
}

module.exports = {
    buildCore,
    buildResource,
    buildStandalone,
    copyResource
}
