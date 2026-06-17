## OpenCore CLI v1.4.1

### Fix — Dev mode build loop

Fixed an infinite rebuild loop that could happen when running:

- `opencore build`
- `opencore dev`

The watcher now ignores build-generated artifacts (including `.opencore/autoload.*.controllers.ts`) and output/deploy directories, preventing recursive self-triggered rebuilds.  
Result: dev mode stabilizes after the initial build and only rebuilds on real source changes.
