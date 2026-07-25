## OpenCore CLI v1.5.1

### Fix for FiveM Enhanced msgpack codec

Fixed server resources failing on every event and cross-resource export under FiveM Enhanced (Node 26) with:

```text
this.codec.encode is not a function
this.codec.decode is not a function
```

Large CJS server bundles declare their modules at the script top level, which corrupts the runtime's global msgpack codec and breaks command registration and net events.

The server bundle is now wrapped in an IIFE so those declarations live in a function scope instead of the top level. This applies only to server builds using the default `cjs` format, client builds already use `iife` and are unaffected.
