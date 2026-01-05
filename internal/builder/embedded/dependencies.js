const path = require('path')
const fs = require('fs')
const { getBuildOptions, getExternals } = require('./config')

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
    const serverBuildOptions = getBuildOptions('server', options)
    const clientBuildOptions = getBuildOptions('client', options)

    const serverExternals = serverBuildOptions !== null ? getExternals('server', options) : []
    const clientExternals = clientBuildOptions !== null ? getExternals('client', options) : []

    return (Array.isArray(serverExternals) && serverExternals.length > 0) ||
        (Array.isArray(clientExternals) && clientExternals.length > 0)
}

module.exports = {
    handleDependencies,
    shouldHandleDependencies
}
