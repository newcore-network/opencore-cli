// Type definitions for @open-core/cli

export interface CoreEntryPoints {
  server: string;
  client: string;
}

export interface CoreConfig {
  path: string;
  resourceName: string;
  entryPoints?: CoreEntryPoints;
}

export interface ResourceConfig {
  include: string[];
  explicit?: ExplicitResource[];
}

export interface ExplicitResource {
  path: string;
  resourceName: string;
}

export interface ViewsConfig {
  path: string;
  enabled?: boolean;
}

export interface BuildConfig {
  minify: boolean;
  sourceMaps: boolean;
}

export interface OpencoreConfig {
  name: string;
  architecture?: 'domain-driven' | 'layer-based' | 'feature-based' | 'hybrid';
  outDir: string;
  core: CoreConfig;
  resources: ResourceConfig;
  views?: ViewsConfig;
  modules?: string[];
  build: BuildConfig;
}

/**
 * Define OpenCore configuration with TypeScript support
 * @param config - OpenCore configuration object
 * @returns The same configuration object (for type inference)
 */
export function defineConfig(config: OpencoreConfig): OpencoreConfig;

