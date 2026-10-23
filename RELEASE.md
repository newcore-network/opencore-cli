## OpenCore CLI v1.7.0

### Typed library events

Typegen now types the events of OpenCore libraries, in the new `serverLibraryEvents` and
`clientLibraryEvents` maps of each resource's `.opencore/opencore.gen.ts`, keyed by library name:

- `@Server.OnLibraryEvent(library, event)` / `@Client.OnLibraryEvent(library, event)` listeners type
  an event by their first parameter. An optional (`payload?: T`), defaulted or rest parameter keeps
  its declared type.
- When several listeners take a payload, the event is typed by the intersection of their
  parameters, since the payload reaches each of them. Listeners that disagree make `emit()` fail to
  compile instead of one of them being dropped silently.
- A listener that ignores the payload says nothing about it: the event is typed by its `emit()`
  calls, or as `unknown` when there are none.
- `library.emit(event, payload)` calls on a library created from a literal name type the events no
  listener types, with the union of the payloads they pass. A payload forwarded from a method
  parameter is typed through that parameter when the class is exported, and from the argument
  otherwise.

Library and event names must resolve to string literals; a plain `string` is skipped with a
warning. Typed library events require `@open-core/framework` with library event support in its
`Register`.

### Library name collisions

The framework merges the events of every library sharing a name across the project, although each
resource has its own library bus at runtime. Typegen now warns, at the `createServerLibrary` /
`createClientLibrary` call, when another resource creates a library of the same side and name:

```
server library "core" is also created by resource:resources/characters; typegen merges the events
of libraries sharing a name, so give one of them a different name
```

The factory is found through its symbol, so aliased imports, re-exports and
`Server.createServerLibrary` count. Listening to or emitting on a library does not, and libraries
created from a non-literal name are not checked.

`opencore build` now generates the types of every resource before bundling, with or without
`build.typecheck`, so both resources of a collision report it on the first build. In `opencore dev`
a resource reports its warnings once per session.

### Typegen fixes

- `@Client.OnView` handlers with an optional, defaulted or rest first parameter no longer type their
  WebView message as `undefined`.
- A `WebView.send()` that forwards a method parameter of a class that is not exported is typed from
  its argument, instead of importing the class and breaking the generated file.
- The `__Payload` helper is only declared when a map uses it, so projects compiled with
  `noUnusedLocals` no longer fail on it.
- The typegen summary reads `typegen: N warning(s) in <resource>`: it also counts `send()` and
  `emit()` calls, which are not handlers.
