import path from 'node:path'
import { defineConfig, type UserConfig } from 'vite'

export type OpenCoreViteConfigOptions = {
  root: string
  outDir?: string
}

export function createOpenCoreViteConfig(options: OpenCoreViteConfigOptions): UserConfig {
  const root = path.resolve(options.root)

  return defineConfig({
    root,
    base: './',
    resolve: {
      alias: {
        '@': path.join(root, 'src'),
      },
    },
    css: {
      devSourcemap: true,
    },
    build: {
      outDir: options.outDir ?? path.join(root, 'dist'),
      emptyOutDir: true,
      sourcemap: false,
      target: 'es2020',
    },
  })
}

export default defineConfig(({ mode }) => {
  const root = process.env.OPENCORE_VIEW_ROOT || process.cwd()
  const outDir = process.env.OPENCORE_VIEW_OUTDIR

  return createOpenCoreViteConfig({
    root,
    outDir,
  })
})
