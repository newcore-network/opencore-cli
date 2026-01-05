const path = require('path')
const fs = require('fs')
const { getBuildOptions, getExternals } = require('./config')

// =============================================================================
// FiveM Runtime
// =============================================================================
// Server: Full Node.js runtime with all Node APIs available
// Client: Neutral JavaScript runtime with limitations:
//   - NO Node.js APIs (no fs, path, http, child_process, etc.)
//   - NO Web APIs (no DOM, fetch, localStorage, etc.)
//   - NO native C++ bindings (.node files)
//   - Only pure JavaScript/ES2020 code works
// =============================================================================

// Known native packages that won't work in FiveM (use C++ bindings)
const KNOWN_NATIVE_PACKAGES = [
    'bcrypt',
    'sharp',
    'canvas',
    'sqlite3',
    'better-sqlite3',
    'node-gyp',
    'node-pre-gyp',
    'fsevents',
    'esbuild',
    'swc',
    '@swc/core',
    'lightningcss',
    'sodium-native',
    'argon2',
    'cpu-features',
    'microtime',
    'bufferutil',
    'utf-8-validate',
]

// Packages with known pure JS alternatives that work in FiveM
const NATIVE_ALTERNATIVES = {
    'bcrypt': 'bcryptjs',
    'argon2': 'hash.js or js-sha3',
    'sharp': 'jimp',
    'canvas': 'pureimage',
    'sqlite3': 'sql.js',
    'better-sqlite3': 'sql.js',
}

/**
 * Check if a package has native bindings
 */
async function isNativePackage(packagePath) {
    try {
        // Check for binding.gyp (node-gyp build file)
        if (fs.existsSync(path.join(packagePath, 'binding.gyp'))) {
            return { isNative: true, reason: 'has binding.gyp (C++ addon)' }
        }

        // Check for .node files (compiled binaries)
        const prebuildsPath = path.join(packagePath, 'prebuilds')
        if (fs.existsSync(prebuildsPath)) {
            return { isNative: true, reason: 'has prebuilt binaries' }
        }

        // Check for build directory with .node files
        const buildPath = path.join(packagePath, 'build')
        if (fs.existsSync(buildPath)) {
            const files = await fs.promises.readdir(buildPath, { recursive: true }).catch(() => [])
            for (const file of files) {
                if (file.endsWith('.node')) {
                    return { isNative: true, reason: 'has compiled .node binaries' }
                }
            }
        }

        // Check package.json for gypfile or binary fields
        const pkgJsonPath = path.join(packagePath, 'package.json')
        if (fs.existsSync(pkgJsonPath)) {
            const pkgJson = JSON.parse(await fs.promises.readFile(pkgJsonPath, 'utf8'))
            if (pkgJson.gypfile || pkgJson.binary) {
                return { isNative: true, reason: 'package.json declares native bindings' }
            }
        }

        return { isNative: false }
    } catch (e) {
        return { isNative: false }
    }
}

/**
 * Scan node_modules for native packages and warn about them
 */
async function detectNativePackages(nodeModulesPath, externals = []) {
    const warnings = []
    const errors = []

    if (!fs.existsSync(nodeModulesPath)) {
        return { warnings, errors }
    }

    // Check known native packages
    for (const pkgName of KNOWN_NATIVE_PACKAGES) {
        const pkgPath = path.join(nodeModulesPath, pkgName)
        const pnpmPath = path.join(nodeModulesPath, '.pnpm')

        let found = false
        let foundPath = pkgPath

        // Check direct path
        if (fs.existsSync(pkgPath)) {
            found = true
        }

        // Check pnpm structure
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
                // Native package marked as external - will fail at runtime
                errors.push({
                    package: pkgName,
                    message: `"${pkgName}" is a native C++ package and WILL NOT WORK in FiveM runtime.`,
                    hint: alternative
                        ? `Use "${alternative}" instead.`
                        : `This package requires Node.js native bindings that are incompatible with FiveM.`,
                    severity: 'error'
                })
            } else {
                // Native package being bundled - will fail during bundling or runtime
                warnings.push({
                    package: pkgName,
                    message: `"${pkgName}" is a native C++ package being bundled.`,
                    hint: alternative
                        ? `Consider using "${alternative}" instead.`
                        : `This package may cause build errors or runtime failures.`,
                    severity: 'warning'
                })
            }
        }
    }

    return { warnings, errors }
}

/**
 * Print native package warnings/errors to console
 */
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

async function handleDependencies(resourcePath, outDir) {
    const absSrcPath = path.resolve(resourcePath)
    const absOutDir = path.resolve(outDir)

    if (absSrcPath === absOutDir) {
        return
    }

    const pkgSrc = path.join(absSrcPath, 'package.json')
    const pkgDst = path.join(absOutDir, 'package.json')

    if (fs.existsSync(pkgSrc)) {
        try {
            const pkgContent = JSON.parse(await fs.promises.readFile(pkgSrc, 'utf8'))
            delete pkgContent.devDependencies
            delete pkgContent.scripts 
            await fs.promises.writeFile(pkgDst, JSON.stringify(pkgContent, null, 2))
        } catch (e) {
            console.warn(`[deps] Failed to process package.json: ${e.message}`)
        }
    }

    const srcModules = path.join(absSrcPath, 'node_modules')
    const dstModules = path.join(absOutDir, 'node_modules')

    if (fs.existsSync(srcModules)) {
        try {
            if (fs.existsSync(dstModules)) {
                const stat = await fs.promises.lstat(dstModules)
                if (stat.isSymbolicLink()) {
                    const linkTarget = await fs.promises.readlink(dstModules)
                    if (path.resolve(linkTarget) === path.resolve(srcModules)) {
                        return
                    }
                }
                await fs.promises.rm(dstModules, { recursive: true, force: true })
            }
            await fs.promises.symlink(srcModules, dstModules, 'junction')
        } catch (e) {
            console.warn(`[deps] Failed to link node_modules: ${e.message}`)
        }
    }
}

function shouldHandleDependencies(options = {}) {
    // Only server can use externals - client must bundle everything
    // FiveM client has no filesystem access, so node_modules is useless there
    const serverBuildOptions = getBuildOptions('server', options)
    const serverExternals = serverBuildOptions !== null ? getExternals('server', options) : []

    return Array.isArray(serverExternals) && serverExternals.length > 0
}

module.exports = {
    handleDependencies,
    shouldHandleDependencies,
    detectNativePackages,
    printNativePackageWarnings
}
