# LAN File Share

基于 Go + 原生 HTML/JS 的局域网文件共享工具，单文件分发，支持多端协同。

## Project

- **Stack**: Go 1.21+ (`net/http`, `embed`, `archive/zip`), 原生 HTML/CSS/JS（零前端依赖, 除 highlight.js CDN）
- **Entry point**: `main.go` — CLI flags, `//go:embed static/*`, 启动 `server.FileServer`
- **输出**: 单文件 PE32+ 可执行（`lan-file-share.exe`, ~8MB），前端完全嵌入二进制
- **管理分离**: 共享页面（`index.html`）无管理入口；管理面板（`admin.html`）仅本机访问

## Commands

```bash
# 构建（Windows, 无 GUI）
go build -ldflags="-s -w" -o lan-file-share.exe .

# 构建（含 GUI 系统托盘, 需要 mingw-w64 + CGO_ENABLED=1）
set CGO_ENABLED=1
go build -ldflags="-s -w" -tags gui -o lan-file-share.exe .

# 运行
lan-file-share.exe --port 8080 --dir ./shared
lan-file-share.exe --port 8080 --dir ./shared --user admin --pass secret
lan-file-share.exe --config config.json

# 构建脚本
build.bat
```

**Flags**: `--port` (8080), `--dir` (./shared), `--max-upload` (500 MB), `--user`, `--pass`, `--config`.
**GUI 构建前提**: `mingw-w64` (x86_64-win32-seh-ucrt), `CGO_ENABLED=1`, `-tags gui`。

## Architecture

```
main.go  ← embed static/*
  └── server.New(cfg, embedFS)
        └── FileServer.Start() → mux routes:
              ├── /                    → renderIndex() (index.html 模板)
              ├── /admin.html          → renderAdmin() (admin.html 静态)
              ├── /static/*            → embedded FS (css/style.css, js/app.js)
              ├── /api/list            → handlers.go ListHandler
              ├── /api/upload          → upload.go UploadHandler (multipart)
              ├── /api/download        → handlers.go DownloadHandler (single + zip dir)
              ├── /api/preview         → handlers.go PreviewHandler (mime-type)
              ├── /api/delete          → handlers.go DeleteHandler
              ├── /api/folder          → handlers.go CreateFolderHandler
              ├── /api/share           → share.go ShareHandler (CRUD)
              ├── /api/archive         → archive.go ArchivePreviewHandler (zip listing)
              ├── /api/info            → server.go InfoHandler (incl. guest_permissions)
              ├── /api/permissions     → permissions.go PermissionsHandler
              ├── /api/admin/config    → admin.go AdminHandler (GET/PUT)
              ├── /api/admin/status    → admin.go AdminStatusHandler
              ├── /api/admin/toggle    → server.go AdminToggleHandler (PAUSE/RESUME)
              └── /share/             → share.go AccessSharedHandler
```

| Layer | File | Role |
|-------|------|------|
| `main.go` | — | Entry, CLI parsing, `embed.FS`, `server.New()` |
| `gui/` | `gui.go` | 系统托盘 GUI (CGO, build tag `gui`), `Run()` / `Stop()` |
| `server/` | `config.go` | Config struct, `GuestPermissions`, `LoadConfig()`, `SaveConfig()`, `LocalIP()` |
| | `server.go` | `FileServer` struct, route registration, `Start()`, pause/resume, middlewares, `renderAdmin()` |
| | `middleware.go` | `CORS`, `Logging`, `BasicAuth`, `AuthChallenge` |
| | `handlers.go` | List/Download/Preview/Delete/CreateFolder handlers |
| | `upload.go` | Multipart file/folder upload with progress |
| | `share.go` | Share link CRUD + password/expiry access |
| | `archive.go` | ZIP content listing (native stream), rar/7z info |
| | `admin.go` | Admin config GET/PUT, status endpoint |
| | `permissions.go` | `HasPermission()`, `isAuthenticated()`, `CheckPermission()` |
| | `zip.go` | Directory-to-ZIP streaming |
| `models/` | `share.go` | `ShareLink` + `ShareStore` (in-memory, mutex-guarded) |
| `static/` | `index.html` | 中文共享界面（无管理入口）, 帮助弹窗内嵌, highlight.js CDN |
| | `admin.html` | 独立管理面板（服务启停、端口、访客权限） |
| | `css/style.css` | Responsive layout, dark/light theme, gallery, archive, skeleton, selection |
| | `js/app.js` | SPA — gallery, upload, share, permissions-aware UI, batch selection/delete |

**Middleware chain**: `CORS` → `Logging` → `pauseMiddleware`.  
- `CORS`: 允许所有 Origin（局域网跨域）。  
- `pausedMiddleware`: 当 `s.paused == true` 时仅放行 `/api/admin/*`, `/admin.html`, `/static/*`，其余返回 503。  
- Auth **不**在中间件层施加；每个 handler 按需调用 `CheckPermission(w, r, perm)`（返回 401 或 403）。

**Data flow**: All handlers on `*FileServer` struct — `s.config`, `s.shareStore`.  
- 共享目录 (`s.config.SharedDir`) 是唯一持久存储。  
- 分享链接在内存中（`time.Ticker` 每小时清理）。  
- 配置可通过 `SaveConfig()` 持久化到 `--config` 路径。

**Permission flow**（当 `--user` 设置时）:  
1. `/api/info` 返回 `guest_permissions` → 前端存入 `state.permissions`  
2. 前端尝试调用 `/api/admin/config` → 若 401 则用户为访客  
3. 前端根据 `state.permissions` 隐藏受限 UI 按钮

**GUI flow**（`-tags gui` 构建时）:  
1. `main()` 启动 HTTP 服务后调用 `gui.Run(srv, cfg)`  
2. 系统托盘图标显示 → 右键菜单: 打开浏览器 / 暂停恢复 / 退出  
3. 退出时调用 `gui.Stop()` 关闭 server + 退出程序

## Conventions

- **Error handling**: handlers 返回 `http.Error` 表示 4xx/5xx；`writeJSON(w, status, data)` 表示 API 成功。`CheckPermission()` 在未认证时发送 401（`AuthChallenge`），已认证但无权限时发送 403。
- **JSON API**: 所有 `/api/*` 端点使用 JSON。前端使用 `API.get/post/delete/put` 封装。
- **Path safety**: 共享目录内的相对路径。`sanitizePath()` 移除 `..` 和开头的 `/`。所有文件访问使用 `strings.HasPrefix(absFull, absShared)` 验证。
- **Naming**: Go — PascalCase 导出, camelCase 私有。JS — camelCase。CSS — kebab-case。
- **Frontend**: 单 `app.js`；全局 `state` 对象；`state.selectedFiles` (Set) 管理多选。DOM 通过 `getElementById` 引用。文件列表通过 `renderFiles()` 重建（不保留 DOM 状态）。
- **Selection/batch**: Ctrl+点击切换多选, Ctrl+A 全选, Delete 批量删除。选中后出现选择栏（`#selectionBar`）提供批量下载/删除/取消。
- **Code preview**: 文本/代码文件使用 highlight.js (CDN) 语法高亮 + 语言自动检测。顶部"复制代码"按钮。预览在 `#previewModal` 中。
- **Skeleton**: 文件列表加载时显示 8 格 shimmer 动画占位（`.skeleton-grid`），替代纯文字 loading。
- **Icons**: Emoji 图标（无图标库）。主题色存储在 `localStorage`。
- **Auth**: 可选 HTTP Basic Auth，SHA-256 哈希 + constant-time 比较。每个 handler 通过 `CheckPermission()` 按需调用。
- **管理分离**: 共享页面（`index.html`）无任何管理入口；`admin.html` 通过 `127.0.0.1/admin.html` 访问，专门控制服务启停和配置。管理 API 由 pause 中间件保护。
- **Embed**: `//go:embed static/*` — 不要修改 embed 指令。
- **Build (no GUI)**: `go build -ldflags="-s -w" -o lan-file-share.exe .`
- **Build (with GUI)**: 需要 `mingw-w64`, `CGO_ENABLED=1`, `-tags gui`。

## Notes

- 共享目录是安全边界：所有文件操作被限定在其中。
- 分享链接是临时的（重启后清空）。
- catch-all handler (`/`) 同时是 index 路由和共享目录回退文件服务。
- 画廊模式：打开图片时自动索引当前目录所有图片，支持 ← → 导航。
- 复制到剪贴板使用 `copyToClipboard(text)` + `fallbackCopy()`（兼容 HTTP）。
- 管理面板无法从共享页面导航到达，需在地址栏直接输入 `http://localhost:8080/admin.html`。
- highlight.js 从 CDN 加载 (`cdnjs.cloudflare.com`)，离线环境需自托管。
- GUI 构建需要先安装 mingw-w64 (x86_64-win32-seh-ucrt 版本) 并确保在 PATH 中。
