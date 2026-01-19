/**
 * @fileoverview Type definitions for @open-core/cli
 *
 * OpenCore CLI is the official build tool for OpenCore Framework projects.
 * It compiles TypeScript resources for FiveM servers with full decorator support.
 *
 * ## FiveM Runtime Environments
 *
 * FiveM has **three distinct runtime environments**:
 *
 * ### Server (Node.js)
 * - Full Node.js runtime with all APIs
 * - Can use `external` packages
 * - FiveM server APIs available
 * - Platform: `node`
 *
 * ### Client (Neutral JS)
 * - NO Node.js APIs, NO Web APIs
 * - FiveM client APIs + GTA natives available
 * - All dependencies MUST be bundled
 * - Platform: `neutral`
 *
 * ### Views/NUI (Web Browser)
 * - Embedded Chromium browser
 * - Standard Web APIs (DOM, fetch, etc.)
 * - Some version limitations (not fully documented)
 * - Platform: `browser`
 *
 * ### Incompatible Packages (Client)
 *
 * These packages use C++ bindings and will NOT work on client:
 * - `bcrypt` -> use `bcryptjs`
 * - `sharp` -> use `jimp`
 * - `sqlite3` / `better-sqlite3` -> use `sql.js`
 * - `argon2` -> use `hash.js` or `js-sha3`
 * - `canvas` -> use `pureimage`
 *
 * @example
 * ```typescript
 * // opencore.config.ts
 * import { defineConfig } from '@open-core/cli'
 *
 * export default defineConfig({
 *   name: 'my-server',
 *   destination: 'C:/FXServer/server-data/resources/[my-server]',
 *   core: {
 *     path: './core',
 *     resourceName: 'core',
 *   },
 *   resources: {
 *     include: ['./resources/*'],
 *   },
 *   build: {
 *     minify: true,
 *     parallel: true,
 *   },
 * })
 * ```
 */

/**
 * Log levels supported by the OpenCore Framework.
 */
export type LogLevel = 'TRACE' | 'DEBUG' | 'INFO' | 'WARN' | 'ERROR' | 'FATAL' | 'OFF';

/**
 * Entry points for server and client scripts.
 * Allows overriding the default entry file locations.
 * 
 * @example
 * ```typescript
 * entryPoints: {
 *   server: './core/src/server.ts',    // Default: ./src/server.ts
 *   client: './core/src/client.ts',    // Default: ./src/client.ts
 * }
 * ```
 */
export interface EntryPoints {
  /**
   * Path to the server entry file.
   * This file will be compiled and output as `server.js` in the resource folder.
   * @example './core/src/server.ts'
   */
  server: string;

  /**
   * Path to the client entry file.
   * This file will be compiled and output as `client.js` in the resource folder.
   * @example './core/src/client.ts'
   */
  client: string;
}

/**
 * Configuration for NUI/Views (web interfaces).
 * Used for resources that have a web-based UI component.
 *
 * @example
 * ```typescript
 * views: {
 *   path: './core/views',
 *   framework: 'react',
 *   entryPoint: 'main.tsx',  // Optional: explicit entry point
 *   ignore: ['*.config.ts', 'test/**'],  // Optional: ignore patterns
 * }
 * ```
 */
export interface ViewsConfig {
  /**
   * Path to the views/NUI source folder.
   * This folder should contain your web application source code.
   * @example './core/views'
   */
  path: string;

  /**
   * Frontend framework used for the views.
   * The CLI will use the appropriate build configuration for each framework.
   * Astro is supported only with static output.
   * @default 'vanilla'
   */
  framework?: 'react' | 'vue' | 'svelte' | 'vanilla' | 'astro';


  /**
   * Explicit entry point file for the views build.
   * If not specified, the CLI will auto-detect common entry points:
   * index.tsx, index.ts, main.tsx, main.ts, app.tsx, app.ts, etc.
   *
   * Path is relative to the `path` directory.
   * @example 'main.ng.ts' // For Angular projects
   * @example 'src/index.tsx' // For nested entry points
   */
  entryPoint?: string;

  /**
   * Patterns of files to ignore during the build process.
   * Works in combination with `.ocignore` file if present.
   * Uses glob patterns similar to `.gitignore`.
   *
   * **Note:** `node_modules`, `.git`, and `.ocignore` are always ignored.
   *
   * @example ['*.config.ts', '*.config.js', 'test/**', '**/*.spec.ts']
   */
  ignore?: string[];

  /**
   * Force include static files by name when they are not being copied.
   * Useful when assets are not imported in JS/CSS and are skipped by default.
   * Matching is done by filename only (not full paths).
   *
   * @example ['favicon.ico', 'robots.txt', '*.mp3']
   */
  forceInclude?: string[];

  /**
   * Custom build command for static frameworks like Astro.
   * Defaults to `pnpm astro build` when framework is `astro`.
   */
  buildCommand?: string;

  /**
   * Output directory for static frameworks like Astro.
   * Defaults to `dist` when framework is `astro`.
   */
  outputDir?: string;
}


/**
 * Configuration for the core resource.
 * The core resource is the main entry point that initializes OpenCore Framework
 * and provides shared functionality to all satellite resources.
 *
 * @example
 * ```typescript
 * core: {
 *   path: './core',
 *   resourceName: '[core]',
 *   entryPoints: {
 *     server: './core/src/server.ts',
 *     client: './core/src/client.ts',
 *   },
 *   build: {
 *     platform: 'node',
 *     external: [],
 *   },
 * }
 * ```
 */
export interface CoreConfig {
  /**
   * Path to the core resource source folder.
   * @example './core'
   */
  path: string;

  /**
   * Name of the resource in the FiveM server.
   * This will be the folder name in the output directory.
   * Use brackets for category folders: '[core]', '[my-server]'
   * @example '[core]'
   */
  resourceName: string;

  /**
   * Custom entry points for server and client scripts.
   * If not specified, defaults to `./src/server.ts` and `./src/client.ts`
   * relative to the resource path.
   */
  entryPoints?: EntryPoints;

  /**
   * Configuration for NUI/Views if the core has a web interface.
   */
  views?: ViewsConfig;

  /**
   * Build options specific to the core resource.
   * Overrides the global build configuration.
   */
  build?: ResourceBuildConfig;

  /**
   * Path to a custom build script.
   * Use this if you need custom build logic instead of the CLI's embedded compiler.
   * The script receives build parameters via command line arguments.
   * @example './scripts/core-build.js'
   */
  customCompiler?: string;
}

/**
 * Build options for individual resources.
 * Allows fine-grained control over what gets compiled for each resource.
 * All options override the global build configuration.
 *
 * You can configure server and client builds separately:
 * - Set to `false` to skip that side's build
 * - Set to `true` or omit to use defaults from global config
 * - Set to an object to customize build options for that side
 *
 * @example
 * ```typescript
 * // Example 1: Full-stack resource with custom configs
 * build: {
 *   minify: true,
 *   server: {
 *     platform: 'node',
 *     external: ['typeorm', 'pg'],
 *   },
 *   client: {
 *     platform: 'browser',
 *     external: ['three'],
 *   },
 * }
 *
 * // Example 2: Server-only resource
 * build: {
 *   server: {
 *     platform: 'node',
 *     format: 'cjs',
 *   },
 *   client: false,  // Skip client build
 * }
 *
 * // Example 3: Client-only resource
 * build: {
 *   server: false,  // Skip server build
 *   client: {
 *     platform: 'browser',
 *   },
 * }
 * ```
 */
export interface ResourceBuildConfig {
  /**
   * Whether this resource has NUI (web interface).
   * @default false
   */
  nui?: boolean;

  /**
   * Whether to minify the output for this specific resource.
   * Overrides the global build.minify setting.
   * Applies to both server and client unless overridden in their configs.
   */
  minify?: boolean;

  /**
   * Whether to generate source maps for this specific resource.
   * Overrides the global build.sourceMaps setting.
   * Applies to both server and client unless overridden in their configs.
   */
  sourceMaps?: boolean;

  /**
   * Server-side build configuration for this resource.
   *
   * - `false`: Skip server build entirely
   * - `true` or omit: Build server using global defaults
   * - `object`: Custom server build configuration
   *
   * @example false // No server build
   * @example true // Use global server config
   * @example { platform: 'node', external: ['pg'] }
   */
  server?: boolean | SideBuildConfig;

  /**
   * Client-side build configuration for this resource.
   *
   * - `false`: Skip client build entirely
   * - `true` or omit: Build client using global defaults
   * - `object`: Custom client build configuration
   *
   * @example false // No client build
   * @example true // Use global client config
   * @example { platform: 'browser', external: ['three'] }
   */
  client?: boolean | SideBuildConfig;

  /**
   * Server-only binaries to copy next to server.js.
   * If omitted and a `bin/` folder exists, it is copied automatically.
   * Paths are relative to the resource path.
   *
   * @example ['bin', 'tools/mytool.exe']
   */
  serverBinaries?: string[];
}


/**
 * Configuration for an explicitly defined resource.
 * Use this when you need custom settings for a specific resource
 * instead of using glob patterns.
 *
 * @example
 * ```typescript
 * explicit: [
 *   {
 *     path: './resources/admin',
 *     resourceName: 'admin-panel',
 *     build: {
 *       nui: true,
 *       server: {
 *         platform: 'node',
 *       },
 *       client: {
 *         platform: 'browser',
 *         external: ['react', 'react-dom'],  // Don't bundle React
 *       },
 *     },
 *     views: {
 *       path: './resources/admin/ui',
 *       framework: 'react',
 *     },
 *   },
 *   {
 *     path: './resources/database-bridge',
 *     resourceName: 'db-bridge',
 *     build: {
 *       server: {
 *         platform: 'node',
 *         format: 'cjs',  // Use CommonJS for server
 *         external: ['typeorm', 'pg'],  // External DB packages
 *       },
 *       client: false,  // No client build
 *     },
 *   },
 * ]
 * ```
 */
export interface ExplicitResource {
  /**
   * Path to the resource source folder.
   * @example './resources/admin'
   */
  path: string;

  /**
   * Custom name for the resource in the output.
   * If not specified, uses the folder name from path.
   * @example 'admin-panel'
   */
  resourceName?: string;

  /**
   * Resource type identifier (for internal use).
   */
  type?: string;

  /**
   * Whether to compile this resource.
   * Set to `false` to just copy files without compilation.
   * Useful for legacy Lua resources or pre-compiled code.
   * @default true
   */
  compile?: boolean;

  /**
   * Custom entry points for this resource.
   */
  entryPoints?: EntryPoints;

  /**
   * Build options for this specific resource.
   */
  build?: ResourceBuildConfig;

  /**
   * Views/NUI configuration for this resource.
   */
  views?: ViewsConfig;

  /**
   * Path to a custom build script for this resource.
   * @example './scripts/admin-build.js'
   */
  customCompiler?: string;
}

/**
 * Configuration for satellite resources.
 * Satellite resources depend on the core resource at runtime
 * and use `@open-core/framework` as an external dependency.
 * 
 * @example
 * ```typescript
 * resources: {
 *   include: ['./resources/*'],
 *   explicit: [
 *     { path: './resources/admin', resourceName: 'admin-panel' },
 *   ],
 * }
 * ```
 */
export interface ResourcesConfig {
  /**
   * Glob patterns to include resources.
   * Each matched directory will be compiled as a satellite resource.
   * @example ['./resources/*', './features/*']
   */
  include?: string[];

  /**
   * Explicitly configured resources with custom settings.
   * Use this for resources that need special build configuration.
   */
  explicit?: ExplicitResource[];
}

/**
 * Configuration for standalone resources.
 * Standalone resources do NOT depend on the core resource
 * and are compiled independently with all dependencies bundled.
 * 
 * @example
 * ```typescript
 * standalones: {
 *   include: ['./standalones/*'],
 *   explicit: [
 *     { path: './standalones/utils', compile: true },
 *     { path: './standalones/legacy', compile: false },  // Just copy
 *   ],
 * }
 * ```
 */
export interface StandaloneConfig {
  /**
   * Glob patterns to include standalone resources.
   * @example ['./standalones/*']
   */
  include?: string[];

  /**
   * Explicitly configured standalone resources.
   */
  explicit?: ExplicitResource[];
}

/**
 * Build configuration for server or client side.
 * These settings control how the code is compiled for each environment.
 *
 * ## FiveM Runtime
 *
 * - **Server**: Full Node.js runtime (`platform: 'node'`)
 * - **Client**: Neutral runtime (`platform: 'neutral'`) - no Node/Web APIs
 *
 * ## Client Limitations
 *
 * **IMPORTANT**: Client does NOT support `external` packages.
 * - Client has no filesystem access
 * - Cannot load modules from `node_modules`
 * - All dependencies MUST be bundled into the final `.js` file
 * - If you configure `client.external`, it will be ignored with a warning
 *
 * ## Server
 *
 * Server has full Node.js APIs and CAN use `external` packages.
 *
 * @example Server configuration
 * ```typescript
 * server: {
 *   platform: 'node',      // Full Node.js runtime (default)
 *   format: 'cjs',         // CommonJS format
 *   target: 'es2020',      // ES2020 features
 *   external: [],          // Bundle everything, or specify externals
 * }
 * ```
 *
 * @example Client configuration
 * ```typescript
 * client: {
 *   platform: 'neutral',   // Neutral runtime (default)
 *   format: 'iife',        // IIFE format for FiveM
 *   target: 'es2020',      // ES2020 features
 *   // external: NOT supported - all deps must be bundled
 * }
 * ```
 */
export interface SideBuildConfig {
  /**
   * Build platform for esbuild.
   *
   * - `'node'`: Full Node.js APIs (default for server)
   * - `'neutral'`: No environment-specific APIs (default for client)
   * - `'browser'`: Browser APIs (not recommended for FiveM)
   *
   * @default 'node' for server, 'neutral' for client
   */
  platform?: 'node' | 'browser' | 'neutral';

  /**
   * Output format for the bundle.
   *
   * - `'iife'`: Immediately Invoked Function Expression (recommended for FiveM)
   * - `'cjs'`: CommonJS (module.exports)
   * - `'esm'`: ES Modules (import/export)
   *
   * @default 'iife'
   */
  format?: 'iife' | 'cjs' | 'esm';

  /**
   * JavaScript target version.
   * FiveM supports modern JavaScript features.
   *
   * @default 'es2020'
   * @example 'es2020' | 'es2021' | 'es2022' | 'esnext'
   */
  target?: string;

  /**
   * Packages to mark as external (not bundled).
   * These packages won't be included in the output bundle.
   *
   * ## Server Only
   *
   * **IMPORTANT**: This option is only supported for SERVER builds.
   * Client builds ignore this option because FiveM client has no
   * filesystem access and cannot load external modules.
   *
   * For server, external packages must be available in `node_modules`
   * at runtime. The CLI will symlink `node_modules` to the output directory.
   *
   * ## Recommendation
   *
   * Bundle everything (empty array) for maximum portability.
   * Only use externals for very large packages that cause build issues.
   *
   * @default []
   * @example [] // Bundle everything (recommended)
   * @example ['typeorm', 'pg'] // Server with large database packages
   */
  external?: string[];

  /**
   * Whether to minify the output code for this side.
   * Overrides the global minify setting.
   *
   * @example true // Always minify this side
   * @example false // Never minify this side
   */
  minify?: boolean;

  /**
   * Whether to generate inline source maps for this side.
   * Overrides the global sourceMaps setting.
   *
   * @example true // Generate source maps for debugging
   * @example false // No source maps for production
   */
  sourceMaps?: boolean;
}

/**
 * Global build configuration.
 * These settings apply to all resources unless overridden.
 *
 * ## FiveM Runtime
 *
 * - **Server**: Full Node.js runtime with all APIs
 * - **Client**: Neutral runtime (no Node.js/Web APIs)
 *
 * ## Client vs Server
 *
 * - **Client**: All dependencies MUST be bundled. `external` is NOT supported.
 * - **Server**: Full Node.js, can use `external` packages.
 *
 * @example
 * ```typescript
 * build: {
 *   minify: true,
 *   sourceMaps: false,
 *   parallel: true,
 *   maxWorkers: 8,
 *
 *   // Server: Node.js runtime
 *   server: {
 *     platform: 'node',
 *     format: 'cjs',
 *     target: 'es2020',
 *     external: [],
 *   },
 *
 *   // Client: Neutral runtime, NO externals
 *   client: {
 *     platform: 'neutral',
 *     format: 'iife',
 *     target: 'es2020',
 *   },
 * }
 * ```
 */
export interface BuildConfig {
  /**
   * Default log level for the project.
   * @default 'INFO'
   */
  logLevel?: LogLevel;

  /**
   * Whether to minify the output code.
   * Reduces file size but makes debugging harder.
   * Applies to both server and client unless overridden in their configs.
   * @default false
   */
  minify?: boolean;

  /**
   * Whether to generate inline source maps.
   * Useful for debugging in development.
   * Applies to both server and client unless overridden in their configs.
   * @default false
   */
  sourceMaps?: boolean;

  /**
   * Whether to build resources in parallel.
   * Significantly speeds up builds for projects with many resources.
   * @default false
   */
  parallel?: boolean;

  /**
   * Maximum number of parallel workers.
   * Defaults to the number of CPU cores.
   * @default CPU cores
   */
  maxWorkers?: number;

  /**
   * Server-side build configuration.
   * Server runs in FiveM with full Node.js runtime.
   * Can use `external` packages.
   *
   * @example
   * ```typescript
   * server: {
   *   platform: 'node',
   *   format: 'cjs',
   *   target: 'es2020',
   *   external: [],
   * }
   * ```
   */
  server?: SideBuildConfig;

  /**
   * Client-side build configuration.
   * Client runs in neutral runtime WITHOUT Node.js/Web APIs.
   * All dependencies MUST be bundled - `external` is NOT supported.
   *
   * @example
   * ```typescript
   * client: {
   *   platform: 'neutral',
   *   format: 'iife',
   *   target: 'es2020',
   * }
   * ```
   */
  client?: SideBuildConfig;
}

/**
 * Main OpenCore configuration object.
 * This is the root configuration that defines your entire project structure.
 *
 * @example
 * ```typescript
 * import { defineConfig } from '@open-core/cli'
 *
 * export default defineConfig({
 *   name: 'my-server',
 *   outDir: './build',
 *   destination: 'C:/FXServer/server-data/resources/[my-server]',
 *
 *   core: {
 *     path: './core',
 *     resourceName: '[core]',
 *     build: {
 *       // Core-specific build options
 *       server: {
 *         platform: 'node',
 *         external: [],  // Bundle everything for server
 *       },
 *       client: {
 *         platform: 'browser',
 *         external: [],
 *       },
 *     },
 *   },
 *
 *   resources: {
 *     include: ['./resources/*'],
 *     explicit: [
 *       {
 *         path: './resources/ui-heavy',
 *         build: {
 *           // Client-only resource with custom config
 *           server: false,  // No server build
 *           client: {
 *             platform: 'browser',
 *             external: ['three', 'react'],  // Don't bundle large libs
 *           },
 *         },
 *       },
 *       {
 *         path: './resources/database-service',
 *         build: {
 *           // Server-only resource
 *           server: {
 *             platform: 'node',
 *             external: ['typeorm', 'pg'],  // Node.js packages external
 *           },
 *           client: false,  // No client build
 *         },
 *       },
 *     ],
 *   },
 *
 *   build: {
 *     // Global build options (used as defaults)
 *     minify: true,
 *     sourceMaps: false,
 *     parallel: true,
 *     maxWorkers: 8,
 *
 *     // Default server config
 *     server: {
 *       platform: 'node',
 *       format: 'iife',
 *       target: 'es2020',
 *       external: [],
 *     },
 *
 *     // Default client config
 *     client: {
 *       platform: 'browser',
 *       format: 'iife',
 *       target: 'es2020',
 *       external: [],
 *     },
 *   },
 * })
 * ```
 */
export interface OpenCoreConfig {
  /**
   * Project name.
   * Used for identification and logging.
   * @example 'my-awesome-server'
   */
  name: string;

  /**
   * Deployment destination path.
   * **Required**. Compiled resources will be output directly to this path.
   * Typically points to your FiveM server's resources folder.
   * @example 'C:/FXServer/server-data/resources/[my-server]'
   */
  destination: string;

  /**
   * Core resource configuration.
   * **Required**. The core resource initializes OpenCore Framework.
   */
  core: CoreConfig;

  /**
   * Satellite resources configuration.
   * These resources depend on the core at runtime.
   */
  resources?: ResourcesConfig;

  /**
   * Standalone resources configuration.
   * These resources are independent and don't use the core.
   */
  standalones?: StandaloneConfig;

  /**
   * OpenCore modules to use.
   * @example ['@open-core/identity']
   */
  modules?: string[];

  /**
   * Global build configuration.
   */
  build?: BuildConfig;

  /**
   * Development mode configuration.
   */
  dev?: DevConfig;
}

/**
 * Development mode settings.
 *
 * @example
 * ```typescript
 * dev: {
 *   port: 3847,
 *   // txAdmin integration for CORE hot-reload
 *   txAdminUrl: 'http://localhost:40120',
 *   txAdminUser: 'admin',
 *   txAdminPassword: 'my-password',
 * }
 * ```
 */
export interface DevConfig {
  /**
   * Port for the framework's hot-reload server.
   * This should match the port configured in the framework.
   * @default 3847
   */
  port?: number;

  /**
   * txAdmin panel URL for hot-reload integration.
   * When configured, the CLI will use txAdmin API to restart resources,
   * which allows hot-reloading the CORE resource (not possible via internal HTTP).
   *
   * Can also be set via `OPENCORE_TXADMIN_URL` environment variable.
   * @example 'http://localhost:40120'
   */
  txAdminUrl?: string;

  /**
   * txAdmin username for authentication.
   * The user must have the `commands.resources` permission.
   *
   * Can also be set via `OPENCORE_TXADMIN_USER` environment variable.
   * @example 'admin'
   */
  txAdminUser?: string;

  /**
   * txAdmin password for authentication.
   *
   * **Security note**: For production, prefer using the
   * `OPENCORE_TXADMIN_PASSWORD` environment variable instead.
   */
  txAdminPassword?: string;
}

/**
 * Define OpenCore configuration with full TypeScript support.
 * 
 * This function provides type checking and autocompletion for your configuration.
 * It returns the same object passed in, serving only for type inference.
 * 
 * @param config - OpenCore configuration object
 * @returns The same configuration object (for type inference)
 * 
 * @example
 * ```typescript
 * // opencore.config.ts
 * import { defineConfig } from '@open-core/cli'
 * 
 * export default defineConfig({
 *   name: 'my-server',
 *   destination: 'C:/FXServer/server-data/resources/[my-server]',
 *   core: {
 *     path: './core',
 *     resourceName: 'core',
 *   },
 *   resources: {
 *     include: ['./resources/*'],
 *   },
 *   build: {
 *     minify: true,
 *     parallel: true,
 *   },
 * })
 * ```
 */
export function defineConfig(config: OpenCoreConfig): OpenCoreConfig;
