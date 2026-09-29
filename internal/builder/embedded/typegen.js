// Typegen analyzer: reads a request on stdin, writes the listeners found through the TypeScript AST on stdout.
'use strict'

const fs = require('fs')
const path = require('path')
const { pathToFileURL } = require('url')

const FRAMEWORK = '@open-core/framework'
const DECORATORS = {
    server: { OnNet: 'serverEvents', OnRPC: 'serverRpc', Command: 'commands' },
    client: { OnNet: 'clientEvents', OnRPC: 'clientRpc', OnView: 'view' },
}
const WEBVIEW_CLASSES = ['WebViewBridge', 'NuiBridge']
const WEBVIEW_GLOBALS = ['WebView', 'NUI']
const IMPORT_MARK = '\u0001'

async function loadHost(tsDir) {
    const pkg = JSON.parse(fs.readFileSync(path.join(tsDir, 'package.json'), 'utf8'))
    if (parseInt(pkg.version, 10) >= 7) return syncApiHost(tsDir, pkg)
    return compilerApiHost(require(tsDir))
}

function compilerApiHost(ts) {
    return {
        SK: ts.SyntaxKind,
        TF: ts.TypeFlags,
        SF: ts.SymbolFlags,
        OF: ts.ObjectFlags,
        CALL: ts.SignatureKind.Call,
        open(configPath) {
            const config = ts.readConfigFile(configPath, ts.sys.readFile).config
            const parsed = ts.parseJsonConfigFileContent(config, ts.sys, path.dirname(configPath), undefined, configPath)
            const program = ts.createProgram(parsed.fileNames, parsed.options)
            return {
                checker: program.getTypeChecker(),
                getSourceFile: (file) => program.getSourceFile(toSlash(file)),
                fileNames: () => program.getSourceFiles().map((sf) => sf.fileName),
                close() {},
            }
        },
        forEachChild: (node, cb) => ts.forEachChild(node, cb),
        declarations: (symbol) => (symbol.declarations || []).map((node) => ({ file: node.getSourceFile().fileName, node: () => node })),
        typeSymbol: (type) => type.symbol,
        aliasSymbol: (type) => type.aliasSymbol,
        aliasTypeArguments: (type) => type.aliasTypeArguments || [],
        members: (type) => type.types || [],
        typeParameterCount: (type) => ((type.target || type).typeParameters || []).length,
        objectTypeOf: (type) => type.objectType,
    }
}

async function syncApiHost(tsDir, pkg) {
    const load = (subpath) => import(pathToFileURL(path.join(tsDir, pkg.exports[subpath])).href)
    const api = await load('./unstable/sync')
    const ast = await load('./unstable/ast')
    return {
        SK: ast.SyntaxKind,
        TF: api.TypeFlags,
        SF: api.SymbolFlags,
        OF: api.ObjectFlags,
        CALL: api.SignatureKind.Call,
        open(configPath) {
            const client = new api.API({ cwd: path.dirname(configPath) })
            const snapshot = client.updateSnapshot({ openProject: configPath })
            const project = snapshot.getProject(configPath) || snapshot.getProjects()[0]
            return {
                checker: project.checker,
                getSourceFile: (file) => project.program.getSourceFile(toSlash(file)),
                fileNames: () => project.program.getSourceFileNames(),
                close: () => client.close(),
            }
        },
        forEachChild: (node, cb) => node.forEachChild(cb),
        declarations: (symbol) => (symbol.declarations || []).map((handle) => ({ file: handle.path, node: () => handle.resolve() })),
        typeSymbol: (type) => type.getSymbol(),
        aliasSymbol: (type) => type.getAliasSymbol(),
        aliasTypeArguments: (type) => type.getAliasTypeArguments() || [],
        members: (type) => type.getTypes() || [],
        typeParameterCount: (type) => {
            const target = type.getTarget ? type.getTarget() : type
            return target.getTypeParameters ? target.getTypeParameters().length : 0
        },
        objectTypeOf: (type) => type.getObjectType(),
    }
}

function toSlash(file) {
    return file.split(path.sep).join('/')
}

const packageCache = new Map()

function owningPackage(dir) {
    if (!packageCache.has(dir)) {
        const manifest = path.join(dir, 'package.json')
        const parent = path.dirname(dir)
        let owner = parent === dir ? null : undefined
        if (fs.existsSync(manifest)) owner = { name: JSON.parse(fs.readFileSync(manifest, 'utf8')).name, root: dir }
        packageCache.set(dir, owner === undefined ? owningPackage(parent) : owner)
    }
    return packageCache.get(dir)
}

// 'server' / 'client' for a file of the framework package, '' for its shared code, null otherwise.
function frameworkSideOf(file) {
    const owner = owningPackage(path.dirname(path.resolve(file)))
    if (!owner || owner.name !== FRAMEWORK) return null
    const segments = toSlash(path.relative(owner.root, path.resolve(file))).split('/')
    return segments.find((s) => s === 'server' || s === 'client') || ''
}

function specifierSide(specifier) {
    return { [FRAMEWORK]: '', [`${FRAMEWORK}/server`]: 'server', [`${FRAMEWORK}/client`]: 'client' }[specifier] ?? null
}

function namespaceSide(name) {
    return { Server: 'server', Client: 'client' }[name] ?? null
}

class Analyzer {
    constructor(host, program) {
        this.h = host
        this.SK = host.SK
        this.program = program
        this.checker = program.checker
    }

    analyzeFile(file) {
        this.result = { handlers: [], views: [], sends: [], warnings: [] }
        this.sf = this.program.getSourceFile(file)
        if (this.sf && /@|send/.test(this.sf.text)) this.visit(this.sf, null)
        return this.result
    }

    visit(node, classNode) {
        const SK = this.SK
        if (node.kind === SK.ClassDeclaration || node.kind === SK.ClassExpression) {
            classNode = node
            for (const member of node.members) this.visitMember(member, node)
        } else if (node.kind === SK.CallExpression) {
            this.visitCall(node, classNode)
        }
        this.h.forEachChild(node, (child) => {
            this.visit(child, classNode)
        })
    }

    visitMember(member, classNode) {
        const SK = this.SK
        const methodName = this.memberName(member)
        if (member.kind !== SK.MethodDeclaration || methodName === null || !classNode.name) return

        for (const decorator of member.modifiers || []) {
            const call = decorator.expression
            if (decorator.kind !== SK.Decorator || call.kind !== SK.CallExpression) continue
            const match = this.classifyDecorator(call.expression)
            const kind = match && DECORATORS[match.side] && DECORATORS[match.side][match.name]
            if (!kind) continue

            const line = this.lineOf(decorator)
            const name = kind === 'commands' ? this.commandName(call.arguments[0]) : this.eventName(call.arguments[0])
            if (!name) continue
            const entry = { event: name.value, className: classNode.name.text, methodName, line }
            if (kind === 'view') this.result.views.push(entry)
            else this.result.handlers.push({ kind, description: name.description || '', usage: name.usage || '', ...entry })
        }
    }

    memberName(member) {
        const SK = this.SK
        const name = member.name
        return name && [SK.Identifier, SK.StringLiteral, SK.NumericLiteral].includes(name.kind) ? name.text : null
    }

    // Returns { side, name } for a framework decorator, whatever the import style.
    classifyDecorator(callee) {
        const SK = this.SK
        const nameNode = callee.kind === SK.PropertyAccessExpression ? callee.name : callee
        if (nameNode.kind !== SK.Identifier) return null
        const symbol = this.checker.getSymbolAtLocation(nameNode)
        const bySymbol = symbol && this.classifySymbol(symbol)
        if (bySymbol !== undefined) return bySymbol
        if (callee.kind === SK.PropertyAccessExpression && callee.expression.kind === SK.Identifier) {
            return this.classifyNamespaceMember(callee.expression, callee.name.text)
        }
        return null
    }

    // Follows the alias chain; undefined when the target could not be resolved.
    classifySymbol(symbol) {
        const SF = this.h.SF
        let current = symbol
        while (current) {
            const imported = this.frameworkImportOf(current)
            if (imported) return imported.side ? imported : null
            if (!(current.flags & SF.Alias)) break
            const next = this.checker.getImmediateAliasedSymbol(current)
            current = next && next !== current && !this.isUnknown(next) ? next : null
        }
        if (!current) return undefined
        for (const decl of this.h.declarations(current)) {
            const side = frameworkSideOf(decl.file)
            if (side === null) return null
            if (side) return { side, name: current.name }
        }
        return undefined
    }

    frameworkImportOf(symbol) {
        const SK = this.SK
        for (const decl of this.h.declarations(symbol)) {
            const node = decl.node()
            let specifier = null
            if (node.kind === SK.ImportSpecifier) specifier = node.parent.parent.parent.moduleSpecifier
            else if (node.kind === SK.ExportSpecifier) specifier = node.parent.parent.moduleSpecifier
            const side = specifier && specifier.kind === SK.StringLiteral ? specifierSide(specifier.text) : null
            if (side !== null) return { side, name: (node.propertyName || node.name).text }
        }
        return null
    }

    // `NS.Member` when NS is an unresolved framework import or an undeclared `Server` / `Client`.
    classifyNamespaceMember(namespaceId, member) {
        const SK = this.SK
        const symbol = this.checker.getSymbolAtLocation(namespaceId)
        if (!symbol) {
            const side = namespaceSide(namespaceId.text)
            return side && { side, name: member }
        }
        for (const decl of this.h.declarations(symbol)) {
            const node = decl.node()
            if (node.kind === SK.NamespaceImport) {
                const side = specifierSide(node.parent.parent.moduleSpecifier.text)
                if (side) return { side, name: member }
            } else if (node.kind === SK.ImportSpecifier && specifierSide(node.parent.parent.parent.moduleSpecifier.text) !== null) {
                const side = namespaceSide((node.propertyName || node.name).text)
                if (side) return { side, name: member }
            }
        }
        return null
    }

    isUnknown(symbol) {
        return this.checker.isUnknownSymbol ? this.checker.isUnknownSymbol(symbol) : !symbol.declarations && symbol.name === 'unknown'
    }

    eventName(arg) {
        const literals = arg && this.stringLiterals(this.checker.getTypeAtLocation(arg))
        if (literals && literals.length === 1) return { value: literals[0] }
        return this.unresolved(arg)
    }

    commandName(arg) {
        if (!arg) return null
        const type = this.checker.getTypeAtLocation(arg)
        const literals = this.stringLiterals(type)
        if (literals && literals.length === 1) return { value: literals[0] }
        const command = this.propertyLiteral(arg, type, 'command')
        if (!command) return this.unresolved(arg)
        return {
            value: command,
            description: this.propertyLiteral(arg, type, 'description'),
            usage: this.propertyLiteral(arg, type, 'usage'),
        }
    }

    unresolved(arg) {
        if (arg) {
            const type = this.checker.getTypeAtLocation(arg)
            this.warn(this.lineOf(arg), `could not resolve the name ${this.sourceOf(arg)}: its type is \`${this.checker.typeToString(type)}\`, not a string literal; skipped by typegen`)
        }
        return null
    }

    // An inline object literal is widened by the decorator's parameter type, so its initializers are read.
    propertyLiteral(arg, type, key) {
        let valueType = null
        if (arg.kind === this.SK.ObjectLiteralExpression) {
            const prop = arg.properties.find((p) => p.initializer && this.memberName(p) === key)
            valueType = prop && this.checker.getTypeAtLocation(prop.initializer)
        } else {
            const prop = this.checker.getPropertyOfType(type, key)
            valueType = prop && this.checker.getTypeOfSymbolAtLocation(prop, arg)
        }
        const literals = valueType && this.stringLiterals(valueType)
        return literals && literals.length === 1 ? literals[0] : ''
    }

    stringLiterals(type) {
        const TF = this.h.TF
        if (type.flags & TF.StringLiteral) return [String(type.value)]
        if (type.flags & TF.Union) {
            const members = this.h.members(type)
            return members.every((m) => m.flags & TF.StringLiteral) ? members.map((m) => String(m.value)) : null
        }
        if (type.flags & TF.TypeParameter) {
            const constraint = this.checker.getBaseConstraintOfType(type)
            return constraint && constraint !== type ? this.stringLiterals(constraint) : null
        }
        return null
    }

    visitCall(call, classNode) {
        const callee = call.expression
        if (callee.kind !== this.SK.PropertyAccessExpression || callee.name.text !== 'send' || !this.isWebViewSend(callee)) return

        const line = this.lineOf(call)
        const [nameArg, payloadArg] = call.arguments
        const nameType = nameArg && this.checker.getTypeAtLocation(nameArg)
        const literals = nameType && this.stringLiterals(nameType)
        if (!literals || literals.length === 0) {
            if (nameArg) this.warn(line, `could not resolve a WebView send() event name (${this.sourceOf(nameArg)}); skipped by typegen`)
            return
        }

        // A typed forwarder, `send<K extends keyof M>(name: K, data: M[K])`, sends one message per key.
        const forwarder = literals.length > 1 && nameType.flags & this.h.TF.TypeParameter
        if (forwarder && this.isFrameworkMapped(nameType)) return
        for (const event of literals) {
            const payload = forwarder ? this.keyedPayload(payloadArg, event) : this.sendPayload(payloadArg, call, classNode)
            if (payload.kind === 'type' && payload.type === 'unknown') {
                this.warn(line, `could not derive the payload type of ${JSON.stringify(event)}; the WebView receives \`unknown\``)
            }
            this.result.sends.push({ event, line, payload })
        }
    }

    isWebViewSend(callee) {
        const symbol = this.checker.getSymbolAtLocation(callee.name)
        const decls = symbol ? this.h.declarations(symbol) : []
        if (decls.length > 0) {
            return decls.some((decl) => {
                const owner = frameworkSideOf(decl.file) !== null && decl.node().parent
                return Boolean(owner && owner.name && WEBVIEW_CLASSES.includes(owner.name.text))
            })
        }
        const receiver = callee.expression
        if (receiver.kind !== this.SK.Identifier || !WEBVIEW_GLOBALS.includes(receiver.text)) return false
        const receiverSymbol = this.checker.getSymbolAtLocation(receiver)
        if (!receiverSymbol) return true
        const imported = this.frameworkImportOf(receiverSymbol)
        return Boolean(imported && imported.side !== 'server')
    }

    // A forwarder keyed by the framework's own registered map would feed typegen its own output.
    isFrameworkMapped(nameType) {
        const symbol = this.h.typeSymbol(nameType)
        const decl = symbol && this.h.declarations(symbol)[0]
        const constraint = decl && decl.node().constraint
        return Boolean(constraint) && /\b(ViewSend|ViewReceive|NameOf)\b/.test(this.sourceOf(constraint))
    }

    keyedPayload(payloadArg, event) {
        if (!payloadArg) return this.payload(null)
        const symbol = this.checker.getSymbolAtLocation(payloadArg)
        const declared = symbol && this.checker.getTypeOfSymbol(symbol)
        if (declared && declared.flags & this.h.TF.IndexedAccess) {
            const prop = this.checker.getPropertyOfType(this.h.objectTypeOf(declared), event)
            if (prop) return this.payload(this.checker.getTypeOfSymbolAtLocation(prop, payloadArg), payloadArg)
        }
        return this.payload(this.checker.getTypeAtLocation(payloadArg), payloadArg)
    }

    sendPayload(payloadArg, call, classNode) {
        if (!payloadArg) return this.payload(null)
        return this.forwardedParameter(payloadArg, call, classNode) || this.payload(this.checker.getTypeAtLocation(payloadArg), payloadArg)
    }

    // `send(name, data)` inside `method(data: T)` is typed through the method's parameter.
    forwardedParameter(payloadArg, call, classNode) {
        const SK = this.SK
        const symbol = payloadArg.kind === SK.Identifier && classNode && classNode.name && this.checker.getSymbolAtLocation(payloadArg)
        if (!symbol) return null
        for (const decl of this.h.declarations(symbol)) {
            const param = decl.node()
            const method = param.parent
            if (param.kind !== SK.Parameter || method.kind !== SK.MethodDeclaration || method.parent !== classNode) continue
            let inside = false
            for (let node = call; node && !inside; node = node.parent) inside = node === method
            const methodName = this.memberName(method)
            if (inside && methodName !== null) {
                return { kind: 'param', className: classNode.name.text, methodName, index: method.parameters.indexOf(param) }
            }
        }
        return null
    }

    payload(type, location) {
        if (!type) return { kind: 'type', type: 'undefined' }
        return { kind: 'type', type: new TypePrinter(this, location).print(type, 0) ?? 'unknown' }
    }

    lineOf(node) {
        return this.sf.getLineAndCharacterOfPosition(skipTrivia(this.sf.text, node.pos)).line + 1
    }

    sourceOf(node) {
        return this.sf.text.slice(skipTrivia(this.sf.text, node.pos), node.end).replace(/\s+/g, ' ')
    }

    warn(line, message) {
        this.result.warnings.push({ line, message })
    }
}

// Renders a type the generated file can resolve: exported project types through `import('…')`,
// globals by name, anything else expanded. Returns null when the type cannot be expressed.
class TypePrinter {
    constructor(analyzer, location) {
        this.a = analyzer
        this.h = analyzer.h
        this.checker = analyzer.checker
        this.location = location
        this.stack = new Set()
    }

    print(type, depth) {
        if (depth > 8 || this.stack.has(type)) return 'unknown'
        this.stack.add(type)
        try {
            return this.printType(type, depth)
        } finally {
            this.stack.delete(type)
        }
    }

    printType(type, depth) {
        const TF = this.h.TF
        const f = type.flags
        if (f & TF.Any) return (type.isErrorType ? type.isErrorType() : type.intrinsicName === 'error') ? null : 'any'
        if (f & TF.StringLiteral) return JSON.stringify(type.value)
        if (f & (TF.TemplateLiteral | TF.StringMapping)) return 'string'
        if (f & (TF.Unknown | TF.String | TF.Number | TF.Boolean | TF.BigInt | TF.Void | TF.Undefined | TF.Null | TF.Never | TF.NonPrimitive | TF.Literal)) {
            return this.checker.typeToString(type)
        }
        if (f & TF.TypeParameter) {
            const constraint = this.checker.getBaseConstraintOfType(type)
            return constraint && constraint !== type ? this.print(constraint, depth + 1) : 'unknown'
        }
        if (f & (TF.Union | TF.Intersection)) {
            const parts = this.h.members(type).map((member) => this.wrap(this.print(member, depth + 1)))
            return parts.includes(null) ? null : [...new Set(parts)].join(f & TF.Union ? ' | ' : ' & ')
        }
        const alias = this.h.aliasSymbol(type)
        const aliasRef = alias && this.reference(alias, this.h.aliasTypeArguments(type), depth)
        if (aliasRef) return aliasRef
        if (f & TF.Object) return this.printObject(type, depth)
        return 'unknown'
    }

    printObject(type, depth) {
        const checker = this.checker
        if (checker.isArrayType(type) || checker.isTupleType(type)) {
            const parts = checker.getTypeArguments(type).map((arg) => this.print(arg, depth + 1))
            if (parts.includes(null)) return null
            return checker.isArrayType(type) ? `${this.wrap(parts[0])}[]` : `[${parts.join(', ')}]`
        }
        const symbol = this.h.typeSymbol(type)
        const args = type.objectFlags & this.h.OF.Reference ? checker.getTypeArguments(type).slice(0, this.h.typeParameterCount(type)) : []
        const ref = symbol && this.reference(symbol, args, depth)
        if (ref) return ref
        if (checker.getSignaturesOfType(type, this.h.CALL).length > 0) return 'unknown'

        const members = []
        for (const prop of checker.getPropertiesOfType(type)) {
            if (prop.flags & this.h.SF.Method || /^(__@|#)/.test(prop.name)) continue
            const optional = prop.flags & this.h.SF.Optional
            let propType = checker.getTypeOfSymbolAtLocation(prop, this.location)
            if (optional && propType.flags & this.h.TF.Union) {
                const defined = this.h.members(propType).filter((m) => !(m.flags & this.h.TF.Undefined))
                if (defined.length === 1) propType = defined[0]
            }
            const printed = this.print(propType, depth + 1)
            if (printed === null) return null
            const key = /^[A-Za-z_$][\w$]*$/.test(prop.name) ? prop.name : JSON.stringify(prop.name)
            members.push(`${key}${optional ? '?' : ''}: ${printed}`)
        }
        return members.length === 0 ? '{}' : `{ ${members.join('; ')} }`
    }

    reference(symbol, typeArgs, depth) {
        const decl = this.h.declarations(symbol)[0]
        const node = decl && decl.node()
        if (!node) return null
        const sf = node.getSourceFile()
        let base
        if (!sf.externalModuleIndicator && sf.isDeclarationFile) {
            base = symbol.name
        } else if (!/[\\/]node_modules[\\/]/.test(sf.fileName) && this.isExported(node, sf)) {
            base = `import(${IMPORT_MARK}${path.resolve(sf.fileName).replace(/\.(d\.)?[mc]?tsx?$/, '')}${IMPORT_MARK}).${symbol.name}`
        } else {
            return null
        }
        const args = typeArgs.map((arg) => this.print(arg, depth + 1))
        if (args.includes(null)) return null
        return args.length ? `${base}<${args.join(', ')}>` : base
    }

    isExported(node, sf) {
        const SK = this.a.SK
        const declarationKinds = [SK.InterfaceDeclaration, SK.TypeAliasDeclaration, SK.ClassDeclaration, SK.EnumDeclaration]
        const modifiers = (node.modifiers || []).map((m) => m.kind)
        return node.parent === sf && declarationKinds.includes(node.kind) && modifiers.includes(SK.ExportKeyword) && !modifiers.includes(SK.DefaultKeyword)
    }

    wrap(text) {
        return text && /[|&]/.test(text) && !text.startsWith('{') ? `(${text})` : text
    }
}

function skipTrivia(text, pos) {
    const match = /^(?:\s|\/\/[^\n]*|\/\*[\s\S]*?\*\/)*/.exec(text.slice(pos))
    return pos + match[0].length
}

async function analyze(request) {
    const tsDir = request.typescript || path.dirname(require.resolve('typescript/package.json', { paths: [request.projectPath] }))
    const host = await loadHost(tsDir)

    const configPath = path.join(__dirname, `typegen-${process.pid}.tsconfig.json`)
    const config = { compilerOptions: { noEmit: true, skipLibCheck: true }, files: request.files, include: [] }
    if (request.tsconfig) config.extends = request.tsconfig
    else Object.assign(config.compilerOptions, { target: 'es2022', module: 'esnext', moduleResolution: 'bundler', experimentalDecorators: true, strict: true })
    fs.writeFileSync(configPath, JSON.stringify(config))

    const program = host.open(configPath)
    try {
        const analyzer = new Analyzer(host, program)
        const files = {}
        for (const file of request.files) files[file] = analyzer.analyzeFile(file)
        return { files, dependencies: program.fileNames().filter((name) => !/[\\/]node_modules[\\/]/.test(name)) }
    } finally {
        program.close()
        fs.rmSync(configPath, { force: true })
    }
}

analyze(JSON.parse(fs.readFileSync(0, 'utf8'))).then(
    (response) => {
        process.stdout.write(JSON.stringify(response), () => process.exit(0))
    },
    (err) => {
        process.stderr.write(err && err.message ? err.message : String(err))
        process.exit(1)
    },
)
