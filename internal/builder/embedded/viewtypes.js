/*
 * WebView payload resolution via the TypeScript checker.
 *
 * The Go scanner finds `view.send(name, payload)` calls with regexes, which is enough for the name
 * but not the payload: naming the type of an arbitrary expression is the checker's job. This
 * script re-reads the same files with a real program and reports the full `UiReceives` map.
 *
 * It is a refinement, not a dependency: when it cannot run it says so and the caller keeps the
 * regex results.
 */

const fs = require('fs')
const path = require('path')

function loadTypeScript() {
    try {
        return require('typescript')
    } catch (e) {
        return null
    }
}

function findTsConfig(ts, fromFile, stopAt) {
    let dir = path.dirname(path.resolve(fromFile))
    const stop = path.resolve(stopAt || path.parse(dir).root)

    for (;;) {
        const candidate = path.join(dir, 'tsconfig.json')
        if (fs.existsSync(candidate)) return candidate
        if (dir === stop) return null
        const parent = path.dirname(dir)
        if (parent === dir) return null
        dir = parent
    }
}

function compilerOptionsFor(ts, tsconfigPath) {
    const fallback = {
        target: ts.ScriptTarget.ES2022,
        module: ts.ModuleKind.ESNext,
        moduleResolution: ts.ModuleResolutionKind.Bundler,
        experimentalDecorators: true,
        emitDecoratorMetadata: true,
        skipLibCheck: true,
        noEmit: true,
        allowJs: false,
        strict: false,
    }

    if (!tsconfigPath) return fallback

    const read = ts.readConfigFile(tsconfigPath, ts.sys.readFile)
    if (read.error) return fallback

    const parsed = ts.parseJsonConfigFileContent(
        read.config,
        ts.sys,
        path.dirname(tsconfigPath),
    )
    if (!parsed.options) return fallback

    return Object.assign({}, fallback, parsed.options, NON_NEGOTIABLE_OPTIONS)
}

const NON_NEGOTIABLE_OPTIONS = {
    noEmit: true,
    declaration: false,
    composite: false,
    incremental: false,
    skipLibCheck: true,
    strict: false,
    noUnusedLocals: false,
    noUnusedParameters: false,
}

function relativizeImports(typeText, outDir) {
    let failed = false

    const rewritten = typeText.replace(/import\("([^"]+)"\)/g, (match, spec) => {
        if (isDependencySpecifier(spec)) {
            return `import("${packageSpecifierOf(spec)}")`
        }

        if (!path.isAbsolute(spec)) return match

        let rel = path.relative(outDir, spec)
        if (!rel) {
            failed = true
            return match
        }
        rel = rel.split(path.sep).join('/')
        if (!rel.startsWith('.')) rel = './' + rel
        return `import("${stripExtension(rel)}")`
    })

    return { text: rewritten, ok: !failed }
}

function stripExtension(spec) {
    return spec.replace(/\.d\.ts$/, '').replace(/\.tsx?$/, '')
}

function isDependencySpecifier(spec) {
    return spec.includes('node_modules/')
}

function packageSpecifierOf(spec) {
    const marker = spec.lastIndexOf('node_modules/')
    return stripExtension(spec.slice(marker + 'node_modules/'.length))
}

function looksLikeWebView(ts, checker, receiver) {
    const type = checker.getTypeAtLocation(receiver)
    if (!type) return false
    if (type.flags & ts.TypeFlags.Any) return true

    const symbol = type.getSymbol() || type.aliasSymbol
    if (!symbol) return false

    if (/view/i.test(symbol.getName())) return true

    for (const declaration of symbol.getDeclarations() || []) {
        const file = declaration.getSourceFile()
        if (file && /open-core/.test(file.fileName.replace(/\\/g, '/'))) return true
    }

    return false
}

const PORTABLE_IDENTIFIERS = new Set([
    'any', 'unknown', 'never', 'void', 'null', 'undefined', 'object', 'string', 'number',
    'boolean', 'bigint', 'symbol', 'true', 'false', 'this', 'import', 'typeof', 'keyof',
    'readonly', 'infer', 'extends', 'in', 'is', 'asserts', 'new',
    'Array', 'ReadonlyArray', 'Record', 'Partial', 'Required', 'Readonly', 'Pick', 'Omit',
    'Exclude', 'Extract', 'NonNullable', 'Parameters', 'ReturnType', 'Awaited', 'Promise',
    'Map', 'ReadonlyMap', 'Set', 'ReadonlySet', 'WeakMap', 'WeakSet', 'Date', 'RegExp', 'Error',
    'Function', 'Object', 'String', 'Number', 'Boolean', 'Symbol', 'BigInt', 'JSON', 'Math',
    'Uint8Array', 'Int8Array', 'Uint16Array', 'Int16Array', 'Uint32Array', 'Int32Array',
    'Float32Array', 'Float64Array', 'ArrayBuffer', 'SharedArrayBuffer', 'DataView', 'Blob',
    'File', 'FormData', 'URL', 'URLSearchParams', 'Event', 'MessageEvent', 'CustomEvent',
    'HTMLElement', 'Element', 'Node', 'Document', 'Window', 'Iterable', 'IterableIterator',
])

function unportableNames(typeText) {
    const searchable = withoutImportsAndStringLiterals(typeText)
    const found = []

    let match
    while ((match = TYPE_REFERENCE.exec(searchable)) !== null) {
        const [, , name, , isPropertyName] = match
        if (isPropertyName === ':' || PORTABLE_IDENTIFIERS.has(name)) continue
        found.push(name)
    }

    return found
}

const TYPE_REFERENCE = /(^|[^.\w$])([A-Za-z_$][\w$]*)(\s*)(:?)/g

function withoutImportsAndStringLiterals(typeText) {
    return typeText
        .replace(/import\("[^"]*"\)/g, ' ')
        .replace(/"[^"]*"/g, '""')
        .replace(/'[^']*'/g, "''")
}

function qualifyNamedType(ts, checker, type, printed) {
    if (printed.includes('import(') || printed.includes('<')) return null

    const symbol = type.aliasSymbol || type.getSymbol()
    if (!symbol || symbol.getName() !== printed.trim()) return null

    const declarations = symbol.getDeclarations() || []
    if (declarations.length === 0) return null

    const file = declarations[0].getSourceFile()
    if (!file || file.isDeclarationFile) return null

    const fileSymbol = checker.getSymbolAtLocation(file)
    if (!fileSymbol) return null

    const exported = checker.getExportsOfModule(fileSymbol) || []
    if (!exported.some((e) => e.getName() === symbol.getName())) return null

    return `import("${file.fileName}").${symbol.getName()}`
}

function stringLiteralOf(ts, checker, node) {
    if (ts.isStringLiteralLike(node)) return node.text

    const type = checker.getTypeAtLocation(node)
    if (type && type.isStringLiteral()) return type.value

    return null
}

function toRelPosix(root, file) {
    return path.relative(root, file).split(path.sep).join('/')
}

function analyseResource(ts, resource) {
    const files = resource.files.map((f) => path.resolve(f))
    const tsconfigPath = files.length > 0 ? findTsConfig(ts, files[0], resource.projectRoot) : null
    const program = ts.createProgram(files, compilerOptionsFor(ts, tsconfigPath))

    const scan = {
        ts,
        checker: program.getTypeChecker(),
        outDir: path.resolve(resource.outDir),
        resourcePath: resource.resourcePath,
        uiReceives: [],
        warnings: [],
        seen: new Set(),
    }

    for (const file of files) {
        const source = program.getSourceFile(file)
        if (!source) {
            scan.warnings.push({
                file: toRelPosix(resource.resourcePath, file),
                line: 0,
                message: 'file could not be added to the TypeScript program; payloads not resolved',
            })
            continue
        }
        collectSendCalls(scan, source)
    }

    scan.uiReceives.sort((a, b) => (a.event < b.event ? -1 : a.event > b.event ? 1 : 0))

    return {
        resourcePath: resource.resourcePath,
        uiReceives: scan.uiReceives,
        warnings: scan.warnings,
    }
}

function collectSendCalls(scan, source) {
    const { ts } = scan
    const relFile = toRelPosix(scan.resourcePath, source.fileName)

    const visit = (node) => {
        if (isSendCall(ts, node)) {
            recordSendCall(scan, source, node, relFile)
        }
        ts.forEachChild(node, visit)
    }

    visit(source)
}

function isSendCall(ts, node) {
    return (
        ts.isCallExpression(node) &&
        ts.isPropertyAccessExpression(node.expression) &&
        node.expression.name.text === 'send' &&
        node.arguments.length > 0
    )
}

function recordSendCall(scan, source, call, relFile) {
    const { ts, checker } = scan
    const line = source.getLineAndCharacterOfPosition(call.getStart(source)).line + 1
    const eventName = stringLiteralOf(ts, checker, call.arguments[0])

    if (eventName === null) {
        if (looksLikeWebView(ts, checker, call.expression.expression)) {
            scan.warnings.push({
                file: relFile,
                line,
                message: 'could not resolve a WebView send() event name; skipped by typegen',
            })
        }
        return
    }

    if (scan.seen.has(eventName)) return
    scan.seen.add(eventName)

    scan.uiReceives.push({
        event: eventName,
        payload: payloadTypeOf(ts, checker, call, scan.outDir, relFile, line, scan.warnings),
    })
}

const TYPE_FORMAT_FLAGS = (ts) =>
    ts.TypeFormatFlags.NoTruncation |
    ts.TypeFormatFlags.UseFullyQualifiedType |
    ts.TypeFormatFlags.InTypeAlias |
    ts.TypeFormatFlags.UseStructuralFallback

function payloadTypeOf(ts, checker, call, outDir, relFile, line, warnings) {
    if (call.arguments.length < 2) return 'undefined'

    const giveUp = (message) => {
        warnings.push({ file: relFile, line, message: `${message}; using \`unknown\`` })
        return 'unknown'
    }

    const type = checker.getTypeAtLocation(call.arguments[1])
    if (!type) return giveUp('payload type unavailable')
    if (isUnresolvedType(ts, type)) {
        return giveUp('payload type could not be resolved (unresolved import?)')
    }

    const printed = printTypeQualified(ts, checker, type)
    if (!printed) return giveUp('payload type could not be printed')

    const leftover = unportableNames(printed)
    if (leftover.length > 0) {
        return giveUp(
            `payload type refers to ${quotedList(leftover)}, which the WebView cannot import` +
                ' (export the type, or move it to a shared module)',
        )
    }

    const { text, ok } = relativizeImports(printed, outDir)
    if (!ok) return giveUp('payload type refers to a module the view cannot import')

    return text
}

function isUnresolvedType(ts, type) {
    return Boolean(type.flags & ts.TypeFlags.Any)
}

function printTypeQualified(ts, checker, type) {
    let printed
    try {
        printed = checker.typeToString(type, undefined, TYPE_FORMAT_FLAGS(ts))
    } catch (e) {
        return null
    }
    if (!printed || printed === 'any' || printed === 'error') return null

    return qualifyNamedType(ts, checker, type, printed) || printed
}

function quotedList(names) {
    return names
        .slice(0, 3)
        .map((n) => '`' + n + '`')
        .join(', ')
}

function main() {
    const ts = loadTypeScript()
    if (!ts) {
        process.stdout.write(JSON.stringify({ ok: false, reason: 'typescript is not installed' }))
        return
    }

    let input
    try {
        input = JSON.parse(process.argv[2] || '{}')
    } catch (e) {
        process.stdout.write(JSON.stringify({ ok: false, reason: 'invalid input: ' + e.message }))
        return
    }

    const resources = []
    for (const resource of input.resources || []) {
        try {
            resources.push(analyseResource(ts, resource))
        } catch (e) {
            resources.push({
                resourcePath: resource.resourcePath,
                failed: true,
                reason: String((e && e.message) || e),
            })
        }
    }

    process.stdout.write(JSON.stringify({ ok: true, resources }))
}

main()
