const path = require('path')
const fs = require('fs')
const { getEsbuild, createSwcPlugin, createExcludeNodeAdaptersPlugin, createExternalPackagesPlugin, preserveFiveMExportsPlugin, createNodeGlobalsShimPlugin, createTsconfigPathsPlugin, createReflectMetadataPlugin, createAutoloadDynamicImportShimPlugin, createAutoloadControllersRedirectPlugin, createEnvironmentAliasPlugin } = require('./plugins')
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

function getCorePlugins(isServerBuild = false, externals = [], target = 'es2020', format = 'iife', resourcePath = null, packageManager = null, environmentAliases = null) {
    const plugins = [
        createReflectMetadataPlugin({ packageManager, resourcePath, target: isServerBuild ? 'server' : 'client' }),
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

    // Must be unshifted after tsconfig-paths so it ends up at index 0 and runs first.
    const envPlugin = createEnvironmentAliasPlugin(environmentAliases)
    if (envPlugin) {
        plugins.unshift(envPlugin)
    }

    return plugins
}

function getResourcePlugins(isServerBuild = false, externals = [], target = 'es2020', format = 'iife', resourcePath = null, packageManager = null, environmentAliases = null) {
    const plugins = [
        createReflectMetadataPlugin({ packageManager, resourcePath, target: isServerBuild ? 'server' : 'client' }),
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

    const envPlugin = createEnvironmentAliasPlugin(environmentAliases)
    if (envPlugin) {
        plugins.unshift(envPlugin)
    }

    return plugins
}

function getStandalonePlugins(isServerBuild = false, externals = [], target = 'es2020', format = 'iife', resourcePath = null, packageManager = null, environmentAliases = null) {
    const plugins = [
        createReflectMetadataPlugin({ packageManager, resourcePath, target: isServerBuild ? 'server' : 'client' }),
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

    const envPlugin = createEnvironmentAliasPlugin(environmentAliases)
    if (envPlugin) {
        plugins.unshift(envPlugin)
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

function getLayoutOptions(outDir, options = {}) {
    return {
        runtime: options.runtime || 'fivem',
        serverOutDir: options.serverOutDir || outDir,
        clientOutDir: options.clientOutDir || outDir,
        serverOutFile: options.serverOutFile || 'server.js',
        clientOutFile: options.clientOutFile || 'client.js',
        manifestKind: options.manifestKind || 'fxmanifest',
    }
}

async function buildCore(resourcePath, outDir, options = {}) {
    const esbuild = getEsbuild()
    const shared = getSharedConfig(options)
    const serverEntry = resolveEntry(resourcePath, 'server', options.entryPoints?.server)
    const clientEntry = resolveEntry(resourcePath, 'client', options.entryPoints?.client)
    const layout = getLayoutOptions(outDir, options)

    await fs.promises.mkdir(layout.serverOutDir, { recursive: true })
    await fs.promises.mkdir(layout.clientOutDir, { recursive: true })
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
            outfile: path.join(layout.serverOutDir, layout.serverOutFile),
            plugins: getCorePlugins(true, serverExternals, serverTarget, serverFormat, resourcePath, options.packageManager, options.environmentAliases),
            external: serverExternals,
            define: {
                '__OPENCORE_LOG_LEVEL__': JSON.stringify(options.logLevel || 'INFO'),
                '__OPENCORE_TARGET__': '"server"',
                '__OPENCORE_RESOURCE_NAME__': JSON.stringify(options.resourceName || '')
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
            outfile: path.join(layout.clientOutDir, layout.clientOutFile),
            plugins: getCorePlugins(false, clientExternals, clientTarget, clientFormat, resourcePath, options.packageManager, options.environmentAliases),
            external: clientExternals,
            define: {
                '__OPENCORE_LOG_LEVEL__': JSON.stringify(options.logLevel || 'INFO'),
                '__OPENCORE_TARGET__': '"client"',
                '__OPENCORE_RESOURCE_NAME__': JSON.stringify(options.resourceName || '')
            }
        }))
    }

    const manifestSrc = path.join(resourcePath, 'fxmanifest.lua')
    const manifestDst = path.join(layout.serverOutDir, 'fxmanifest.lua')
    if (layout.manifestKind === 'fxmanifest' && fs.existsSync(manifestSrc)) {
        await fs.promises.copyFile(manifestSrc, manifestDst)
    }

    if (shouldHandleDependencies(options)) {
        await handleDependencies(resourcePath, layout.serverOutDir)
    }

    await Promise.all(builds)
    await copyServerBinaries(resourcePath, layout.serverOutDir, options, serverBuildOptions, serverEntry)
    console.log(`[core] Built ${path.basename(layout.serverOutDir)}`)
}


async function buildResource(resourcePath, outDir, options = {}) {
    const esbuild = getEsbuild()
    const shared = getSharedConfig(options)
    const layout = getLayoutOptions(outDir, options)
    await fs.promises.mkdir(layout.serverOutDir, { recursive: true })
    await fs.promises.mkdir(layout.clientOutDir, { recursive: true })
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
            outfile: path.join(layout.serverOutDir, layout.serverOutFile),
            plugins: getResourcePlugins(true, serverExternals, serverTarget, serverFormat, resourcePath, options.packageManager, options.environmentAliases),
            external: serverExternals,
            define: {
                ...shared.define,
                '__OPENCORE_LOG_LEVEL__': JSON.stringify(options.logLevel || 'INFO'),
                '__OPENCORE_TARGET__': '"server"',
                '__OPENCORE_RESOURCE_NAME__': JSON.stringify(options.resourceName || '')
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
            outfile: path.join(layout.clientOutDir, layout.clientOutFile),
            plugins: getResourcePlugins(false, clientExternals, clientTarget, clientFormat, resourcePath, options.packageManager, options.environmentAliases),
            external: clientExternals,
            define: {
                ...shared.define,
                '__OPENCORE_LOG_LEVEL__': JSON.stringify(options.logLevel || 'INFO'),
                '__OPENCORE_TARGET__': '"client"',
                '__OPENCORE_RESOURCE_NAME__': JSON.stringify(options.resourceName || '')
            }
        }))
    }

    const manifestSrc = path.join(resourcePath, 'fxmanifest.lua')
    const manifestDst = path.join(layout.serverOutDir, 'fxmanifest.lua')
    if (layout.manifestKind === 'fxmanifest' && fs.existsSync(manifestSrc)) {
        await fs.promises.copyFile(manifestSrc, manifestDst)
    }

    if (shouldHandleDependencies(options)) {
        await handleDependencies(resourcePath, layout.serverOutDir)
    }

    if (builds.length > 0) await Promise.all(builds)
    await copyServerBinaries(resourcePath, layout.serverOutDir, options, serverBuildOptions, serverEntry)
    console.log(`[resource] Built ${path.basename(layout.serverOutDir)}`)
}


async function buildStandalone(resourcePath, outDir, options = {}) {
    const esbuild = getEsbuild()
    const shared = getSharedConfig(options)
    const layout = getLayoutOptions(outDir, options)
    await fs.promises.mkdir(layout.serverOutDir, { recursive: true })
    await fs.promises.mkdir(layout.clientOutDir, { recursive: true })
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
            outfile: path.join(layout.serverOutDir, layout.serverOutFile),
            plugins: getStandalonePlugins(true, serverExternals, serverTarget, serverFormat, resourcePath, options.packageManager, options.environmentAliases),
            external: serverExternals,
            define: {
                ...shared.define,
                '__OPENCORE_LOG_LEVEL__': JSON.stringify(options.logLevel || 'INFO'),
                '__OPENCORE_TARGET__': '"server"',
                '__OPENCORE_RESOURCE_NAME__': JSON.stringify(options.resourceName || '')
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
            outfile: path.join(layout.clientOutDir, layout.clientOutFile),
            plugins: getStandalonePlugins(false, clientExternals, clientTarget, clientFormat, resourcePath, options.packageManager, options.environmentAliases),
            external: clientExternals,
            define: {
                ...shared.define,
                '__OPENCORE_LOG_LEVEL__': JSON.stringify(options.logLevel || 'INFO'),
                '__OPENCORE_TARGET__': '"client"',
                '__OPENCORE_RESOURCE_NAME__': JSON.stringify(options.resourceName || '')
            }
        }))
    }

    const manifestSrc = path.join(resourcePath, 'fxmanifest.lua')
    const manifestDst = path.join(layout.serverOutDir, 'fxmanifest.lua')
    if (layout.manifestKind === 'fxmanifest' && fs.existsSync(manifestSrc)) {
        await fs.promises.copyFile(manifestSrc, manifestDst)
    }

    if (shouldHandleDependencies(options)) {
        await handleDependencies(resourcePath, layout.serverOutDir)
    }

    if (builds.length > 0) await Promise.all(builds)
    await copyServerBinaries(resourcePath, layout.serverOutDir, options, serverBuildOptions, serverEntry)
    console.log(`[standalone] Built ${path.basename(layout.serverOutDir)}`)
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
