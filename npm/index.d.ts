/**
 * @fileoverview Type definitions for @open-core/cli
 * 
 * OpenCore CLI is the official build tool for OpenCore Framework projects.
 * It compiles TypeScript resources for FiveM servers with full decorator support.
 * 
 * @example
 * ```typescript
 * // opencore.config.ts
 * import { defineConfig } from '@open-core/cli'
 * 
 * export default defineConfig({
 *   name: 'my-server',
 *   outDir: './build',
 *   core: {
 *     path: './core',
 *     resourceName: '[core]',
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
   * @default 'vanilla'
   */
  framework?: 'react' | 'vue' | 'svelte' | 'solid' | 'vanilla';

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
   * @example ['*.config.ts', '*.config.js', 'test/**', '**\/*.spec.ts']
   */
  ignore?: string[];
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
 * @example
 * ```typescript
 * build: {
 *   // Control what to build
 *   compileServer: true,
 *   compileClient: true,
 *   nui: false,
 *
 *   // Global settings for this resource
 *   minify: true,
 *   sourceMaps: false,
 *
 *   // Server-specific settings
 *   server: {
 *     platform: 'node',
 *     external: ['typeorm', 'pg'],
 *   },
 *
 *   // Client-specific settings
 *   client: {
 *     platform: 'browser',
 *     external: ['three'],
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
   * Applies to both server and client unless overridden in server/client config.
   */
  minify?: boolean;

  /**
   * Whether to generate source maps for this specific resource.
   * Overrides the global build.sourceMaps setting.
   * Applies to both server and client unless overridden in server/client config.
   */
  sourceMaps?: boolean;

  /**
   * Server-side build configuration for this resource.
   * Set to `false` to skip server build.
   * Set to `true` or omit to use defaults.
   * Set to object to customize server build options.
   *
   * @example false // Skip server build
   * @example true // Build with defaults
   * @example { platform: 'node', external: ['pg'] } // Custom config
   */
  server?: boolean | SideBuildConfig;

  /**
   * Client-side build configuration for this resource.
   * Set to `false` to skip client build.
   * Set to `true` or omit to use defaults.
   * Set to object to customize client build options.
   *
   * @example false // Skip client build
   * @example true // Build with defaults
   * @example { platform: 'browser', external: [] } // Custom config
   */
  client?: boolean | SideBuildConfig;

  // Legacy support for backward compatibility
  /**
   * @deprecated Use `server.target` and `client.target` instead
   */
  target?: string;

  /**
   * @deprecated Use `server.platform` and `client.platform` instead
   */
  platform?: 'node' | 'browser' | 'neutral';

  /**
   * @deprecated Use `server.format` and `client.format` instead
   */
  format?: 'iife' | 'cjs' | 'esm';

  /**
   * @deprecated Use `server.external` and `client.external` instead
   */
  external?: string[];
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
 * standalone: {
 *   include: ['./standalone/*'],
 *   explicit: [
 *     { path: './standalone/utils', compile: true },
 *     { path: './standalone/legacy', compile: false },  // Just copy
 *   ],
 * }
 * ```
 */
export interface StandaloneConfig {
  /**
   * Glob patterns to include standalone resources.
   * @example ['./standalone/*']
   */
  include?: string[];

  /**
   * Explicitly configured standalone resources.
   */
  explicit?: ExplicitResource[];
}

/**
 * Build configuration for server or client side.
 * These settings control how the code is compiled.
 *
 * @example
 * ```typescript
 * {
 *   platform: 'node',
 *   format: 'iife',
 *   target: 'es2020',
 *   external: ['typeorm', 'pg'],
 * }
 * ```
 */
export interface SideBuildConfig {
  /**
   * Build platform for esbuild.
   * @default 'node' for server, 'browser' for client
   * @example 'node' | 'browser' | 'neutral'
   */
  platform?: 'node' | 'browser' | 'neutral';

  /**
   * Output format for the bundle.
   * @default 'iife'
   * @example 'iife' | 'cjs' | 'esm'
   */
  format?: 'iife' | 'cjs' | 'esm';

  /**
   * JavaScript target version.
   * @default 'es2020'
   * @example 'es2020' | 'es2021' | 'esnext'
   */
  target?: string;

  /**
   * Packages to mark as external (not bundled).
   * @default []
   * @example ['typeorm', 'pg'] // For server with DB packages
   * @example ['three'] // For client with large 3D libraries
   */
  external?: string[];

  /**
   * Whether to minify the output code.
   * If not set, uses the global minify setting.
   * @example true
   */
  minify?: boolean;

  /**
   * Whether to generate inline source maps.
   * If not set, uses the global sourceMaps setting.
   * @example true
   */
  sourceMaps?: boolean;
}

/**
 * Global build configuration.
 * These settings apply to all resources unless overridden.
 *
 * @example
 * ```typescript
 * build: {
 *   minify: true,
 *   sourceMaps: false,
 *   parallel: true,
 *   maxWorkers: 8,
 *
 *   server: {
 *     platform: 'node',
 *     format: 'iife',
 *     target: 'es2020',
 *     external: [],
 *   },
 *
 *   client: {
 *     platform: 'browser',
 *     format: 'iife',
 *     target: 'es2020',
 *     external: [],
 *   },
 * }
 * ```
 */
export interface BuildConfig {
  /**
   * Whether to minify the output code.
   * Reduces file size but makes debugging harder.
   * Applies to both server and client unless overridden.
   * @default false
   */
  minify?: boolean;

  /**
   * Whether to generate inline source maps.
   * Useful for debugging in development.
   * Applies to both server and client unless overridden.
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
   * Server runs in FiveM with full Node.js 22 support.
   */
  server?: SideBuildConfig;

  /**
   * Client-side build configuration.
   * Client runs in a browser-like environment in the game.
   */
  client?: SideBuildConfig;

  // Legacy support for backward compatibility
  /**
   * @deprecated Use `server.platform` and `client.platform` instead
   */
  platform?: 'node' | 'browser' | 'neutral';

  /**
   * @deprecated Use `server.format` and `client.format` instead
   */
  format?: 'iife' | 'cjs' | 'esm';

  /**
   * @deprecated Use `server.target` and `client.target` instead
   */
  target?: string;

  /**
   * @deprecated Use `server.external` and `client.external` instead
   */
  external?: string[];
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
   * Output directory for compiled resources.
   * This folder is **cleaned before each build**.
   * @default './build'
   * @example './build'
   */
  outDir?: string;

  /**
   * Deployment destination path.
   * If set, compiled resources will be copied here after build.
   * Typically points to your FiveM server's resources folder.
   * @example 'C:/FXServer/server-data/resources/[my-server]'
   */
  destination?: string;

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
  standalone?: StandaloneConfig;

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
 *   outDir: './build',
 *   core: {
 *     path: './core',
 *     resourceName: '[core]',
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
