import { defineConfig } from '@open-core/cli'
{{ if .InstallFiveMAdapter }}import { FiveMClientAdapter } from '@open-core/fivem-adapter/client'
import { FiveMServerAdapter } from '@open-core/fivem-adapter/server'
{{ else if .InstallRageMPAdapter }}import { RageMPClientAdapter } from '@open-core/ragemp-adapter/client'
import { RageMPServerAdapter } from '@open-core/ragemp-adapter/server'
{{ end }}
// If you get a missing packages error, install dependencies in the project root.

export default defineConfig({
  name: '{{.ProjectName}}',

{{ if .InstallRageMPAdapter }}  // Mandatory: deploy to your RageMP server root.
  // OpenCore will place server files under packages/ and client files under client_packages/.
{{ else }}  // Mandatory: Deploy to FiveM server
  // Here you must add the path where your FiveM resources are located.
{{ end }}
  destination: '{{.Destination}}',

{{ if .InstallFiveMAdapter }}  adapter: {
    server: FiveMServerAdapter(),
    client: FiveMClientAdapter(),
  },

{{ else if .InstallRageMPAdapter }}  adapter: {
    server: RageMPServerAdapter(),
    client: RageMPClientAdapter(),
  },

{{ end }}

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
  build: {
    logLevel: 'INFO', // INFO by default
    minify: {{.UseMinify}}, // If you want to debug the compiled JS, you can set it to 'false' but it makes the build heavier.
    sourceMaps: false, // It's also useful for debugging, but it makes the build very large.
    parallel: true,
    maxWorkers: 8,
{{ if .InstallRageMPAdapter }}    target: 'node14',
{{ end }}
  },

  dev: {
    port: 3847,
    // you can also set your env system
    // txAdminUser: '',
    // txAdminPassword: '',
    // txAdminUrl: ''
  }
})
