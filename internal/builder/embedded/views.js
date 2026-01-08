const path = require('path')
const fs = require('fs')
const { getEsbuild } = require('./plugins')
const { getSharedConfig } = require('./config')

function hasFilesWithExtension(dir, extension) {
    try {
        const entries = fs.readdirSync(dir, { withFileTypes: true })
        for (const entry of entries) {
            if (entry.name === 'node_modules' || entry.name === '.git') continue
            const fullPath = path.join(dir, entry.name)
            if (entry.isDirectory()) {
                if (hasFilesWithExtension(fullPath, extension)) return true
            } else if (entry.name.endsWith(extension)) {
                return true
            }
        }
    } catch (e) {
        // Ignore errors
    }
    return false
}

function hasSvelteFiles(dir) {
    return hasFilesWithExtension(dir, '.svelte')
}

function hasVueFiles(dir) {
    return hasFilesWithExtension(dir, '.vue')
}

function hasReactFiles(dir) {
    return hasFilesWithExtension(dir, '.tsx') || hasFilesWithExtension(dir, '.jsx')
}

function checkDependency(name) {
    try {
        require.resolve(name)
        return true
    } catch (e) {
        return false
    }
}

function getSveltePlugin() {
    const missing = []
    if (!checkDependency('esbuild-svelte')) missing.push('esbuild-svelte')
    if (!checkDependency('svelte')) missing.push('svelte')

    if (missing.length > 0) {
        throw new Error(
            `\n` +
            `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n` +
            `  [views] Missing Svelte dependencies\n` +
            `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n` +
            `\n` +
            `  Svelte files (.svelte) were detected but required\n` +
            `  dependencies are not installed.\n` +
            `\n` +
            `  Missing: ${missing.join(', ')}\n` +
            `\n` +
            `  Run this command to install:\n` +
            `\n` +
            `    pnpm add -D esbuild-svelte svelte\n` +
            `\n` +
            `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n`
        )
    }

    const sveltePlugin = require('esbuild-svelte')
    return sveltePlugin({
        compilerOptions: {
            css: 'injected',
        },
    })
}

function getVuePlugin() {
    const missing = []
    if (!checkDependency('esbuild-plugin-vue3')) missing.push('esbuild-plugin-vue3')
    if (!checkDependency('vue')) missing.push('vue')

    if (missing.length > 0) {
        throw new Error(
            `\n` +
            `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n` +
            `  [views] Missing Vue dependencies\n` +
            `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n` +
            `\n` +
            `  Vue files (.vue) were detected but required\n` +
            `  dependencies are not installed.\n` +
            `\n` +
            `  Missing: ${missing.join(', ')}\n` +
            `\n` +
            `  Run this command to install:\n` +
            `\n` +
            `    pnpm add -D esbuild-plugin-vue3 vue\n` +
            `\n` +
            `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n`
        )
    }

    const vuePlugin = require('esbuild-plugin-vue3')
    return vuePlugin()
}

function checkReactDependencies(viewPath) {
    const missing = []
    if (!checkDependency('react')) missing.push('react')
    if (!checkDependency('react-dom')) missing.push('react-dom')

    if (missing.length > 0) {
        throw new Error(
            `\n` +
            `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n` +
            `  [views] Missing React dependencies\n` +
            `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n` +
            `\n` +
            `  React files (.tsx/.jsx) were detected but required\n` +
            `  dependencies are not installed.\n` +
            `\n` +
            `  Missing: ${missing.join(', ')}\n` +
            `\n` +
            `  Run this command to install:\n` +
            `\n` +
            `    pnpm add react react-dom\n` +
            `\n` +
            `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n`
        )
    }
}

function hasSassFiles(dir) {
    return hasFilesWithExtension(dir, '.scss') || hasFilesWithExtension(dir, '.sass')
}

function getSassPlugin() {
    if (!checkDependency('esbuild-sass-plugin')) {
        throw new Error(
            `\n` +
            `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n` +
            `  [views] Missing SASS/SCSS dependencies\n` +
            `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n` +
            `\n` +
            `  SASS/SCSS files (.scss/.sass) were detected but required\n` +
            `  dependencies are not installed.\n` +
            `\n` +
            `  Missing: esbuild-sass-plugin\n` +
            `\n` +
            `  Run this command to install:\n` +
            `\n` +
            `    pnpm add -D esbuild-sass-plugin sass\n` +
            `\n` +
            `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n`
        )
    }

    const { sassPlugin } = require('esbuild-sass-plugin')
    return sassPlugin()
}

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
        'pnpm-lock.yaml',
        'yarn.lock',
        'tsconfig.json',
        '.ocignore',
        'index.html', // Handled separately with script path replacement
        '*.ts',
        '*.tsx',
        '*.jsx',
        '*.svelte',
        '*.vue',
        '*.scss',
        '*.sass',
    ]
    const allIgnore = [...defaultIgnore, ...ignorePatterns]

    async function getFilesToCopy(srcDir, relativePath = '') {
        const files = []
        const entries = await fs.promises.readdir(srcDir, { withFileTypes: true })

        for (const entry of entries) {
            const srcPath = path.join(srcDir, entry.name)
            const relPath = path.join(relativePath, entry.name)

            if (shouldIgnore(relPath, allIgnore)) continue

            if (entry.isDirectory()) {
                const subFiles = await getFilesToCopy(srcPath, relPath)
                files.push(...subFiles)
            } else {
                // Skip files that esbuild already handles via imports
                const ext = path.extname(entry.name).toLowerCase()
                const esbuildExtensions = ['.js', '.ts', '.tsx', '.jsx', '.svelte', '.vue', '.scss', '.sass']
                if (esbuildExtensions.includes(ext)) continue

                files.push({ src: srcPath, rel: relPath })
            }
        }
        return files
    }

    const files = await getFilesToCopy(viewPath)
    for (const file of files) {
        const dstPath = path.join(outDir, file.rel)
        await fs.promises.mkdir(path.dirname(dstPath), { recursive: true })
        await fs.promises.copyFile(file.src, dstPath)
    }
}

async function buildViews(viewPath, outDir, options = {}) {
    await fs.promises.mkdir(outDir, { recursive: true })

    const esbuild = getEsbuild()
    const shared = getSharedConfig(options)

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
            // Svelte entry points
            path.join(viewPath, 'index.svelte'),
            path.join(viewPath, 'main.svelte'),
            path.join(viewPath, 'App.svelte'),
            path.join(viewPath, 'src/index.svelte'),
            path.join(viewPath, 'src/main.svelte'),
            path.join(viewPath, 'src/App.svelte'),
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

    // Load framework plugins based on file detection
    const plugins = []

    if (hasReactFiles(viewPath)) {
        console.log(`[views] React files detected, checking dependencies...`)
        checkReactDependencies(viewPath)
    }
    if (hasSvelteFiles(viewPath)) {
        console.log(`[views] Svelte files detected, loading svelte plugin...`)
        plugins.push(getSveltePlugin())
    }
    if (hasVueFiles(viewPath)) {
        console.log(`[views] Vue files detected, loading vue plugin...`)
        plugins.push(getVuePlugin())
    }
    if (hasSassFiles(viewPath)) {
        console.log(`[views] SASS/SCSS files detected, loading sass plugin...`)
        plugins.push(getSassPlugin())
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
        plugins,
        loader: {
            // JavaScript/TypeScript
            '.tsx': 'tsx',
            '.jsx': 'jsx',
            // Styles
            '.css': 'css',
            // Images
            '.svg': 'file',
            '.png': 'file',
            '.jpg': 'file',
            '.jpeg': 'file',
            '.gif': 'file',
            '.webp': 'file',
            '.ico': 'file',
            '.bmp': 'file',
            '.avif': 'file',
            // Fonts
            '.woff': 'file',
            '.woff2': 'file',
            '.ttf': 'file',
            '.otf': 'file',
            '.eot': 'file',
            // Audio
            '.mp3': 'file',
            '.wav': 'file',
            '.ogg': 'file',
            '.m4a': 'file',
            // Video
            '.mp4': 'file',
            '.webm': 'file',
            '.ogv': 'file',
            // Data
            '.json': 'json',
            '.txt': 'text',
            // Other
            '.pdf': 'file',
            '.xml': 'text',
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
            /(<script[^>]*\ssrc=["'])([^"']+\.(ts|tsx|jsx|js|svelte|vue))(['"][^>]*>)/gi,
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
