# LAN File Share

一个基于 Go + Web 的局域网文件共享工具，支持多端协同（Windows / macOS / Linux / Android / iOS）。

## ✨ 功能

- **📂 文件浏览** — 网格/列表视图，目录导航，面包屑路径
- **📤 文件上传** — 拖拽上传、选择文件/文件夹，上传进度条
- **📥 文件下载** — 单个文件下载，文件夹自动打包 ZIP
- **👁️ 在线预览** — 图片、视频、音频、PDF、文本/代码文件
- **🔗 分享链接** — 生成带密码和过期时间的分享链接
- **🔍 搜索过滤** — 实时搜索当前目录文件
- **🌓 深色模式** — 明暗主题切换
- **🔒 权限控制** — 可选 HTTP Basic Auth 保护

## 🚀 快速开始

### 方式一：从源码构建

```bash
# 需要安装 Go 1.21+
git clone <repo-url> lan-file-share
cd lan-file-share

# 构建 Windows 可执行文件
go build -o lan-file-share.exe

# 运行
lan-file-share.exe --port 8080 --dir ./shared
```

### 方式二：直接运行（无需构建）

```bash
go run . --port 8080
```

### 访问

启动后在浏览器访问：

- 本机：`http://localhost:8080`
- 局域网其他设备：`http://<本机IP>:8080`

所有连接到同一局域网（或同一 WiFi）的设备都可以通过浏览器访问。

## ⚙️ 命令行参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--port` | `8080` | 服务端口 |
| `--dir` | `./shared` | 共享目录路径 |
| `--max-upload` | `500` | 最大上传大小（MB） |
| `--user` | `""` | 认证用户名（空=无认证） |
| `--pass` | `""` | 认证密码 |
| `--config` | `""` | JSON 配置文件路径 |

### 配置示例（config.json）

```json
{
    "port": 8080,
    "shared_dir": "C:\\Users\\Me\\Share",
    "max_upload_mb": 1000,
    "auth_user": "admin",
    "auth_pass": "mypassword"
}
```

使用配置文件：
```bash
lan-file-share.exe --config config.json
```

## 📡 架构

```
┌──────────────┐     HTTP/HTTPS     ┌──────────────────────┐
│  Browser     │ ◄────────────────► │  Go Server (:8080)    │
│  (any OS)    │                    │  ├─ /api/list         │
│              │                    │  ├─ /api/upload       │
│              │                    │  ├─ /api/download     │
│              │                    │  ├─ /api/preview      │
│              │                    │  ├─ /api/share        │
│              │                    │  └─ /share/:token     │
└──────────────┘                    └───────┬──────────────┘
                                            │
                                    ┌───────▼──────────────┐
                                    │  Shared Directory     │
                                    │  (./shared/)          │
                                    └──────────────────────┘
```

## 🛠 技术栈

- **后端**: Go (net/http, embed, archive/zip)
- **前端**: 原生 HTML/CSS/JavaScript（零依赖）
- **分发**: 单文件二进制，前端嵌入 Go 二进制

## 📝 许可

MIT
