# Changelog

All notable changes to SyncClipboard-magisk are documented in this file.

## [2.6.12] - 2026-03-12

### Fixed
- Fixed `panic: runtime error: slice bounds out of range [:8] with length 0` in `sync/manager.go` when WebDAV server returns clipboard data without a `hash` field (e.g. older server versions).
- `GetHash()` now lazy-computes and caches a SHA256 hash from `Text` when the `hash` field is empty, preventing nil-slice panics at all 4 call sites.
- Added `shortHash()` safety helper in `manager.go` to guard all log truncation slices against short/empty strings.
- Added regression tests: `TestGetHashEmptyFieldLazyCompute`, `TestGetHashBothEmptyReturnsEmpty`, `TestDownloadNowEmptyHashNoPanic`, `TestShortHash`.

## [2.6.11] - 2026-03-04

### Fixed
- Fixed APatch first-open freeze issue by normalizing `clipserver` process cgroups at startup.
- Added startup cgroup migration in `service.sh`: move process out of `freezer:/frozen` into thaw/background groups to avoid inherited top-app frozen state.
- Added cgroup summary logging on startup for faster field diagnostics.

## [2.6.10] - 2026-02-18

### Fixed
- Fixed WebUI stutter/blank risk by adding in-flight guards to prevent overlapping periodic `refreshClipboard`/`refreshStatus` requests.
- Reduced polling pressure (clipboard: 3s, status: 5s) to improve Module Manager WebView stability on constrained ROM environments.

## [2.6.9] - 2026-02-18

### Changed
- Added two explicit auto-sync switches in config/Web UI: `auto_upload_enabled` and `auto_download_enabled`.
- Set both auto switches default to disabled (manual-first by default).
- `POST /api/sync/now` remains manual upload; `POST /api/sync/pull` remains optional manual pull.

### Fixed
- Fixed mixed upload/pull behavior in auto mode by splitting monitor upload and ticker download paths according to independent switches.

## [2.6.8] - 2026-02-18

### Fixed
- Fixed manual upload path reliability by making `POST /api/sync/now` upload local clipboard content to WebDAV by default (manual-first behavior).
- Added explicit manual pull API `POST /api/sync/pull` so download is available when needed without overriding local clipboard during upload tests.
- Updated Web UI action buttons to match behavior: `立即上传` (default) and optional `拉取远端`.

### Changed
- Kept auto sync optional via existing `enabled` setting, while manual operation is now the default interaction path.

## [2.6.7] - 2026-02-18

### Fixed
- Fixed CI compile failure in `HookProtocol.applyWithCapturedTemplate`: reflective `Method.invoke(...)` checked exceptions are now handled inside the method.
- When template invocation fails at runtime, the hook now logs the failure and safely falls back to the ClipboardManager path instead of crashing compile/runtime flow.

## [2.6.6] - 2026-02-18

### Fixed
- Fixed `system_hook` write path false-positive: command apply no longer relies on direct state-file overwrite to indicate success.
- Captured real `ClipboardService` set invocation templates from live system calls and re-used them for command write-back, improving compatibility with ColorOS clipboard routing.
- Added fallback path to `ClipboardManager` only when no captured invocation template is available.

## [2.6.5] - 2026-02-18

### Fixed
- Fixed `POST /api/sync/now` direction logic: when remote WebDAV clipboard differs from local clipboard, it now pulls remote text and writes it to system clipboard first instead of always pushing local content.
- Fixed `service.sh` stale supervisor detection by replacing `kill -0` checks with `/proc/<pid>` existence checks, preventing false-positive "already running" states on some ROM shell environments.

### Added
- Added regression tests for `SyncNow` pull/push decisions and download-failure fallback behavior.

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
