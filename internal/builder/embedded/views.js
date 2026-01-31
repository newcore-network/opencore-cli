const path = require('path')
const fs = require('fs')
const { getEsbuild } = require('./plugins')
const { getSharedConfig } = require('./config')

function getPackageManager(options = {}) {
    const pm = (options.packageManager || '').toLowerCase()
    if (pm === 'pnpm' || pm === 'yarn' || pm === 'npm') return pm
    return 'pnpm'
}

function addCmd(options = {}, pkgs = [], isDev = false) {
    const pm = getPackageManager(options)
    const args = pkgs.join(' ')
    if (pm === 'yarn') return isDev ? `yarn add -D ${args}` : `yarn add ${args}`
    if (pm === 'npm') return isDev ? `npm install -D ${args}` : `npm install ${args}`
    return isDev ? `pnpm add -D ${args}` : `pnpm add ${args}`
}

function execCmd(options = {}, bin, args = []) {
    const pm = getPackageManager(options)
    const rest = args.length ? ` ${args.join(' ')}` : ''
    if (pm === 'yarn') return `yarn ${bin}${rest}`
    if (pm === 'npm') return `npm exec -- ${bin}${rest}`
    return `pnpm ${bin}${rest}`
}

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

function hasAstroFiles(dir) {
    return hasFilesWithExtension(dir, '.astro')
}

function checkDependency(name) {

    try {
        require.resolve(name)
        return true
    } catch (e) {
        return false
    }
}

function resolveDependency(viewPath, name) {
    try {
        return require.resolve(name, { paths: [viewPath, process.cwd()] })
    } catch (e) {
        return null
    }
}

function detectAstroFramework(viewPath) {
    if (hasAstroFiles(viewPath)) {
        return true
    }

    const pkgPath = path.join(viewPath, 'package.json')
    if (fs.existsSync(pkgPath)) {
        try {
            const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8'))
            const deps = { ...pkg.dependencies, ...pkg.devDependencies }
            return typeof deps.astro === 'string'
        } catch (e) {
            return false
        }
    }

    return false
}

function readAstroConfig(viewPath) {
    const configFiles = [
        'astro.config.mjs',
        'astro.config.cjs',
        'astro.config.js',
        'astro.config.ts',
    ]

    for (const fileName of configFiles) {
        const filePath = path.join(viewPath, fileName)
        if (fs.existsSync(filePath)) {
            return filePath
        }
    }

    return null
}

function validateAstroOutput(viewPath) {
    const configPath = readAstroConfig(viewPath)
    if (!configPath) {
        throw new Error(
            `\n` +
            `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n` +
            `  [views] Astro output must be static\n` +
            `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n` +
            `\n` +
            `  Astro config not found. Please add astro.config.* with:\n` +
            `    export default defineConfig({ output: 'static' })\n` +
            `\n` +
            `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n`
        )
    }

    try {
        const content = fs.readFileSync(configPath, 'utf8')
        if (!content.includes('output') || !content.includes('static')) {
            throw new Error(
                `\n` +
                `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n` +
                `  [views] Astro output must be static\n` +
                `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n` +
                `\n` +
                `  Astro detected but config does not specify output: 'static'.\n` +
                `\n` +
                `  Update ${path.basename(configPath)} to include:\n` +
                `    export default defineConfig({ output: 'static' })\n` +
                `\n` +
                `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n`
            )
        }
    } catch (error) {
        if (error instanceof Error) {
            throw error
        }
        throw new Error(String(error))
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

async function buildAstroViews(viewPath, outDir, options = {}) {
    const astroPath = resolveDependency(viewPath, 'astro')
    if (!astroPath) {
        throw new Error(
            `\n` +
            `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n` +
            `  [views] Missing Astro dependency\n` +
            `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n` +
            `\n` +
            `  Astro framework was detected but the package is not installed.\n` +
            `\n` +
            `  Missing: astro\n` +
            `\n` +
            `  Run this command to install:\n` +
            `\n` +
            `    ${addCmd(options, ['astro'], true)}\n` +
            `\n` +
            `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n`
        )
    }

    validateAstroOutput(viewPath)

    const buildCommand = options.buildCommand || execCmd(options, 'astro', ['build'])
    const outputDir = options.outputDir || 'dist'

    console.log(`[views] Astro detected, running: ${buildCommand}`)

    const spawn = require('child_process').spawn

    await new Promise((resolve, reject) => {
        const proc = spawn(buildCommand, { cwd: viewPath, stdio: 'inherit', shell: true })
        proc.on('close', code => {
            if (code === 0) {
                resolve()
            } else {
                reject(new Error(`[views] Astro build failed with exit code ${code}`))
            }
        })
    })

    const outputPath = path.join(viewPath, outputDir)
    if (!fs.existsSync(outputPath)) {
        throw new Error(`[views] Astro output directory not found: ${outputPath}`)
    }

    await copyDirContents(outputPath, outDir)
    console.log(`[views] Astro build copied to ${outDir}`)
}

function findTailwindConfig(viewPath) {
    const configFiles = [
        'tailwind.config.js',
        'tailwind.config.cjs',
        'tailwind.config.mjs',
        'tailwind.config.ts',
    ]

    let currentDir = path.resolve(viewPath)
    const rootDir = path.parse(currentDir).root
    const stopDir = path.resolve(process.cwd())

    while (true) {
        for (const fileName of configFiles) {
            const candidate = path.join(currentDir, fileName)
            if (fs.existsSync(candidate)) {
                return candidate
            }
        }

        if (currentDir === stopDir || currentDir === rootDir) {
            break
        }

        currentDir = path.dirname(currentDir)
    }

    return null
}

function getTailwindInfo(viewPath, options = {}) {
    const configPath = findTailwindConfig(viewPath)
    const packagePath = resolveDependency(viewPath, 'tailwindcss/package.json')

    if (!configPath && !packagePath) {
        return null
    }

    if (!packagePath) {
        throw new Error(
            `\n` +
            `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n` +
            `  [views] Missing Tailwind dependency\n` +
            `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n` +
            `\n` +
            `  Tailwind config was detected but the package is not installed.\n` +
            `\n` +
            `  Missing: tailwindcss\n` +
            `\n` +
            `  Run this command to install:\n` +
            `\n` +
            `    ${addCmd(options, ['tailwindcss'], true)}\n` +
            `\n` +
            `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n`
        )
    }

    if (!configPath) {
        console.log(`[views] Tailwind package detected, but no tailwind.config.* found.`)
    }

    const pkg = JSON.parse(fs.readFileSync(packagePath, 'utf8'))
    const major = parseInt((pkg.version || '0').split('.')[0], 10)
    const majorVersion = Number.isNaN(major) ? 3 : major

    return {
        configPath,
        version: pkg.version || '0.0.0',
        major: majorVersion,
    }
}

function ensureTailwindDependencies(viewPath, tailwindInfo, options = {}) {
    const missing = []

    if (!resolveDependency(viewPath, 'postcss')) {
        missing.push('postcss')
    }

    if (tailwindInfo.major >= 4) {
        if (!resolveDependency(viewPath, '@tailwindcss/postcss')) {
            missing.push('@tailwindcss/postcss')
        }
    } else {
        if (!resolveDependency(viewPath, 'autoprefixer')) {
            missing.push('autoprefixer')
        }
    }

    if (missing.length > 0) {
        throw new Error(
            `\n` +
            `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n` +
            `  [views] Missing Tailwind dependencies\n` +
            `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n` +
            `\n` +
            `  Tailwind CSS was detected but required dependencies are not installed.\n` +
            `\n` +
            `  Missing: ${missing.join(', ')}\n` +
            `\n` +
            `  Run this command to install:\n` +
            `\n` +
            `    ${addCmd(options, missing, true)}\n` +
            `\n` +
            `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n`
        )
    }
}

function createTailwindPlugin(viewPath, options = {}) {
    const tailwindInfo = getTailwindInfo(viewPath, options)
    if (!tailwindInfo) {
        return null
    }

    ensureTailwindDependencies(viewPath, tailwindInfo, options)

    const postcssPath = resolveDependency(viewPath, 'postcss')
    const postcss = require(postcssPath)
    const plugins = []

    if (tailwindInfo.major >= 4) {
        const pluginPath = resolveDependency(viewPath, '@tailwindcss/postcss')
        const tailwindPlugin = require(pluginPath)
        const pluginOptions = tailwindInfo.configPath ? { config: tailwindInfo.configPath } : {}
        plugins.push(tailwindPlugin(pluginOptions))
        const autoprefixerPath = resolveDependency(viewPath, 'autoprefixer')
        const autoprefixer = autoprefixerPath ? require(autoprefixerPath) : null
        if (autoprefixer) {
            plugins.push(autoprefixer())
        }
    } else {
        const pluginPath = resolveDependency(viewPath, 'tailwindcss')
        const tailwindPlugin = require(pluginPath)
        const pluginOptions = tailwindInfo.configPath ? { config: tailwindInfo.configPath } : {}
        plugins.push(tailwindPlugin(pluginOptions))
        const autoprefixerPath = resolveDependency(viewPath, 'autoprefixer')
        const autoprefixer = autoprefixerPath ? require(autoprefixerPath) : null
        if (autoprefixer) {
            plugins.push(autoprefixer())
        }
    }

    console.log(`[views] Tailwind detected (v${tailwindInfo.version})`)

    return {
        name: 'tailwindcss',
        setup(build) {
            build.onLoad({ filter: /\.css$/ }, async (args) => {
                const source = await fs.promises.readFile(args.path, 'utf8')
                const result = await postcss(plugins).process(source, {
                    from: args.path,
                    map: false,
                })

                return {
                    contents: result.css,
                    loader: 'css',
                }
            })
        },
    }
}

function getSveltePlugin(options = {}) {

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
            `    ${addCmd(options, ['esbuild-svelte', 'svelte'], true)}\n` +
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

function getVuePlugin(options = {}) {
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
            `    ${addCmd(options, ['esbuild-plugin-vue3', 'vue'], true)}\n` +
            `\n` +
            `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n`
        )
    }

    const vuePlugin = require('esbuild-plugin-vue3')
    return vuePlugin()
}

function checkReactDependencies(viewPath, options = {}) {
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
            `    ${addCmd(options, ['react', 'react-dom'], false)}\n` +
            `\n` +
            `━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n`
        )
    }
}

function hasSassFiles(dir) {
    return hasFilesWithExtension(dir, '.scss') || hasFilesWithExtension(dir, '.sass')
}

function getSassPlugin(options = {}) {
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
            `    ${addCmd(options, ['esbuild-sass-plugin', 'sass'], true)}\n` +
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

function isForceIncluded(filePath, forceInclude) {
    if (!forceInclude || forceInclude.length === 0) {
        return false
    }
    const fileName = path.basename(filePath)
    for (const pattern of forceInclude) {
        if (pattern === fileName) return true
        if (pattern.startsWith('*.')) {
            const ext = pattern.slice(1)
            if (fileName.endsWith(ext)) return true
        }
    }
    return false
}

async function copyStaticAssets(viewPath, outDir, ignorePatterns = [], forceInclude = []) {
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

            if (shouldIgnore(relPath, allIgnore) && !isForceIncluded(relPath, forceInclude)) continue

            if (entry.isDirectory()) {
                const subFiles = await getFilesToCopy(srcPath, relPath)
                files.push(...subFiles)
            } else {
                // Skip files that esbuild already handles via imports
                const ext = path.extname(entry.name).toLowerCase()
                const esbuildExtensions = ['.js', '.ts', '.tsx', '.jsx', '.svelte', '.vue', '.scss', '.sass']
                if (esbuildExtensions.includes(ext) && !isForceIncluded(relPath, forceInclude)) continue

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

    const isAstro = (options.framework || '').toLowerCase() === 'astro' || detectAstroFramework(viewPath)
    if (isAstro) {
        await buildAstroViews(viewPath, outDir, options)
        return
    }

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
        checkReactDependencies(viewPath, options)
    }
    if (hasAstroFiles(viewPath)) {
        console.log(`[views] Astro files detected, running static build...`)
    }
    if (hasSvelteFiles(viewPath)) {
        console.log(`[views] Svelte files detected, loading svelte plugin...`)
        plugins.push(getSveltePlugin(options))
    }
    if (hasVueFiles(viewPath)) {
        console.log(`[views] Vue files detected, loading vue plugin...`)
        plugins.push(getVuePlugin(options))
    }
    if (hasSassFiles(viewPath)) {
        console.log(`[views] SASS/SCSS files detected, loading sass plugin...`)
        plugins.push(getSassPlugin(options))
    }

    const tailwindPlugin = createTailwindPlugin(viewPath, options)
    if (tailwindPlugin) {
        plugins.push(tailwindPlugin)
    }

    await esbuild.build({
        ...shared,
        banner: {
            js: "", // No reflect-metadata for views
        },
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
    await copyStaticAssets(viewPath, outDir, ignorePatterns, options.forceInclude || [])

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
