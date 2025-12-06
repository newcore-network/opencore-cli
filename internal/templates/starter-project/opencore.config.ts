import { defineConfig } from '@open-core/cli'

export default defineConfig({
  name: '{{.ProjectName}}',
  architecture: '{{.Architecture}}',
  outDir: './dist/resources',
  
  core: {
    path: './core',
    resourceName: '[core]',
  },
  
  resources: {
    include: ['./resources/*'],
  },
  {{if .InstallIdentity}}
  modules: ['@open-core/identity'],
  {{end}}
  build: {
    minify: {{.UseMinify}},
    sourceMaps: true,
  }
})

