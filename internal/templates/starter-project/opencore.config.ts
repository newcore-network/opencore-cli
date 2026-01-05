import { defineConfig } from '@open-core/cli'
// If you get a missing packages error, you should run `pnpm i` to install everything.

export default defineConfig({
  name: '{{.ProjectName}}',
  // Mandatory: Deploy to FiveM server
  destination: 'C:/FXServer/server-data/resources/[{{.ProjectName}}]',

  core: {
    path: './core',
    resourceName: 'core',
    entryPoints: {
      server: './core/src/server.ts',
      client: './core/src/client.ts',
    },
    // Optional: Use custom build script instead of CLI's embedded compiler
    // customCompiler: './scripts/core-build.js',
    //
    // Optional: Views/NUI for core
    // views: {
    //   path: './core/views',
    //   framework: 'react',
    // },
  },

  // Satellite resources (depend on core at runtime)
  resources: {
    include: ['./resources/*'],
    // explicit: [
    //   {
    //     path: './resources/admin',
    //     resourceName: 'admin-panel',
    //     build: { client: true, nui: true },
    //     customCompiler: './scripts/admin-build.js',  // Optional
    //     views: {
    //       path: './resources/admin/ui',
    //       framework: 'react',
    //     },
    //   },
    // ],
  },

  // Standalone resources (no core dependency)
  // standalone: {
  //   include: ['./standalone/*'],
  //   explicit: [
  //     { path: './standalone/utils', compile: true },
  //     { path: './standalone/legacy', compile: false },  // Just copy, no build
  //     { path: './standalone/custom', customCompiler: './scripts/custom-build.js' },
  //   ],
  // },
{{ if .InstallIdentity }}
  modules: ['@open-core/identity'],
  {{ end }}
  build: {
    minify: {{.UseMinify}},
    sourceMaps: false,
    target: 'ES2020',
    parallel: true,
    maxWorkers: 8,
  },

  dev: {
    port: 3847,
    // or you can use enviroment variables

    // VAR: OPENCORE_TXADMIN_USER
    txAdminUser: '',
    // VAR: OPENCORE_TXADMIN_PASSWORD
    txAdminPassword: '',
    // VAR: OPENCORE_TXADMIN_URL
    txAdminUrl: ''
  }
})
