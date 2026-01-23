# FiveM Runtime Environments

FiveM has **three distinct runtime environments**. Understanding their differences is essential for building compatible resources.

## Overview

| Feature | Server | Client | Views (NUI) |
|---------|--------|--------|-------------|
| Runtime | Node.js | Neutral JS | Web Browser |
| Platform | `node` | `neutral` | `browser` |
| Node.js APIs | Available | NOT available | NOT available |
| Web APIs | NOT available | NOT available | Available |
| FiveM APIs | Available | Available | Limited |
| GTA Natives | NOT available | Available | NOT available |
| External packages | Supported | NOT supported | N/A (separate build) |

---

## Server Runtime

The server runs in a **full Node.js environment**.

### Available

- All Node.js APIs (`fs`, `path`, `http`, `crypto`, `child_process`, etc.)
- FiveM server-side APIs and events
- External packages from `node_modules`
- Filesystem access
- Network requests

### NOT Available

- GTA natives (server has no game context)
- Web APIs (`DOM`, `window`, etc.)

### Build Configuration

```typescript
server: {
  platform: 'node',
  format: 'cjs',
  target: 'es2020',
  external: ['typeorm', 'pg'],  // Optional
}
```

---

## Client Runtime

The client runs in a **neutral JavaScript environment** inside the game.

### Available

- FiveM client-side APIs and events
- GTA V natives (game functions)
- Pure JavaScript/ES2020

### NOT Available

- Node.js APIs (`fs`, `path`, `http`, etc.)
- Web APIs (`DOM`, `fetch`, `localStorage`, `window`, etc.)
- Filesystem access
- External packages (everything must be bundled)

### Build Configuration

```typescript
client: {
  platform: 'neutral',
  format: 'iife',
  target: 'es2020',
  // external: NOT supported - all deps must be bundled
}
```

If you configure `client.external`, it will be ignored with a warning.

---

## Views Runtime (NUI)

Views/NUI run in an **embedded web browser** (Chromium-based).

### Available

- Web APIs (`DOM`, `fetch`, `localStorage`, `window`, etc.)
- CSS, HTML, JavaScript
- Web frameworks (React, Vue, Svelte, etc.)
- Communication with client via `SendNUIMessage` / `RegisterNUICallback`

### NOT Available

- Node.js APIs
- FiveM APIs (must communicate via NUI callbacks)
- GTA natives

### Limitations

The embedded browser has **version limitations** that are not fully documented. Some modern Web APIs may not be available. Test your NUI on actual FiveM to ensure compatibility.

Known considerations:
- Older Chromium version than current Chrome
- Some CSS features may not work
- Some modern JS APIs may be missing

### Build Configuration

Views are built separately using web bundlers:

```typescript
views: {
  path: './core/views',
  framework: 'react',  // or 'vue', 'svelte', 'solid', 'vanilla'
}
```

---

## Communication Between Environments

```
┌─────────────────────────────────────────────────────────────┐
│                        SERVER                                │
│                      (Node.js)                               │
│                                                              │
│  - Database access                                           │
│  - Business logic                                            │
│  - External APIs                                             │
└──────────────────────┬──────────────────────────────────────┘
                       │
                    emitNet
                       │
┌──────────────────────▼──────────────────────────────────────┐
│                        CLIENT                                │
│                    (Neutral JS)                              │
│                                                              │
│  - GTA natives                                               │
│  - Game interaction                                          │
│  - Player input                                              │
└──────────────────────┬──────────────────────────────────────┘
                       │
            SendNUIMessage / RegisterNUICallback
                       │
┌──────────────────────▼──────────────────────────────────────┐
│                      VIEWS (NUI)                             │
│                    (Web Browser)                             │
│                                                              │
│  - User interface                                            │
│  - HTML/CSS/JS                                               │
│  - Web frameworks                                            │
└─────────────────────────────────────────────────────────────┘
```

---

## Incompatible Packages (Client)

These packages use C++ bindings and will NOT work on the client:

| Package | Alternative |
|---------|-------------|
| `bcrypt` | `bcryptjs` |
| `argon2` | `hash.js`, `js-sha3` |
| `sharp` | `jimp` |
| `canvas` | `pureimage` |
| `sqlite3` | `sql.js` |
| `better-sqlite3` | `sql.js` |

The CLI will warn you if it detects incompatible packages. (nothing is promised)

---

## Best Practices

1. **Server**: Use for heavy computation, database access, external APIs, source of truth
2. **Client**: Keep minimal - only game interaction and natives
3. **Views**: Standard web development, but test on FiveM
4. **Bundle client deps**: NEVER use `external` for client
5. **Test on actual FiveM**: Some packages may have hidden dependencies