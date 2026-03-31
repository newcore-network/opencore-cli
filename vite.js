const fs = require('node:fs')
const path = require('node:path')
const { defineConfig } = require('vite')

const POSTCSS_CONFIG_FILES = [
  'postcss.config.js',
  'postcss.config.cjs',
  'postcss.config.mjs',
  'postcss.config.ts',
]

const PROJECT_ROOT_MARKERS = [
  'opencore.config.ts',
  'opencore.config.js',
  'opencore.config.mjs',
  'opencore.config.cjs',
]

function findProjectRoot(startDir) {
  let current = path.resolve(startDir)

  while (true) {
    for (const marker of PROJECT_ROOT_MARKERS) {
      if (fs.existsSync(path.join(current, marker))) {
        return current
      }
    }

    const parent = path.dirname(current)
    if (parent === current) {
      return null
    }
    current = parent
  }
}

function findPostcssConfig(projectRoot) {
  if (!projectRoot) {
    return null
  }

  for (const fileName of POSTCSS_CONFIG_FILES) {
    const configPath = path.join(projectRoot, fileName)
    if (fs.existsSync(configPath)) {
      return configPath
    }
  }

  return null
}

function resolveViewRoot(inputRoot) {
  return path.resolve(inputRoot || process.env.OPENCORE_VIEW_ROOT || process.cwd())
}

function resolveProjectRoot(viewRoot, explicitProjectRoot) {
  if (explicitProjectRoot) {
    return path.resolve(explicitProjectRoot)
  }

  return findProjectRoot(viewRoot) || process.cwd()
}

function resolveOutDir(viewRoot, explicitOutDir, buildConfig) {
  if (buildConfig && buildConfig.outDir) {
    return buildConfig.outDir
  }

  const outDir = explicitOutDir || process.env.OPENCORE_VIEW_OUTDIR
  if (outDir) {
    return path.resolve(outDir)
  }

  return path.join(viewRoot, 'dist')
}

function resolveCssConfig(cssConfig, postcssConfigPath) {
  if (!postcssConfigPath || cssConfig?.postcss !== undefined) {
    return cssConfig
  }

  return {
    ...cssConfig,
    postcss: postcssConfigPath,
  }
}

function createOpenCoreViteConfig(config = {}) {
  const { opencore, build, css, ...rest } = config
  const viewRoot = resolveViewRoot(opencore?.root)
  const projectRoot = resolveProjectRoot(viewRoot, opencore?.projectRoot)
  const postcssConfigPath =
    opencore?.postcss === false
      ? null
      : typeof opencore?.postcss === 'string'
        ? path.resolve(projectRoot, opencore.postcss)
        : findPostcssConfig(projectRoot)

  return defineConfig({
    ...rest,
    root: viewRoot,
    base: rest.base ?? './',
    css: resolveCssConfig(css, postcssConfigPath),
    build: {
      ...build,
      outDir: resolveOutDir(viewRoot, opencore?.outDir, build),
      emptyOutDir: build?.emptyOutDir ?? true,
      target: build?.target ?? opencore?.target ?? 'es2020',
    },
  })
}

module.exports = {
  createOpenCoreViteConfig,
}
