# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

LAN File Share — a Go-based LAN file sharing tool with an embedded native HTML/CSS/JS frontend. Single binary distribution (~8MB), zero frontend dependencies except highlight.js CDN.

**Stack**: Go 1.21+ (`net/http`, `embed`, `archive/zip`), vanilla HTML/CSS/JS with dark/light theme.

## Build Commands

```bash
# Standard build (Windows, no GUI)
go build -ldflags="-s -w" -o lan-file-share.exe .

# Run directly
go run . --port 8080 --dir ./shared

# Build with GUI system tray (requires mingw-w64 + CGO_ENABLED=1)
set CGO_ENABLED=1
go build -ldflags="-s -w" -tags gui -o lan-file-share.exe .

# Quick build script
build.bat
```

**CLI flags**: `--port` (8080), `--dir` (./shared), `--max-upload` (500 MB), `--user`, `--pass`, `--config` (JSON config path).

## Architecture

```
main.go  ← embed static/* (//go:embed static/*)
  └── server.New(cfg, embedFS)
        └── FileServer.Start() → http.ServeMux routes
```

**Middleware chain**: `CORS` → `Logging` → `pauseMiddleware`
- Auth is NOT applied at middleware level; each handler calls `CheckPermission(w, r, perm)` as needed.

**Data flow**: All handlers on `*FileServer` struct — `s.config`, `s.shareStore`.
- Shared directory (`s.config.SharedDir`) is the only persistent storage.
- Share links are in-memory (cleaned hourly by `time.Ticker`).
- Config can be persisted via `SaveConfig()` to `--config` path.

**Permission flow** (when `--user` is set):
1. `/api/info` returns `guest_permissions` → frontend stores in `state.permissions`
2. Frontend tries `/api/admin/config` → 401 means guest user
3. Frontend hides restricted UI buttons based on `state.permissions`

## Key Files

| File | Role |
|------|------|
| `main.go` | Entry point, CLI parsing, `embed.FS`, `server.New()` |
| `server/server.go` | `FileServer` struct, route registration, `Start()`, pause/resume, middlewares |
| `server/handlers.go` | List/Download/Preview/Delete/CreateFolder handlers + `validatePath()` helper |
| `server/upload.go` | Multipart file/folder upload + `sanitizePath()` + `writeJSON()` |
| `server/share.go` | Share link CRUD + password/expiry access |
| `server/permissions.go` | `HasPermission()`, `isAuthenticated()`, `CheckPermission()`, permission constants |
| `server/admin.go` | Admin config GET/PUT, status endpoint |
| `server/archive.go` | ZIP content listing (native stream) |
| `server/middleware.go` | `CORS`, `Logging`, `BasicAuth`, `AuthChallenge` |
| `server/config.go` | Config struct, `GuestPermissions`, `LoadConfig()`, `LocalIP()` |
| `models/share.go` | `ShareLink` + `ShareStore` (in-memory, mutex-guarded) |
| `static/js/app.js` | SPA — gallery, upload, share, permissions-aware UI, batch selection/delete |
| `static/css/style.css` | Responsive layout, dark/light theme, skeleton loading |
| `static/index.html` | Shared interface (no admin entry) |
| `static/admin.html` | Standalone admin panel (localhost only) |

## Path Safety

All file operations are bounded to the shared directory:
- `sanitizePath()` removes `..` and leading `/`
- Every handler validates with `strings.HasPrefix(absFull, absShared)` via `validatePath()` helper
- The catch-all handler (`/`) serves both index route and shared directory fallback

## Conventions

- **Error handling**: Handlers return `http.Error` for 4xx/5xx; `writeJSON(w, status, data)` for API success.
- **JSON API**: All `/api/*` endpoints use JSON. Frontend uses `API.get/post/delete/put` wrapper.
- **Naming**: Go — PascalCase exports, camelCase private. JS — camelCase. CSS — kebab-case.
- **Frontend**: Single `app.js`; global `state` object; `state.selectedFiles` (Set) for multi-select.
- **Auth**: Optional HTTP Basic Auth, SHA-256 hash + constant-time comparison via `crypto/subtle`.
- **Concurrency**: `FileServer.paused` uses `atomic.Bool` for thread-safe pause state.
- **Embed**: `//go:embed static/*` — do not modify embed directives.
- **Admin separation**: `index.html` has no admin entry; `admin.html` accessible only via `127.0.0.1/admin.html`.

## Important Notes

- The shared directory is the security boundary — all file operations are confined within it.
- Share links are temporary (cleared on restart).
- Gallery mode: opening an image indexes all images in current directory, supports ← → navigation.
- Clipboard uses `copyToClipboard(text)` + `fallbackCopy()` for HTTP compatibility.
- highlight.js loads from CDN (`cdnjs.cloudflare.com`) — offline environments need self-hosting.
- No test files exist in this project.
