import { defineConfig } from '@open-core/cli'
// If you get a missing packages error, you should run `pnpm i` to install everything.

export default defineConfig({
  name: '{{.ProjectName}}',

  // Mandatory: Deploy to FiveM server
  // Here you must add the path where your FiveM resources are located.
  destination: '{{.Destination}}',

  core: {
    path: './core',
    resourceName: 'core',
    entryPoints: {
      server: './core/src/server.ts',
      client: './core/src/client.ts',
    },
  },

  // Satellite resources (depend on core at runtime)
  resources: {
    include: ['./resources/*'],
  },

  // Standalone resources (no core dependency)
  standalones: {
    include: ['./standalones/*'],
  },
{{ if .InstallIdentity }}
  modules: ['@open-core/identity'],
{{ end }}
  build: {
    logLevel: 'INFO', // INFO by default
    minify: {{.UseMinify}}, // If you want to debug the compiled JS, you can set it to 'false' but it makes the build heavier.
    sourceMaps: false, // It's also useful for debugging, but it makes the build very large.
    parallel: true,
    maxWorkers: 8,

    server: {
      format: 'cjs',
      platform: 'node',
      target: 'es2023',
    },

    client: {
      format: 'iife',
      platform: 'neutral',
      target: 'es2020',
    },
  },

  dev: {
    port: 3847,
    txAdminUser: '',
    txAdminPassword: '',
    txAdminUrl: ''
  }
})
