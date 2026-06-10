const path = require('path')
const fs = require('fs')
const { builtinModules, createRequire } = require('module')
const { execFile } = require('child_process')
const { promisify } = require('util')
const crypto = require('crypto')
const { getBuildOptions, getExternals } = require('./config')

const execFileAsync = promisify(execFile)
const builtins = new Set([...builtinModules, ...builtinModules.map(name => `node:${name}`)])

const KNOWN_NATIVE_PACKAGES = [
    'bcrypt', 'sharp', 'canvas', 'sqlite3', 'better-sqlite3', 'node-gyp',
    'node-pre-gyp', 'fsevents', 'esbuild', 'swc', '@swc/core', 'lightningcss',
    'sodium-native', 'argon2', 'cpu-features', 'microtime', 'bufferutil', 'utf-8-validate',
]

const NATIVE_ALTERNATIVES = {
    bcrypt: 'bcryptjs',
    argon2: 'hash.js or js-sha3',
    sharp: 'jimp',
    canvas: 'pureimage',
    sqlite3: 'sql.js',
    'better-sqlite3': 'sql.js',
}

async function isNativePackage(packagePath) {
    try {
        if (fs.existsSync(path.join(packagePath, 'binding.gyp'))) return { isNative: true, reason: 'has binding.gyp (C++ addon)' }
        if (fs.existsSync(path.join(packagePath, 'prebuilds'))) return { isNative: true, reason: 'has prebuilt binaries' }

        const buildPath = path.join(packagePath, 'build')
        if (fs.existsSync(buildPath)) {
            const files = await fs.promises.readdir(buildPath, { recursive: true }).catch(() => [])
            for (const file of files) {
                if (file.endsWith('.node')) return { isNative: true, reason: 'has compiled .node binaries' }
            }
        }

        const pkgJsonPath = path.join(packagePath, 'package.json')
        if (fs.existsSync(pkgJsonPath)) {
            const pkgJson = JSON.parse(await fs.promises.readFile(pkgJsonPath, 'utf8'))
            if (pkgJson.gypfile || pkgJson.binary) return { isNative: true, reason: 'package.json declares native bindings' }
        }

        return { isNative: false }
    } catch (e) {
        return { isNative: false }
    }
}

async function detectNativePackages(nodeModulesPath, externals = []) {
    const warnings = []
    const errors = []

    if (!fs.existsSync(nodeModulesPath)) return { warnings, errors }

    for (const pkgName of KNOWN_NATIVE_PACKAGES) {
        const pkgPath = path.join(nodeModulesPath, pkgName)
        const pnpmPath = path.join(nodeModulesPath, '.pnpm')
        let found = false
        let foundPath = pkgPath

        if (fs.existsSync(pkgPath)) found = true

        if (!found && fs.existsSync(pnpmPath)) {
            try {
                const pnpmDirs = await fs.promises.readdir(pnpmPath)
                for (const dir of pnpmDirs) {
                    if (dir.startsWith(pkgName + '@')) {
                        found = true
                        foundPath = path.join(pnpmPath, dir, 'node_modules', pkgName)
                        break
                    }
                }
            } catch (e) {}
        }

        if (found) {
            const isExternal = externals.includes(pkgName)
            const alternative = NATIVE_ALTERNATIVES[pkgName]
            if (isExternal) {
                errors.push({
                    package: pkgName,
                    message: `"${pkgName}" is a native C++ package and WILL NOT WORK in FiveM runtime.`,
                    hint: alternative ? `Use "${alternative}" instead.` : 'This package requires Node.js native bindings that are incompatible with FiveM.',
                    severity: 'error'
                })
            } else {
                warnings.push({
                    package: pkgName,
                    message: `"${pkgName}" is a native C++ package being bundled.`,
                    hint: alternative ? `Consider using "${alternative}" instead.` : 'This package may cause build errors or runtime failures.',
                    severity: 'warning'
                })
            }
        }
    }

    return { warnings, errors }
}

function printNativePackageWarnings(warnings, errors) {
    if (errors.length > 0) {
        console.log('')
        console.log('\x1b[31m╔══════════════════════════════════════════════════════════════════╗\x1b[0m')
        console.log('\x1b[31m║  NATIVE C++ PACKAGES DETECTED - INCOMPATIBLE WITH FIVEM          ║\x1b[0m')
        console.log('\x1b[31m╚══════════════════════════════════════════════════════════════════╝\x1b[0m')
        console.log('\x1b[90m  FiveM uses a neutral JS runtime without Node.js or Web APIs.\x1b[0m')
        console.log('\x1b[90m  Native packages with C++ bindings (.node) will NOT work.\x1b[0m')
        console.log('')
        for (const err of errors) {
            console.log(`\x1b[31m  x ${err.message}\x1b[0m`)
            console.log(`\x1b[33m    -> ${err.hint}\x1b[0m`)
        }
        console.log('')
    }

    if (warnings.length > 0) {
        console.log('')
        console.log('\x1b[33m┌──────────────────────────────────────────────────────────────────┐\x1b[0m')
        console.log('\x1b[33m│  Native Package Warnings                                         │\x1b[0m')
        console.log('\x1b[33m└──────────────────────────────────────────────────────────────────┘\x1b[0m')
        for (const warn of warnings) {
            console.log(`\x1b[33m  ! ${warn.message}\x1b[0m`)
            console.log(`\x1b[90m    -> ${warn.hint}\x1b[0m`)
        }
        console.log('')
    }
}

function dependencyConfig(options = {}) {
    return options.dependencyResolution || {}
}

function dependencyMode(options = {}) {
    const mode = (dependencyConfig(options).mode || 'auto').toLowerCase()
    if (mode === 'auto') return 'isolated'
    if (mode === 'isolated' || mode === 'symlink' || mode === 'shared-resource' || mode === 'bundle') return mode
    throw new Error(`[deps] Invalid dependencyResolution.mode "${mode}". Expected auto, isolated, symlink, shared-resource, or bundle.`)
}

function normalizeExternalImport(specifier) {
    const value = String(specifier || '').trim()
    if (!value || builtins.has(value)) return null
    if (value.startsWith('.') || value.startsWith('/') || value.startsWith('file:')) {
        throw new Error(`[deps] Invalid server.external entry "${value}". Use package names only, not filesystem paths.`)
    }
    if (value.startsWith('@')) {
        const parts = value.split('/')
        if (parts.length < 2 || !parts[0] || !parts[1]) throw new Error(`[deps] Invalid scoped package external "${value}".`)
        return `${parts[0]}/${parts[1]}`
    }
    return value.split('/')[0]
}

function normalizedExternals(options = {}) {
    const packages = []
    const seen = new Set()
    for (const external of getExternals('server', options)) {
        const normalized = normalizeExternalImport(external)
        if (normalized && !seen.has(normalized)) {
            seen.add(normalized)
            packages.push(normalized)
        }
    }
    return packages
}

async function readPackageJson(dir) {
    const pkgPath = path.join(dir, 'package.json')
    if (!fs.existsSync(pkgPath)) return null
    return JSON.parse(await fs.promises.readFile(pkgPath, 'utf8'))
}

function dependencySpecFromPackage(pkg, name) {
    if (!pkg) return null
    return pkg.dependencies?.[name] || pkg.optionalDependencies?.[name] || pkg.peerDependencies?.[name] || null
}

function validateInstallSpec(name, spec) {
    if (!spec || String(spec).trim() === '' || spec === 'latest' || spec === '*') {
        throw new Error(`[deps] Could not resolve a pinned version for "${name}". Add it to the resource or root package.json; refusing to install latest.`)
    }
    if (/^(file:|link:|workspace:|portal:|\.\.?\/|\/)/.test(spec)) {
        throw new Error(`[deps] Dependency "${name}" uses unsupported local version spec "${spec}". Use a registry version for deployable resources.`)
    }
    return spec
}

function installedPackageJsonPath(name, resourcePath) {
    for (const base of [resourcePath, process.cwd()]) {
        try {
            const req = createRequire(path.join(base, 'package.json'))
            return req.resolve(`${name}/package.json`)
        } catch (e) {}
    }
    return null
}

function installedPackagePath(name, resourcePath) {
    const pkgJsonPath = installedPackageJsonPath(name, resourcePath)
    return pkgJsonPath ? path.dirname(pkgJsonPath) : null
}

async function scanDynamicRequires(packagePath) {
    const warnings = []
    const maxWarnings = 10
    const dynamicRequirePattern = /\brequire\s*\(\s*(?!['"`])[^)]*\)/

    async function walk(current) {
        if (warnings.length >= maxWarnings) return
        const entries = await fs.promises.readdir(current, { withFileTypes: true }).catch(() => [])
        for (const entry of entries) {
            if (warnings.length >= maxWarnings) return
            const entryPath = path.join(current, entry.name)
            if (entry.isDirectory()) {
                if (entry.name === 'node_modules' || entry.name === '.git') continue
                await walk(entryPath)
                continue
            }
            if (!/\.(js|cjs|mjs)$/.test(entry.name)) continue
            const contents = await fs.promises.readFile(entryPath, 'utf8').catch(() => '')
            if (dynamicRequirePattern.test(contents)) warnings.push(entryPath)
        }
    }

    await walk(packagePath)
    return warnings
}

async function checkBundleCompatibility(resourcePath, options = {}) {
    if (dependencyMode(options) !== 'bundle') return

    const packageNames = normalizedExternals(options)
    if (packageNames.length === 0) return

    for (const name of packageNames) {
        const packagePath = installedPackagePath(name, resourcePath)
        if (!packagePath) {
            throw new Error(`[deps] Cannot bundle "${name}" because it is not installed. Install it in the resource or root project first.`)
        }

        const native = await isNativePackage(packagePath)
        if (native.isNative) {
            throw new Error(`[deps] Cannot bundle "${name}" in dependencyResolution.mode "bundle": ${native.reason}. Use "isolated" for packages with native bindings or runtime assets.`)
        }

        const dynamicRequires = await scanDynamicRequires(packagePath)
        if (dynamicRequires.length > 0) {
            console.warn(`[deps] Warning: "${name}" contains dynamic require() calls that may not bundle reliably:`)
            for (const file of dynamicRequires.slice(0, 5)) console.warn(`[deps]   ${path.relative(packagePath, file)}`)
            if (dynamicRequires.length > 5) console.warn(`[deps]   ...and ${dynamicRequires.length - 5} more`)
        }
    }
}

async function resolveDependencyVersions(resourcePath, packageNames) {
    const resourcePkg = await readPackageJson(resourcePath)
    const rootPkg = await readPackageJson(process.cwd())
    const dependencies = {}

    for (const name of packageNames) {
        let spec = dependencySpecFromPackage(resourcePkg, name)
        if (!spec) spec = dependencySpecFromPackage(rootPkg, name)
        if (!spec) {
            const installedPkgPath = installedPackageJsonPath(name, resourcePath)
            if (installedPkgPath) {
                const installedPkg = JSON.parse(await fs.promises.readFile(installedPkgPath, 'utf8'))
                spec = installedPkg.version
            }
        }
        dependencies[name] = validateInstallSpec(name, spec)
    }

    return dependencies
}

function getInstallPackageManager(options = {}) {
    const configured = (dependencyConfig(options).packageManager || '').toLowerCase()
    if (configured && configured !== 'auto') return configured
    const resolved = (options.packageManager || '').toLowerCase()
    if (resolved === 'npm' || resolved === 'pnpm' || resolved === 'yarn') return resolved
    if (fs.existsSync(path.join(process.cwd(), 'pnpm-lock.yaml'))) return 'pnpm'
    if (fs.existsSync(path.join(process.cwd(), 'yarn.lock'))) return 'yarn'
    return 'npm'
}

function installCommand(pm, options = {}) {
    const allowScripts = dependencyConfig(options).allowInstallScripts === true
    if (pm === 'npm') return { command: 'npm', args: ['install', '--omit=dev', '--package-lock=false', ...(allowScripts ? [] : ['--ignore-scripts'])] }
    if (pm === 'yarn') return { command: 'yarn', args: ['install', '--production=true', '--no-lockfile', ...(allowScripts ? [] : ['--ignore-scripts'])] }
    if (pm === 'pnpm') return { command: 'pnpm', args: ['install', '--prod', '--ignore-workspace', '--no-lockfile', '--reporter=append-only', '--package-import-method=copy', ...(allowScripts ? [] : ['--ignore-scripts'])] }
    throw new Error(`[deps] Invalid dependencyResolution.packageManager "${pm}". Expected auto, npm, pnpm, or yarn.`)
}

function installEnv() {
    return { ...process.env, CI: process.env.CI || 'true' }
}

function stableHash(value) {
    return crypto.createHash('sha256').update(JSON.stringify(value)).digest('hex').slice(0, 24)
}

function dependencyCacheRoot() {
    return path.join(process.cwd(), 'node_modules', '.cache', 'opencore', 'dependencies')
}

function dependencyCacheKey(dependencies, pm, options = {}) {
    return stableHash({
        dependencies,
        packageManager: pm,
        allowInstallScripts: dependencyConfig(options).allowInstallScripts === true,
        platform: process.platform,
        arch: process.arch,
        node: process.versions.node.split('.')[0],
    })
}

function dependencyMarkerPath(outDir) {
    return path.join(outDir, '.opencore-deps.json')
}

async function sleep(ms) {
    await new Promise(resolve => setTimeout(resolve, ms))
}

async function withCacheLock(cacheDir, fn) {
    const lockDir = `${cacheDir}.lock`
    await fs.promises.mkdir(path.dirname(lockDir), { recursive: true })
    const started = Date.now()
    while (true) {
        try {
            await fs.promises.mkdir(lockDir, { recursive: false })
            break
        } catch (error) {
            if (error.code !== 'EEXIST') throw error
            if (Date.now() - started > 5 * 60 * 1000) {
                await fs.promises.rm(lockDir, { recursive: true, force: true }).catch(() => {})
                continue
            }
            await sleep(200)
        }
    }

    try {
        return await fn()
    } finally {
        await fs.promises.rm(lockDir, { recursive: true, force: true }).catch(() => {})
    }
}

async function copyDirPhysical(src, dst) {
    await fs.promises.rm(dst, { recursive: true, force: true })
    await fs.promises.mkdir(path.dirname(dst), { recursive: true })
    await fs.promises.cp(src, dst, { recursive: true, force: true, dereference: true })
}

async function readDependencyMarker(outDir) {
    try {
        return JSON.parse(await fs.promises.readFile(dependencyMarkerPath(outDir), 'utf8'))
    } catch (e) {
        return null
    }
}

async function writeDependencyMarker(outDir, marker) {
    await fs.promises.writeFile(dependencyMarkerPath(outDir), JSON.stringify(marker, null, 2))
}

async function cleanupDependencyArtifacts(outDir) {
    await fs.promises.rm(path.join(outDir, 'node_modules'), { recursive: true, force: true }).catch(() => {})
    await fs.promises.rm(dependencyMarkerPath(outDir), { force: true }).catch(() => {})
}

async function writePackageJson(outDir, pkg) {
    await fs.promises.mkdir(outDir, { recursive: true })
    await fs.promises.writeFile(path.join(outDir, 'package.json'), JSON.stringify(pkg, null, 2))
}

async function minimalPackage(resourcePath, outDir, packageNames) {
    const resourcePkg = await readPackageJson(resourcePath)
    const dependencies = await resolveDependencyVersions(resourcePath, packageNames)
    const pkg = {
        name: resourcePkg?.name || path.basename(outDir),
        version: resourcePkg?.version || '0.0.0',
        private: true,
        type: resourcePkg?.type,
        dependencies,
    }
    if (!pkg.type) delete pkg.type
    return pkg
}

async function validateSandboxPaths(resourceDir) {
    const root = await fs.promises.realpath(resourceDir)
    async function walk(current) {
        const entries = await fs.promises.readdir(current, { withFileTypes: true })
        for (const entry of entries) {
            const entryPath = path.join(current, entry.name)
            const stat = await fs.promises.lstat(entryPath)
            if (stat.isSymbolicLink()) {
                const target = await fs.promises.realpath(entryPath)
                if (target !== root && !target.startsWith(root + path.sep)) {
                    throw new Error(`[deps] Sandbox validation failed: symlink ${entryPath} resolves outside resource (${target}).`)
                }
            }
            if (entry.isDirectory()) await walk(entryPath)
        }
    }
    await walk(resourceDir)
}

async function linkNodeModules(resourcePath, outDir) {
    const srcModules = path.join(path.resolve(resourcePath), 'node_modules')
    const dstModules = path.join(path.resolve(outDir), 'node_modules')
    if (!fs.existsSync(srcModules)) return

    console.warn('[deps] Warning: symlink mode may fail under the FXServer Node.js 22 filesystem sandbox. Use mode "isolated" for production.')
    if (fs.existsSync(dstModules)) await fs.promises.rm(dstModules, { recursive: true, force: true })
    await fs.promises.symlink(srcModules, dstModules, 'junction')
}

async function installIsolatedDependencies(resourcePath, outDir, options = {}) {
    const packageNames = normalizedExternals(options)
    if (packageNames.length === 0) return

    await fs.promises.mkdir(outDir, { recursive: true })
    const pkg = await minimalPackage(resourcePath, outDir, packageNames)
    await writePackageJson(outDir, pkg)

    const nodeModules = path.join(outDir, 'node_modules')
    const pm = getInstallPackageManager(options)
    const useCache = dependencyConfig(options).cache !== false
    const cacheKey = dependencyCacheKey(pkg.dependencies, pm, options)
    const existingMarker = await readDependencyMarker(outDir)

    if (existingMarker?.cacheKey === cacheKey && fs.existsSync(nodeModules)) {
        console.log(`[deps] Reused isolated dependencies in ${path.basename(outDir)}`)
        return
    }

    if (fs.existsSync(nodeModules)) await fs.promises.rm(nodeModules, { recursive: true, force: true })

    if (useCache) {
        const cacheDir = path.join(dependencyCacheRoot(), cacheKey)
        const cacheReady = path.join(cacheDir, '.ready')

        await withCacheLock(cacheDir, async () => {
            if (!fs.existsSync(cacheReady)) {
                await fs.promises.rm(cacheDir, { recursive: true, force: true })
                await writePackageJson(cacheDir, { name: `opencore-deps-${cacheKey}`, version: '0.0.0', private: true, dependencies: pkg.dependencies })
                const { command, args } = installCommand(pm, options)
                try {
                    await execFileAsync(command, args, { cwd: cacheDir, env: installEnv(), maxBuffer: 1024 * 1024 * 10 })
                } catch (error) {
                    const output = [error.stdout, error.stderr].filter(Boolean).join('\n')
                    throw new Error(`[deps] Failed to install isolated dependency cache with ${pm}: ${error.message}${output ? `\n${output}` : ''}`)
                }
                await fs.promises.writeFile(cacheReady, JSON.stringify({ dependencies: pkg.dependencies, packageManager: pm }, null, 2))
            }
        })

        await copyDirPhysical(path.join(cacheDir, 'node_modules'), nodeModules)
    } else {
        const { command, args } = installCommand(pm, options)
        try {
            await execFileAsync(command, args, { cwd: outDir, env: installEnv(), maxBuffer: 1024 * 1024 * 10 })
        } catch (error) {
            const output = [error.stdout, error.stderr].filter(Boolean).join('\n')
            throw new Error(`[deps] Failed to install isolated dependencies with ${pm}: ${error.message}${output ? `\n${output}` : ''}`)
        }
    }

    if (dependencyConfig(options).verifySandboxPaths !== false) {
        await validateSandboxPaths(outDir)
    }
    await writeDependencyMarker(outDir, { cacheKey, dependencies: pkg.dependencies, packageManager: pm, sandboxValidated: dependencyConfig(options).verifySandboxPaths !== false })
}

async function aggregateSharedDependencies(entries = []) {
    const dependencies = {}
    const owners = {}

    for (const entry of entries) {
        const resourcePath = path.resolve(entry.resourcePath)
        const packageNames = []
        const seen = new Set()
        for (const external of entry.externals || []) {
            const normalized = normalizeExternalImport(external)
            if (normalized && !seen.has(normalized)) {
                seen.add(normalized)
                packageNames.push(normalized)
            }
        }

        const resolved = await resolveDependencyVersions(resourcePath, packageNames)
        for (const [name, spec] of Object.entries(resolved)) {
            if (dependencies[name] && dependencies[name] !== spec) {
                throw new Error(`[deps] Shared dependency conflict for "${name}": ${owners[name]} requires "${dependencies[name]}", ${entry.resourcePath} requires "${spec}".`)
            }
            dependencies[name] = spec
            owners[name] = entry.resourcePath
        }
    }

    return dependencies
}

async function generateSharedDependencyResource(outDir, options = {}) {
    const sharedResourceName = options.sharedResourceName || '__opencore_deps'
    const absOutDir = path.resolve(outDir)
    await fs.promises.mkdir(absOutDir, { recursive: true })

    const dependencies = await aggregateSharedDependencies(options.dependencies || [])
    const pkg = {
        name: sharedResourceName,
        version: '0.0.0',
        private: true,
        dependencies,
    }

    await fs.promises.writeFile(path.join(absOutDir, 'package.json'), JSON.stringify(pkg, null, 2))
    await fs.promises.writeFile(path.join(absOutDir, 'noop.js'), '// OpenCore shared dependency resource.\n')
    await fs.promises.writeFile(path.join(absOutDir, 'fxmanifest.lua'), [
        "fx_version 'cerulean'",
        "game 'gta5'",
        "node_version '22'",
        "server_script 'noop.js'",
        '',
    ].join('\n'))

    if (Object.keys(dependencies).length > 0) {
        const pm = getInstallPackageManager({ packageManager: options.packageManager, dependencyResolution: options })
        const cacheKey = dependencyCacheKey(dependencies, pm, { dependencyResolution: options })
        const nodeModules = path.join(absOutDir, 'node_modules')
        const existingMarker = await readDependencyMarker(absOutDir)

        if (existingMarker?.cacheKey === cacheKey && fs.existsSync(nodeModules)) {
            console.log(`[deps] Reused shared dependency resource ${sharedResourceName}`)
            return
        }

        if (fs.existsSync(nodeModules)) await fs.promises.rm(nodeModules, { recursive: true, force: true })
        const useCache = options.cache !== false
        if (useCache) {
            const cacheDir = path.join(dependencyCacheRoot(), cacheKey)
            const cacheReady = path.join(cacheDir, '.ready')

            await withCacheLock(cacheDir, async () => {
                if (!fs.existsSync(cacheReady)) {
                    await fs.promises.rm(cacheDir, { recursive: true, force: true })
                    await writePackageJson(cacheDir, { name: `opencore-deps-${cacheKey}`, version: '0.0.0', private: true, dependencies })
                    const { command, args } = installCommand(pm, { dependencyResolution: options })
                    try {
                        await execFileAsync(command, args, { cwd: cacheDir, env: installEnv(), maxBuffer: 1024 * 1024 * 10 })
                    } catch (error) {
                        const output = [error.stdout, error.stderr].filter(Boolean).join('\n')
                        throw new Error(`[deps] Failed to install shared dependency cache with ${pm}: ${error.message}${output ? `\n${output}` : ''}`)
                    }
                    await fs.promises.writeFile(cacheReady, JSON.stringify({ dependencies, packageManager: pm }, null, 2))
                }
            })

            await copyDirPhysical(path.join(cacheDir, 'node_modules'), nodeModules)
        } else {
            const { command, args } = installCommand(pm, { dependencyResolution: options })
            try {
                await execFileAsync(command, args, { cwd: absOutDir, env: installEnv(), maxBuffer: 1024 * 1024 * 10 })
            } catch (error) {
                const output = [error.stdout, error.stderr].filter(Boolean).join('\n')
                throw new Error(`[deps] Failed to install shared dependencies with ${pm}: ${error.message}${output ? `\n${output}` : ''}`)
            }
        }
        await writeDependencyMarker(absOutDir, { cacheKey, dependencies, packageManager: pm, sandboxValidated: options.verifySandboxPaths !== false })
    } else {
        await cleanupDependencyArtifacts(absOutDir)
    }

    if (options.verifySandboxPaths !== false) await validateSandboxPaths(absOutDir)
    console.log(`[deps] Generated shared dependency resource ${sharedResourceName}`)
}

async function handleDependencies(resourcePath, outDir, options = {}) {
    const absSrcPath = path.resolve(resourcePath)
    const absOutDir = path.resolve(outDir)
    if (absSrcPath === absOutDir) return

    const mode = dependencyMode(options)
    if (mode === 'symlink') {
        const pkgSrc = path.join(absSrcPath, 'package.json')
        if (fs.existsSync(pkgSrc)) {
            const pkgContent = JSON.parse(await fs.promises.readFile(pkgSrc, 'utf8'))
            delete pkgContent.devDependencies
            delete pkgContent.scripts
            await fs.promises.writeFile(path.join(absOutDir, 'package.json'), JSON.stringify(pkgContent, null, 2))
        }
        await linkNodeModules(absSrcPath, absOutDir)
        return
    }

    if (mode === 'shared-resource') {
        return
    }

    if (mode === 'bundle') {
        return
    }

    await installIsolatedDependencies(absSrcPath, absOutDir, options)
}

function shouldHandleDependencies(options = {}) {
    if (dependencyMode(options) === 'bundle') return false
    const serverBuildOptions = getBuildOptions('server', options)
    const serverExternals = serverBuildOptions !== null ? normalizedExternals(options) : []
    return Array.isArray(serverExternals) && serverExternals.length > 0
}

module.exports = {
    handleDependencies,
    shouldHandleDependencies,
    normalizeExternalImport,
    resolveDependencyVersions,
    validateSandboxPaths,
    checkBundleCompatibility,
    cleanupDependencyArtifacts,
    generateSharedDependencyResource,
    detectNativePackages,
    printNativePackageWarnings
}
