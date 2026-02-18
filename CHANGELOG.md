# Changelog

All notable changes to SyncClipboard-magisk are documented in this file.

## [2.6.2] - 2026-02-18

### Added
- Added root/system clipboard pipeline based on `system_server` hook protocol (`state/command/ack`).
- Added `SyncClipboardSystemHook` module packaging into the standard release ZIP.
- Added runtime hook storage under `/data/system/syncclipboard_hook`.

### Changed
- Switched clipboard primary route to `system_hook` in `clipserver` strategy.
- Updated release/build pipeline to compile and sign the system hook APK in CI.
- Updated startup scripts to prepare hook runtime directory at boot.

### Fixed
- Fixed Android/ROM permission issues caused by previous hook path under `/data/adb/...` by moving to `/data/system/...`.
- Fixed CI Gradle build failures (project wiring, settings ordering, Java compatibility).
- Fixed clipboard API and sync flows to use `system_hook` successfully when LSPosed scope is enabled.

### Deprecated
- Deprecated helper-app-first clipboard route as the primary path.

### Compatibility Notes
- Requires LSPosed module enabled for `android` / `system_server` scope.
- Requires root environment (Magisk/KernelSU/APatch with Zygisk enabled).
- Web UI features remain unchanged; no functional removal in existing UI endpoints.

### Validation Summary (2.6.2)
- `GET /api/clipboard`: PASS via `method=system_hook`
- `POST /api/accounts/test`: PASS
- `POST /api/sync/now`: PASS
