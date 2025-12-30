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
 *   target: 'ES2020',
 *   parallel: true,
 *   maxWorkers: 8,
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
   * FiveM supports ES2020 features.
   * @default 'ES2020'
   */
  target?: string;

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
 */
export interface DevConfig {
  /**
   * Port for the framework's hot-reload server.
   * This should match the port configured in the framework.
   * @default 3847
   */
  port?: number;
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
