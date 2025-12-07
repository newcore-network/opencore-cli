import { defineConfig } from '@open-core/cli'

export default defineConfig({
  name: '{{.ProjectName}}',
  architecture: '{{.Architecture}}',
  outDir: './dist/resources',

  core: {
    path: './core',
    resourceName: '[core]',
    entryPoints: {
      server: './core/src/server.ts',
      client: './core/src/client.ts',
    },
  },

  resources: {
    include: ['./resources/*'],
  },

  views: {
    path: './views',
  },
  {{ if .InstallIdentity }}
  modules: ['@open-core/identity'],
  {{ end }}
  build: {
  minify: {{.UseMinify }},
  sourceMaps: true,
  }
})

