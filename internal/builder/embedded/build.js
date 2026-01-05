const path = require('path')
const { buildCore, buildResource, buildStandalone, copyResource } = require('./build_functions')
const { buildViews } = require('./views')

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
