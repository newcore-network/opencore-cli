## OpenCore CLI v1.3.0

### Summary

- Views now use a single shared Vite configuration by default, with a minimal `vanilla` fallback for simple HTML/JS/TS UIs.
- The CLI now exposes a public `createOpenCoreViteConfig` helper through `@open-core/cli/vite` so project-level Vite configs stay small and consistent.
- PostCSS is auto-resolved from the OpenCore project root when present, which keeps Tailwind and older Chromium targets like RageMP CEF working without extra wiring.
- Legacy framework-specific CLI view builders were removed. React, Vue, Svelte, and Astro should now be configured in Vite, not in the CLI.
- Build path normalization and explicit-resource resolution were fixed so duplicated resource builds and stale overrides no longer happen when paths are written with or without `./`.

### Recommended Setup

- Keep a shared root Vite config next to `opencore.config.ts`.
- Use `@open-core/cli/vite` to keep that config small.
- Use `views.framework: 'vite'` for React, Vue, Svelte, Astro, Tailwind, PostCSS, Sass, or any advanced UI setup.
- Use `views.framework: 'vanilla'` only for simple HTML/CSS/JS/TS views.
- Add PostCSS only when your project needs it, such as older embedded browsers like RageMP CEF.

### Breaking Change

- Existing configs using `views.framework: 'react'`, `'vue'`, `'svelte'`, or `'astro'` must be updated to `views.framework: 'vite'`.
- If those legacy framework values are still used, the CLI now fails with a clear migration error.
