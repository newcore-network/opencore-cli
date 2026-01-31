const path = require('path')
const fs = require('fs')
const { getEsbuild, createSwcPlugin, createExcludeNodeAdaptersPlugin, createExternalPackagesPlugin, preserveFiveMExportsPlugin, createNodeGlobalsShimPlugin, createTsconfigPathsPlugin, createReflectMetadataPlugin, createAutoloadDynamicImportShimPlugin, createAutoloadControllersRedirectPlugin } = require('./plugins')
const { getSharedConfig, getBuildOptions, getExternals } = require('./config')
const { handleDependencies, shouldHandleDependencies, detectNativePackages, printNativePackageWarnings } = require('./dependencies')

function normalizeServerBinaryPlatform(platform) {
    if (!platform) return null
    const value = platform.toLowerCase()
    if (['windows', 'win32', 'win'].includes(value)) return 'win32'
    if (['linux', 'linux64'].includes(value)) return 'linux'
    if (['darwin', 'mac', 'macos', 'osx'].includes(value)) return 'darwin'
    return value
}

function getServerBinaryPlatform(options = {}) {
    if (options.serverBinaryPlatform) {
        return normalizeServerBinaryPlatform(options.serverBinaryPlatform)
    }
    return process.platform
}

function resolveServerBinaries(resourcePath, options = {}) {
    const platform = getServerBinaryPlatform(options)
    const platformBin = platform ? path.join('bin', platform) : null

    if (options.serverBinaries === undefined) {
        if (platformBin && fs.existsSync(path.join(resourcePath, platformBin))) {
            return [platformBin]
        }
        const defaultDir = path.join(resourcePath, 'bin')
        if (fs.existsSync(defaultDir)) {
            return ['bin']
        }
        return []
    }

    if (Array.isArray(options.serverBinaries)) {
        if (platformBin && options.serverBinaries.includes('bin')) {
            const platformPath = path.join(resourcePath, platformBin)
            if (fs.existsSync(platformPath)) {
                return options.serverBinaries.map(p => (p === 'bin' ? platformBin : p))
            }
        }
        return options.serverBinaries
    }

    return []
}

function shouldCopyServerBinaries(options = {}, serverBuildOptions, serverEntry) {
    return !!serverBuildOptions && !!serverEntry
}

async function copyServerBinaries(resourcePath, outDir, options = {}, serverBuildOptions, serverEntry) {
    if (!shouldCopyServerBinaries(options, serverBuildOptions, serverEntry)) {
        return
    }

    const patterns = resolveServerBinaries(resourcePath, options)
    if (patterns.length === 0) {
        return
    }

    for (const pattern of patterns) {
        const srcPath = path.join(resourcePath, pattern)
        if (!fs.existsSync(srcPath)) {
            console.warn(`[server] serverBinaries path not found: ${pattern}`)
            continue
        }

        const stats = await fs.promises.stat(srcPath)
        if (stats.isDirectory()) {
            await copyDirContents(srcPath, outDir)
        } else {
            await fs.promises.copyFile(srcPath, path.join(outDir, path.basename(srcPath)))
        }
    }
}

async function copyDirContents(srcDir, destDir) {
    await fs.promises.mkdir(destDir, { recursive: true })
    const entries = await fs.promises.readdir(srcDir, { withFileTypes: true })

    for (const entry of entries) {
        const srcPath = path.join(srcDir, entry.name)
        const dstPath = path.join(destDir, entry.name)
        if (entry.isDirectory()) {
            await copyDirContents(srcPath, dstPath)
        } else {
            await fs.promises.copyFile(srcPath, dstPath)
        }
    }
}

function getCorePlugins(isServerBuild = false, externals = [], target = 'es2020', format = 'iife', resourcePath = null) {
    const plugins = [
        createReflectMetadataPlugin(),
        createAutoloadDynamicImportShimPlugin(),
        createAutoloadControllersRedirectPlugin(resourcePath),
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
        createReflectMetadataPlugin(),
        createAutoloadDynamicImportShimPlugin(),
        createAutoloadControllersRedirectPlugin(resourcePath),
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
        createReflectMetadataPlugin(),
        createAutoloadDynamicImportShimPlugin(),
        createAutoloadControllersRedirectPlugin(resourcePath),
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
    await copyServerBinaries(resourcePath, outDir, options, serverBuildOptions, serverEntry)
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
            ...shared,
            ...serverBuildOptions,
            target: serverTarget,
            entryPoints: [serverEntry],
            outfile: path.join(outDir, 'server.js'),
            plugins: getResourcePlugins(true, serverExternals, serverTarget, serverFormat, resourcePath),
            external: serverExternals,
            define: {
                ...shared.define,
                '__OPENCORE_LOG_LEVEL__': JSON.stringify(options.logLevel || 'INFO'),
                '__OPENCORE_TARGET__': '"server"'
            }
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
            define: {
                ...shared.define,
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

    if (builds.length > 0) await Promise.all(builds)
    await copyServerBinaries(resourcePath, outDir, options, serverBuildOptions, serverEntry)
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
            define: {
                ...shared.define,
                '__OPENCORE_LOG_LEVEL__': JSON.stringify(options.logLevel || 'INFO'),
                '__OPENCORE_TARGET__': '"server"'
            }
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
            define: {
                ...shared.define,
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

    if (builds.length > 0) await Promise.all(builds)
    await copyServerBinaries(resourcePath, outDir, options, serverBuildOptions, serverEntry)
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

    const copyServerEntry = resolveEntry(resourcePath, 'server', options.entryPoints?.server)
    await copyServerBinaries(resourcePath, outDir, options, { platform: 'node' }, copyServerEntry)
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
