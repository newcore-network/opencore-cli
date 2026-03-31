import type { UserConfig } from 'vite';

export interface OpenCoreViteOptions {
  /**
   * Override the resolved view root. Defaults to OPENCORE_VIEW_ROOT or process.cwd().
   */
  root?: string;

  /**
   * Override the resolved OpenCore project root.
   * By default this is auto-detected by walking upward until opencore.config.* is found.
   */
  projectRoot?: string;

  /**
   * Override the output directory. Defaults to OPENCORE_VIEW_OUTDIR or <viewRoot>/dist.
   */
  outDir?: string;

  /**
   * Override the default build target.
   * @default 'es2020'
   */
  target?: string;

  /**
   * Control PostCSS config resolution.
   * - undefined: auto-detect postcss.config.* from the OpenCore project root
   * - false: disable auto PostCSS wiring
   * - string: explicit path relative to the project root or absolute path
   */
  postcss?: false | string;
}

export type OpenCoreViteConfig = UserConfig & {
  opencore?: OpenCoreViteOptions;
};

export function createOpenCoreViteConfig(config?: OpenCoreViteConfig): UserConfig;

export default createOpenCoreViteConfig;
