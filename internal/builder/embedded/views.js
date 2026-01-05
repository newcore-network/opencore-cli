const path = require('path')
const fs = require('fs')
const { getEsbuild } = require('./plugins')
const { getSharedConfig } = require('./config')

function readOcIgnore(viewPath) {
    const ocignorePath = path.join(viewPath, '.ocignore')
    if (!fs.existsSync(ocignorePath)) {
        return []
    }

    try {
        const content = fs.readFileSync(ocignorePath, 'utf8')
        return content
            .split('\n')
            .map(line => line.trim())
            .filter(line => line && !line.startsWith('#'))
    } catch (error) {
        console.warn(`[views] Failed to read .ocignore: ${error.message}`)
        return []
    }
}

function shouldIgnore(filePath, ignorePatterns) {
    const fileName = path.basename(filePath)
    for (const pattern of ignorePatterns) {
        if (pattern === fileName) return true
        if (pattern.startsWith('*.')) {
            const ext = pattern.slice(1)
            if (fileName.endsWith(ext)) return true
        }
        if (filePath.includes(pattern)) return true
    }
    return false
}

async function copyStaticAssets(viewPath, outDir, ignorePatterns = []) {
    const defaultIgnore = [
        'node_modules',
        '.git',
        'package.json',
        'package-lock.json',
        'tsconfig.json',
        '.ocignore',
        'index.html', // Handled separately with script path replacement
        '*.ts',
        '*.tsx',
        '*.jsx',
    ]
    const allIgnore = [...defaultIgnore, ...ignorePatterns]

    async function copyDir(srcDir, dstDir, relativePath = '') {
        const entries = await fs.promises.readdir(srcDir, { withFileTypes: true })

        for (const entry of entries) {
            const srcPath = path.join(srcDir, entry.name)
            const dstPath = path.join(dstDir, entry.name)
            const relPath = path.join(relativePath, entry.name)

            if (shouldIgnore(relPath, allIgnore)) continue

            if (entry.isDirectory()) {
                await fs.promises.mkdir(dstPath, { recursive: true })
                await copyDir(srcPath, dstPath, relPath)
            } else {
                // Skip files that esbuild already handles via imports
                const ext = path.extname(entry.name).toLowerCase()
                const esbuildExtensions = ['.js', '.ts', '.tsx', '.jsx']
                if (esbuildExtensions.includes(ext)) continue

                // Copy static assets (css, images, fonts, etc.)
                const dstDir = path.dirname(dstPath)
                await fs.promises.mkdir(dstDir, { recursive: true })
                await fs.promises.copyFile(srcPath, dstPath)
            }
        }
    }

    await copyDir(viewPath, outDir)
}

async function buildViews(viewPath, outDir, options = {}) {
    const esbuild = getEsbuild()
    const shared = getSharedConfig(options)

    await fs.promises.mkdir(outDir, { recursive: true })

    let entryPoint = null

    if (options.viewEntry) {
        const explicitEntry = path.join(viewPath, options.viewEntry)
        if (fs.existsSync(explicitEntry)) {
            entryPoint = explicitEntry
            console.log(`[views] Using explicit entry point: ${options.viewEntry}`)
        } else {
            throw new Error(`[views] Configured entry point not found: ${options.viewEntry}`)
        }
    }

    if (!entryPoint) {
        const possibleEntries = [
            path.join(viewPath, 'index.tsx'),
            path.join(viewPath, 'index.jsx'),
            path.join(viewPath, 'index.ts'),
            path.join(viewPath, 'index.js'),
            path.join(viewPath, 'main.tsx'),
            path.join(viewPath, 'main.jsx'),
            path.join(viewPath, 'main.ts'),
            path.join(viewPath, 'main.js'),
            path.join(viewPath, 'app.tsx'),
            path.join(viewPath, 'app.jsx'),
            path.join(viewPath, 'app.ts'),
            path.join(viewPath, 'app.js'),
            path.join(viewPath, 'src/index.tsx'),
            path.join(viewPath, 'src/index.ts'),
            path.join(viewPath, 'src/main.tsx'),
            path.join(viewPath, 'src/main.ts'),
            path.join(viewPath, 'src/app.tsx'),
            path.join(viewPath, 'src/app.ts'),
        ]

        for (const entry of possibleEntries) {
            if (fs.existsSync(entry)) {
                entryPoint = entry
                break
            }
        }

        if (!entryPoint) {
            const errorMsg = `[views] No entry point found in ${viewPath}\nSearched for: ${possibleEntries.map(p => path.basename(p)).join(', ')}`
            console.error(errorMsg)
            throw new Error(errorMsg)
        }

        console.log(`[views] Auto-detected entry point: ${path.relative(viewPath, entryPoint)}`)
    }

    await esbuild.build({
        ...shared,
        entryPoints: [entryPoint],
        outdir: outDir,
        platform: 'browser',
        target: options.target || 'es2020',
        format: 'esm',
        bundle: true,
        splitting: true,
        chunkNames: 'chunks/[name]-[hash]',
        assetNames: 'assets/[name]-[hash]',
        loader: {
            '.tsx': 'tsx',
            '.jsx': 'jsx',
            '.css': 'css',
            '.svg': 'file',
            '.png': 'file',
            '.jpg': 'file',
            '.gif': 'file',
            '.woff': 'file',
            '.woff2': 'file',
        },
        define: {
            'process.env.NODE_ENV': options.minify ? '"production"' : '"development"',
        },
    })

    // Copy static assets (CSS, images, fonts, etc.) that aren't imported in JS
    const ignorePatterns = readOcIgnore(viewPath)
    await copyStaticAssets(viewPath, outDir, ignorePatterns)

    const htmlSrc = path.join(viewPath, 'index.html')
    const htmlDst = path.join(outDir, 'index.html')
    if (fs.existsSync(htmlSrc)) {
        let html = await fs.promises.readFile(htmlSrc, 'utf8')
        const entryBase = path.basename(entryPoint, path.extname(entryPoint))
        
        html = html.replace(
            /(<script[^>]*\ssrc=["'])([^"']+\.(ts|tsx|jsx|js))(['"][^>]*>)/gi,
            (match, prefix, src, ext, suffix) => {
                if (src.includes(entryBase)) {
                    return prefix + entryBase + '.js' + suffix
                }
                return match
            }
        )

        await fs.promises.writeFile(htmlDst, html, 'utf8')
        console.log(`[views] Processed and copied index.html`)
    }

    console.log(`[views] Built ${path.basename(viewPath)}`)
}

module.exports = {
    buildViews
}
