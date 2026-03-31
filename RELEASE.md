## OpenCore CLI v1.3.0

### Views Build Simplification

- View builds are now intentionally limited to `vite` and `vanilla`
- Framework-specific CLI builders for React, Vue, Svelte, and Astro were removed
- Tailwind, PostCSS, Sass, and similar frontend tooling are no longer managed by the CLI
- Projects that need modern frontend tooling should use `framework: 'vite'` and configure their own Vite stack

### Recommended Setup

- Keep a shared root `vite.config.ts` next to `opencore.config.ts`
- Use `views.framework: 'vite'` for React, Vue, Svelte, Astro, Tailwind, PostCSS, or any advanced UI setup
- Use `views.framework: 'vanilla'` only for simple HTML/CSS/JS/TS views
- Add PostCSS only when your project needs it, such as older embedded browsers like RageMP CEF

### Breaking Change

- Existing configs using `views.framework: 'react'`, `'vue'`, `'svelte'`, or `'astro'` must be updated to `views.framework: 'vite'`
- If those legacy framework values are still used, the CLI now fails with a clear migration error
