# OpenCore CLI - Quick Start Guide

Get up and running with OpenCore in 5 minutes!

## Installation

### Via NPM (Recommended)

```bash
npm install -g @open-core/cli
# or
pnpm add -g @open-core/cli
```

### Via Go

```bash
go install github.com/newcore-network/opencore-cli@latest
```

## Create Your First Project

```bash
# Create a new project
opencore init my-fivem-server

# Navigate to project
cd my-fivem-server

# Install dependencies
pnpm install
```

This creates a complete OpenCore project with:
- **Core resource** with feature-based architecture
- **TypeScript setup** with proper configuration
- **Build system** ready to go
- **Package management** with pnpm workspaces

## Project Structure

```
my-fivem-server/
├── core/                      # Core resource (framework)
│   ├── src/
│   │   ├── features/         # Feature modules go here
│   │   ├── client/           # Client-side entry
│   │   └── server/           # Server-side entry
│   ├── fxmanifest.lua
│   ├── package.json
│   └── tsconfig.json
├── resources/                 # Additional resources
├── opencore.config.ts        # CLI configuration
├── package.json
└── pnpm-workspace.yaml
```

## Create Your First Feature

Features are self-contained modules in your core resource:

```bash
opencore create feature banking
```

This creates:
```
core/src/features/banking/
├── banking.controller.ts     # Handles events/commands
├── banking.service.ts        # Business logic
└── index.ts                  # Feature entry point
```

### Example Feature Code

**banking.service.ts:**
```typescript
export class BankingService {
  private accounts = new Map<number, number>();

  getBalance(playerId: number): number {
    return this.accounts.get(playerId) || 0;
  }

  deposit(playerId: number, amount: number): void {
    const current = this.getBalance(playerId);
    this.accounts.set(playerId, current + amount);
  }
}
```

**banking.controller.ts:**
```typescript
import { BankingService } from './banking.service';

export class BankingController {
  constructor(private banking: BankingService) {
    this.registerCommands();
  }

  private registerCommands() {
    RegisterCommand('balance', (source: number) => {
      const balance = this.banking.getBalance(source);
      console.log(`Balance: $${balance}`);
    }, false);
  }
}
```

## Create an Independent Resource

For standalone systems, create independent resources:

```bash
# Server-only resource
opencore create resource admin

# Resource with client code
opencore create resource hud --with-client

# Resource with UI
opencore create resource phone --with-client --with-nui
```

This creates a complete resource in `resources/[name]/`:
```
resources/admin/
├── src/
│   ├── server/
│   │   └── main.ts
│   └── client/              # If --with-client
│       └── main.ts
├── fxmanifest.lua
├── package.json
└── tsconfig.json
```

## Development Mode

Start development mode with hot-reload:

```bash
opencore dev
```

This will:
- ✅ Watch for file changes
- ✅ Automatically rebuild on save
- ✅ Show build status in real-time
- ✅ Catch TypeScript errors

Just edit your `.ts` files and the CLI handles the rest!

## Build for Production

When ready to deploy:

```bash
opencore build
```

This compiles all resources to JavaScript in `dist/resources/`:
```
dist/resources/
├── [core]/                   # Your core resource
│   ├── fxmanifest.lua
│   └── dist/                # Compiled JS
└── [admin]/                 # Additional resources
    ├── fxmanifest.lua
    └── dist/
```

Copy `dist/resources/` to your FiveM server's resources folder!

## Validate Your Setup

Check if everything is configured correctly:

```bash
opencore doctor
```

This verifies:
- ✅ Node.js installed
- ✅ pnpm installed
- ✅ Project structure valid
- ✅ Dependencies installed
- ✅ TypeScript configuration correct

## Configuration

Customize the CLI behavior in `opencore.config.ts`:

```typescript
import { defineConfig } from '@open-core/cli'

export default defineConfig({
  name: 'my-server',
  outDir: './dist/resources',
  
  core: {
    path: './core',
    resourceName: '[core]',
  },
  
  resources: {
    include: ['./resources/*'],
  },
  
  modules: ['@open-core/identity'],
  
  build: {
    minify: true,        // Minify for production
    sourceMaps: true,    // Generate source maps
  }
})
```

## Using Official Modules

Install official OpenCore modules:

```bash
# Add to your project
pnpm add @open-core/identity

# Update config
echo "modules: ['@open-core/identity']" >> opencore.config.ts
```

Official modules:
- `@open-core/identity` - Player identity & authentication
- `@open-core/inventory` - Item management
- `@open-core/vehicles` - Vehicle system

## Common Commands

| Command | Description |
|---------|-------------|
| `opencore init [name]` | Create new project |
| `opencore create feature [name]` | Create feature in core |
| `opencore create resource [name]` | Create independent resource |
| `opencore dev` | Start development mode |
| `opencore build` | Build for production |
| `opencore doctor` | Validate setup |
| `opencore clone [template]` | Clone official template |
| `opencore --version` | Show CLI version |

## Next Steps

1. **Learn the Architecture**: Read [ARCHITECTURE.md](ARCHITECTURE.md)
2. **Explore Templates**: Clone official templates with `opencore clone`
3. **Join Community**: Share your projects and get help
4. **Read Docs**: Check out the [full documentation](README.md)

## Tips & Best Practices

### 🎯 Feature vs Resource

- **Use features** for core gameplay (jobs, banking, housing)
- **Use resources** for standalone systems (admin, chat, HUD)

### ⚡ Development Workflow

1. Run `opencore dev` in one terminal
2. Edit TypeScript files
3. Test in FiveM (restart resource if needed)
4. Repeat!

### 🏗️ Building

- Development: Fast builds, no minification
- Production: `opencore build` with minification enabled

### 🔍 Debugging

- Enable `sourceMaps: true` in config
- Use TypeScript's type checking
- Check build output for errors

## Troubleshooting

### Command not found

```bash
npm install -g @open-core/cli
# or
npx @open-core/cli --version
```

### Build fails

```bash
opencore doctor
pnpm install
```

### TypeScript errors

Make sure `@open-core/framework` is installed:
```bash
pnpm add @open-core/framework
```

## Getting Help

- 📖 [Documentation](README.md)
- 🐛 [Report Issues](https://github.com/newcore-network/opencore-cli/issues)
- 💬 [Discussions](https://github.com/newcore-network/opencore-cli/discussions)

---

**Ready to build something amazing?** 🚀

```bash
opencore init my-project && cd my-project && pnpm install && opencore dev
```

