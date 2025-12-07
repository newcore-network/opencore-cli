const esbuild = require('esbuild')
const { swcPlugin } = require('esbuild-plugin-swc')

const swcConfig = swcPlugin({
    jsc: {
        parser: {
            syntax: 'typescript',
            decorators: true,
            dynamicImport: true,
        },
        transform: {
            legacyDecorator: true,
            decoratorMetadata: true,
        },
        keepClassNames: true,
    },
})

const sharedConfig = {
    bundle: true,
    sourcemap: 'inline',
    minifyWhitespace: true,
    minifySyntax: true,
    minifyIdentifiers: false,
    keepNames: true,
    plugins: [swcConfig],
    logLevel: 'info',
}

async function build() {
    try {
        const serverBuild = esbuild.build({
            ...sharedConfig,
            entryPoints: ['src/server.ts'],
            outfile: 'dist/server.js',
            platform: 'node',
            target: 'node16',
            format: 'cjs',
        })

        const clientBuild = esbuild.build({
            ...sharedConfig,
            entryPoints: ['src/client.ts'],
            outfile: 'dist/client.js',
            platform: 'browser',
            target: 'es2020',
            format: 'iife',
        })

        await Promise.all([serverBuild, clientBuild])

        console.log('Build completado: dist/server.js y dist/client.js creados.')
    } catch (error) {
        console.error('Error en el build:', error)
        process.exit(1)
    }
}

build()
