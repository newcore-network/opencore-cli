## OpenCore CLI v0.5.1

### Highlights
- **Correct LogLevel Propagation**: Ensured `LogLevel` and OpenCore-specific defines (`__OPENCORE_LOG_LEVEL__`, `__OPENCORE_TARGET__`) are correctly applied to all resource build tasks, including UI/Views.

### Changes
- **Build Defines**: Added `__OPENCORE_LOG_LEVEL__` and `__OPENCORE_TARGET__` defines to all resource and standalone builds to prevent `ReferenceError`.

### Fixes
- Fixed `ReferenceError: __OPENCORE_LOG_LEVEL__ is not defined` in satellite resources.
- Repository url fixed for `opencore clone --list`