# OpenCore CLI - Guía Completa en Español

## 📋 Tabla de Contenidos

- [Instalación](#instalación)
- [Inicio Rápido](#inicio-rápido)
- [Comandos](#comandos)
- [Configuración](#configuración)
- [Desarrollo](#desarrollo)
- [Publicación](#publicación)

## 🚀 Instalación

### Via NPM (Recomendado)

```bash
npm install -g @open-core/cli
# o
pnpm add -g @open-core/cli
```

### Via Go

```bash
go install github.com/newcore-network/opencore-cli/cmd/opencore@latest
```

### Desde Código Fuente

```bash
git clone https://github.com/newcore-network/opencore-cli
cd opencore-cli
go build -o opencore ./cmd/opencore
```

## 🏗️ Arquitecturas de Proyecto

OpenCore CLI soporta **4 arquitecturas** para adaptarse al tamaño de tu equipo y complejidad del proyecto:

### 1. Domain-Driven (Recomendada para proyectos grandes)

Organiza el código por dominios de negocio con separación completa client/server/shared:

```
core/src/modules/
├── banking/
│   ├── client/
│   │   ├── banking.controller.ts
│   │   └── banking.ui.ts
│   ├── server/
│   │   ├── banking.controller.ts
│   │   ├── banking.service.ts
│   │   └── banking.repository.ts
│   └── shared/
│       ├── banking.types.ts
│       └── banking.events.ts
```

**Ideal para**: Proyectos grandes, múltiples dominios de negocio, alta cohesión

### 2. Layer-Based (Para equipos grandes)

Arquitectura clásica por capas con separación técnica:

```
core/src/
├── client/
│   ├── controllers/
│   └── services/
├── server/
│   ├── controllers/
│   └── services/
└── shared/
```

**Ideal para**: Equipos grandes con roles especializados (frontend/backend)

### 3. Feature-Based (Simple y rápida)

Estructura ligera para iteración rápida:

```
core/src/features/
├── banking/
│   ├── banking.controller.ts
│   ├── banking.service.ts
│   └── index.ts
```

**Ideal para**: Proyectos pequeños/medianos, features independientes

### 4. Hybrid (Flexible)

Mezcla módulos críticos (domain-driven) con features simples:

```
core/src/
├── core-modules/  # Sistemas críticos
│   └── identity/
├── features/      # Features simples
│   └── notifications/
```

**Ideal para**: Proyectos evolutivos, complejidad mixta

> ¡Elige tu arquitectura durante `opencore init` - los comandos se adaptan automáticamente!

## ⚡ Inicio Rápido

### 1. Crear un Nuevo Proyecto

```bash
opencore init mi-servidor
# Selecciona la arquitectura que prefieras
cd mi-servidor
pnpm install
```

### 2. Crear una Feature

```bash
opencore create feature banking
```

El CLI detecta tu arquitectura y crea los archivos apropiados automáticamente.

### 3. Crear un Resource

```bash
opencore create resource chat --with-client
```

### 4. Modo Desarrollo

```bash
opencore dev
```

### 5. Build para Producción

```bash
opencore build
```

## 📚 Comandos

### `opencore init [nombre]`

Inicializa un nuevo proyecto OpenCore con la estructura completa.

**Opciones interactivas:**
- Nombre del proyecto
- Instalar @open-core/identity
- Habilitar minificación

### `opencore create feature [nombre]`

Crea una nueva feature en `core/src/features/`.

**Ejemplo:**
```bash
opencore create feature jobs
```

### `opencore create resource [nombre]`

Crea un resource independiente en `resources/`.

**Flags:**
- `--with-client` - Incluir código cliente
- `--with-nui` - Incluir UI (NUI)

**Ejemplo:**
```bash
opencore create resource admin --with-client --with-nui
```

### `opencore build`

Compila todos los resources a JavaScript.

**Características:**
- UI animada con progreso
- Timing de cada resource
- Reporte de errores detallado

### `opencore dev`

Inicia modo desarrollo con hot-reload.

**Características:**
- Watch de archivos automático
- Rebuild al detectar cambios
- Debouncing de 300ms

### `opencore doctor`

Valida la configuración del proyecto.

**Verifica:**
- Node.js instalado
- pnpm instalado
- Estructura de proyecto válida
- Dependencias instaladas

### `opencore clone [template]`

Clona un template oficial desde GitHub.

**Templates disponibles:**
- `chat` - Sistema de chat completo
- `admin` - Panel de administración
- `racing` - Sistema de carreras

## ⚙️ Configuración

El archivo `opencore.config.ts` controla el comportamiento del CLI:

```typescript
import { defineConfig } from '@open-core/cli'

export default defineConfig({
  // Nombre del proyecto
  name: 'mi-servidor',
  
  // Arquitectura del proyecto
  architecture: 'domain-driven',  // o 'layer-based', 'feature-based', 'hybrid'
  
  // Directorio de salida
  outDir: './dist/resources',
  
  // Configuración del core
  core: {
    path: './core',
    resourceName: '[core]',
  },
  
  // Resources adicionales
  resources: {
    include: ['./resources/*'],
  },
  
  // Módulos oficiales a instalar
  modules: ['@open-core/identity'],
  
  // Opciones de build
  build: {
    minify: true,
    sourceMaps: true,
  }
})
```

## 🛠️ Desarrollo del CLI

### Estructura del Proyecto

```
opencore-cli/
├── cmd/opencore/           # Entry point
├── internal/
│   ├── commands/           # Implementación de comandos
│   ├── config/             # Loader de configuración
│   ├── builder/            # Sistema de build
│   ├── watcher/            # File watcher
│   ├── templates/          # Templates embebidos
│   └── ui/                 # Estilos visuales
├── npm/                    # NPM wrapper
└── .github/workflows/      # CI/CD
```

### Compilar Localmente

```bash
# Instalar dependencias
go mod download

# Compilar
go build -o opencore ./cmd/opencore

# Ejecutar
./opencore --version
```

### Compilar para Todas las Plataformas

```bash
make build-all
```

Genera binarios en `build/`:
- `opencore-windows-amd64.exe`
- `opencore-darwin-amd64`
- `opencore-darwin-arm64`
- `opencore-linux-amd64`

## 📦 Publicación

### Requisitos

- Cuenta en GitHub
- Cuenta en npmjs.com
- Token de NPM con permisos de publicación

### Pasos

1. **Crear Repositorio en GitHub**
   ```bash
   git remote add origin https://github.com/newcore-network/opencore-cli.git
   git branch -M main
   git push -u origin main
   ```

2. **Configurar NPM Token**
   - Ir a Settings → Secrets → Actions
   - Agregar `NPM_TOKEN` con tu token de npmjs.com

3. **Crear Release**
   ```bash
   git tag -a v0.1.0 -m "Initial release"
   git push origin v0.1.0
   ```

4. **GitHub Actions se encarga del resto:**
   - Compila binarios para todas las plataformas
   - Crea GitHub Release
   - Publica a NPM

## 🎯 Tips y Mejores Prácticas

### Organización de Código

1. **Features** - Para lógica de gameplay (banking, jobs, housing)
2. **Resources** - Para sistemas standalone (chat, admin, utilities)
3. **Core lean** - Mantén el core ligero, mueve sistemas complejos a resources

### Desarrollo

1. Usa `opencore dev` durante desarrollo
2. Ejecuta `opencore doctor` si algo no funciona
3. Revisa `opencore.config.ts` para personalizar el build

### Producción

1. Siempre ejecuta `opencore build` antes de desplegar
2. Habilita minificación en producción
3. Mantén sourceMaps para debugging

## ❓ Troubleshooting

### `opencore: command not found`

**Solución:**
- Instala globalmente con `-g`
- O usa `npx @open-core/cli`

### Build falla

**Solución:**
- Ejecuta `opencore doctor`
- Verifica que Node.js y pnpm estén instalados
- Revisa que las dependencias estén instaladas

### Errores de TypeScript

**Solución:**
- Verifica que `@open-core/framework` esté instalado
- Ejecuta `pnpm install` en la raíz del proyecto

## 🔗 Enlaces Útiles

- [OpenCore Framework](https://github.com/newcore-network/opencore)
- [OpenCore Identity](https://github.com/newcore-network/opencore-identity)
- [NPM Package](https://www.npmjs.com/package/@open-core/cli)
- [GitHub Releases](https://github.com/newcore-network/opencore-cli/releases)

---

**¿Preguntas?** Abre un issue en GitHub o consulta la documentación completa en inglés.


