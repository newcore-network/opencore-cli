import { defineConfig } from '@open-core/cli'
// If you get a missing packages error, install dependencies in the project root.

export default defineConfig({
  name: '{{.ProjectName}}',

  // Mandatory: Deploy to FiveM server
  // Here you must add the path where your FiveM resources are located.
  destination: '{{.Destination}}',

  core: {
    path: './core',
    resourceName: 'core'
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
  },

  dev: {
    port: 3847,
    // you can also set your env system
    // txAdminUser: '',
    // txAdminPassword: '',
    // txAdminUrl: ''
  }
})
