## OpenCore CLI v1.6.2

### Typegen on large resources

Resources whose analysis is larger than 64 KB (a few hundred source files) no longer skip type
generation with `typegen analyzer returned invalid JSON: unexpected end of JSON input`. The analyzer
exited before Node had flushed its output to the CLI, so only the first 64 KB arrived and that
resource's `.opencore/opencore.gen.ts` got no events, RPCs or commands.
