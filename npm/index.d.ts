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
 * 
 * @example
 * ```typescript
 * build: {
 *   server: true,
 *   client: true,
 *   nui: false,
 *   minify: true,
 * }
 * ```
 */
export interface ResourceBuildConfig {
  /**
   * Whether to compile server-side code.
   * @default true
   */
  server?: boolean;

  /**
   * Whether to compile client-side code.
   * @default true (if client folder exists)
   */
  client?: boolean;

  /**
   * Whether this resource has NUI (web interface).
   * @default false
   */
  nui?: boolean;

  /**
   * Whether to minify the output for this specific resource.
   * Overrides the global build.minify setting.
   */
  minify?: boolean;

  /**
   * Whether to generate source maps for this specific resource.
   * Overrides the global build.sourceMaps setting.
   */
  sourceMaps?: boolean;
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
 *     build: { client: true, nui: true },
 *     views: {
 *       path: './resources/admin/ui',
 *       framework: 'react',
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
 * Global build configuration.
 * These settings apply to all resources unless overridden.
 *
 * @example
 * ```typescript
 * build: {
 *   minify: true,
 *   sourceMaps: true,
 *   target: 'es2020',
 *   platform: 'node',
 *   format: 'iife',
 *   parallel: true,
 *   maxWorkers: 8,
 *   external: [],
 * }
 * ```
 */
export interface BuildConfig {
  /**
   * Whether to minify the output code.
   * Reduces file size but makes debugging harder.
   * @default false
   */
  minify?: boolean;

  /**
   * Whether to generate inline source maps.
   * Useful for debugging in development.
   * @default false
   */
  sourceMaps?: boolean;

  /**
   * JavaScript target version.
   * FiveM supports ES2020+ features with Node.js 22.
   * @default 'es2020'
   * @example 'es2020' | 'es2021' | 'esnext'
   */
  target?: string;

  /**
   * Build platform for esbuild.
   * FiveM server supports full Node.js runtime.
   * @default 'node'
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
   * Packages to mark as external (not bundled).
   * Use this to exclude packages from the bundle.
   *
   * **Note**: With FiveM Node.js 22 support, most packages can be bundled.
   * Only use external for packages you want to load separately at runtime.
   *
   * @default []
   * @example ['some-large-package', 'optional-dependency']
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
 *   },
 *   
 *   resources: {
 *     include: ['./resources/*'],
 *   },
 *   
 *   build: {
 *     minify: true,
 *     parallel: true,
 *     maxWorkers: 8,
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
