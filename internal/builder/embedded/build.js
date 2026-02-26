const path = require('path')
const { buildCore, buildResource, buildStandalone, copyResource } = require('./build_functions')
const { buildViews } = require('./views')

/**
 * Check if a dependency is installed
 */
function checkDependency(name) {
    try {
        require.resolve(name)
        return true
    } catch (e) {
        return false
    }
}

/**
 * Get package manager from options
 */
function getPackageManager(options = {}) {
    const pm = (options.packageManager || '').toLowerCase()
    if (pm === 'pnpm' || pm === 'yarn' || pm === 'npm') return pm
    return 'pnpm'
}

/**
 * Get dev install command based on package manager
 */
function devInstallCmd(options = {}, pkgs = []) {
    const pm = getPackageManager(options)
    const args = pkgs.join(' ')
    if (pm === 'yarn') return `yarn add -D ${args}`
    if (pm === 'npm') return `npm install -D ${args}`
    return `pnpm add -D ${args}`
}

/**
 * Verify required base dependencies are installed
 */
function checkBaseDependencies(options = {}) {
    const required = [
        { name: 'esbuild', install: 'esbuild' },
        { name: '@swc/core', install: '@swc/core' },
    ]

    const missing = required.filter(dep => !checkDependency(dep.name))

    if (missing.length > 0) {
        const names = missing.map(d => d.name).join(', ')
        const installCmd = devInstallCmd(options, missing.map(d => d.install))
        throw new Error(
            `\n` +
            `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n` +
            `  [build] Missing required dependencies\n` +
            `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n` +
            `\n` +
            `  The following dependencies are required but not installed:\n` +
            `\n` +
            `  Missing: ${names}\n` +
            `\n` +
            `  Run this command to install:\n` +
            `\n` +
            `    ${installCmd}\n` +
            `\n` +
            `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n`
        )
    }
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
        case 'copy':
            return copyResource(resourcePath, outDir, options)
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
            // Check base dependencies before building
            checkBaseDependencies(options)

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
