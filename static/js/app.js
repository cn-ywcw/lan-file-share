/* ===== 局域网文件共享 - 主应用 ===== */

// ===== 常量定义 =====
const IMAGE_EXTS = new Set(['jpg', 'jpeg', 'png', 'gif', 'svg', 'webp', 'bmp', 'ico']);
const VIDEO_EXTS = new Set(['mp4', 'webm', 'avi', 'mov', 'mkv']);
const AUDIO_EXTS = new Set(['mp3', 'wav', 'ogg', 'flac', 'aac', 'm4a']);
const ARCHIVE_EXTS = new Set(['zip', 'rar', '7z', 'tar', 'gz', 'tgz']);
const TEXT_EXTS = new Set(['txt', 'md', 'html', 'htm', 'css', 'js', 'json', 'xml', 'yaml', 'yml',
    'go', 'py', 'java', 'c', 'cpp', 'h', 'sh', 'bat', 'ps1', 'log', 'csv', 'toml', 'conf',
    'ini', 'cfg', 'env', 'sql', 'php', 'rb', 'rs', 'ts', 'jsx', 'tsx', 'vue', 'svelte']);

const LANG_MAP = {
    'js': 'javascript', 'ts': 'typescript', 'jsx': 'javascript', 'tsx': 'typescript',
    'py': 'python', 'go': 'go', 'java': 'java', 'c': 'c', 'cpp': 'cpp', 'h': 'c',
    'rs': 'rust', 'rb': 'ruby', 'php': 'php', 'sh': 'bash', 'bat': 'dos', 'ps1': 'powershell',
    'css': 'css', 'html': 'xml', 'htm': 'xml', 'json': 'json', 'xml': 'xml', 'yaml': 'yaml',
    'yml': 'yaml', 'md': 'markdown', 'sql': 'sql', 'vue': 'html', 'svelte': 'html',
};

const EXT_ICONS = {
    'jpg': '🖼️', 'jpeg': '🖼️', 'png': '🖼️', 'gif': '🖼️', 'svg': '🖼️', 'webp': '🖼️', 'bmp': '🖼️', 'ico': '🖼️',
    'mp4': '🎬', 'webm': '🎬', 'avi': '🎬', 'mov': '🎬', 'mkv': '🎬',
    'mp3': '🎵', 'wav': '🎵', 'ogg': '🎵', 'flac': '🎵', 'aac': '🎵', 'm4a': '🎵',
    'pdf': '📕',
    'zip': '📦', 'rar': '📦', '7z': '📦', 'tar': '📦', 'gz': '📦',
    'doc': '📄', 'docx': '📄', 'xls': '📊', 'xlsx': '📊', 'ppt': '📽️', 'pptx': '📽️',
    'txt': '📝', 'md': '📝',
    'html': '🌐', 'htm': '🌐', 'css': '🎨', 'js': '⚡', 'ts': '⚡', 'json': '📋', 'xml': '📋',
    'go': '🔵', 'py': '🐍', 'java': '☕', 'c': '⚙️', 'cpp': '⚙️', 'h': '⚙️', 'rs': '🦀',
    'sh': '💻', 'bat': '💻', 'ps1': '💻', 'yaml': '📋', 'yml': '📋',
};

// ===== DOM 缓存 =====
const _escapeDiv = document.createElement('div');

// ===== 状态管理 =====
let state = {
    currentPath: '.',
    viewMode: 'grid',
    files: [],
    shareLinks: [],
    previewFile: null,
    previewIndex: -1,
    imageList: [],
    serverInfo: null,
    selectedFiles: new Set(),
    permissions: {
        browse: true, upload: true, download: true,
        preview: true, delete: true, share: true,
        isAdmin: true,
    },
    permCategories: [],
};

// ===== API 封装 =====
const API = {
    async get(path) {
        const res = await fetch(path);
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return res.json();
    },
    async post(path, body) {
        const res = await fetch(path, {
            method: 'POST',
            headers: body instanceof FormData ? {} : { 'Content-Type': 'application/json' },
            body: body instanceof FormData ? body : JSON.stringify(body),
        });
        if (!res.ok) {
            const err = await res.text();
            throw new Error(err || `HTTP ${res.status}`);
        }
        return res.json();
    },
    async delete(path, body) {
        const res = await fetch(path, {
            method: 'DELETE',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body),
        });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return res.json();
    },
};

// ===== 初始化 =====
document.addEventListener('DOMContentLoaded', () => {
    loadServerInfo();
    loadFileList();
    loadShareLinks();
    setupDragDrop();
    setupFileInputs();
    setupKeyboard();
    loadTheme();
});

async function loadServerInfo() {
    try {
        const info = await API.get('/api/info');
        state.serverInfo = info;
        document.getElementById('serverInfo').textContent = `📍 ${info.local_ip}:${info.port}`;
        document.getElementById('helpLocalUrl').textContent = `http://localhost:${info.port}`;
        document.getElementById('helpLanUrl').textContent = `http://${info.local_ip}:${info.port}`;
        // Load permissions after we have server info
        loadPermissions();
    } catch (e) {
        document.getElementById('serverInfo').textContent = '⚠️ 无法连接服务器';
    }
}

// Load effective permissions + category definitions from /api/permissions
async function loadPermissions() {
    // Default fallback: full access
    state.permissions = {
        browse: true, upload: true, download: true,
        preview: true, delete: true, share: true,
        isAdmin: true,
    };
    state.permCategories = [];

    try {
        const data = await API.get('/api/permissions');
        state.permissions = data.permissions || state.permissions;
        state.permCategories = data.categories || [];
    } catch (e) {
        // Fallback: try loading from serverInfo
        if (state.serverInfo && state.serverInfo.guest_permissions) {
            const gp = state.serverInfo.guest_permissions;
            state.permissions = {
                browse: gp.browse !== false,
                upload: gp.upload !== false,
                download: gp.download !== false,
                preview: gp.preview !== false,
                delete: gp.delete !== false,
                share: gp.share !== false,
                isAdmin: !state.serverInfo.has_auth,
            };
        }
    }
    applyUIPermissions();
}

// ===== 文件浏览 =====
async function loadFileList(path) {
    if (path !== undefined) state.currentPath = path;
    const container = document.getElementById('fileContainer');
    container.innerHTML = '<div class="skeleton-grid">' + Array(8).fill('<div class="skeleton-item"></div>').join('') + '</div>';

    try {
        const data = await API.get(`/api/list?path=${encodeURIComponent(state.currentPath)}`);
        state.files = data.files || [];
        // Build image list for gallery
        state.imageList = state.files
            .filter(f => !f.is_dir && /\.(jpg|jpeg|png|gif|svg|webp|bmp|ico)$/i.test(f.name))
            .map(f => f.path);
        renderFiles();
        renderBreadcrumb();
        updatePathDisplay();
    } catch (e) {
        container.innerHTML = `<div class="empty-state">
            <div class="empty-state-icon">⚠️</div>
            <div class="empty-state-title">加载失败</div>
            <div class="empty-state-desc">${escapeHtml(e.message)}</div>
            <button class="btn btn-primary" style="margin-top:16px" onclick="refreshList()">🔄 重试</button>
        </div>`;
    }
}

function renderFiles() {
    const container = document.getElementById('fileContainer');
    container.className = `file-container ${state.viewMode}-view`;

    const query = document.getElementById('searchBox').value.toLowerCase().trim();
    let files = state.files;
    if (query) {
        files = files.filter(f => f.name.toLowerCase().includes(query));
    }

    if (files.length === 0) {
        const isFiltered = query && state.files.length > 0;
        container.innerHTML = isFiltered
            ? `<div class="empty-state">
                <div class="empty-state-icon">🔍</div>
                <div class="empty-state-title">没有匹配"${escapeHtml(query)}"的文件</div>
                <div class="empty-state-desc">尝试其他关键词搜索</div>
               </div>`
            : `<div class="empty-state">
                <div class="empty-state-icon">📂</div>
                <div class="empty-state-title">此文件夹为空</div>
                <div class="empty-state-desc">拖拽文件到左侧上传区或点击工具栏上传按钮添加文件</div>
               </div>`;
        updateFileCount(0);
        return;
    }

    const fragment = document.createDocumentFragment();
    const isGrid = state.viewMode === 'grid';

    files.forEach(f => {
        const item = document.createElement('div');
        const isSelected = state.selectedFiles.has(f.path);
        item.className = `file-item${isSelected ? ' selected' : ''}`;
        item.dataset.path = f.path;
        item.dataset.isDir = f.is_dir;

        const icon = getFileIcon(f);
        const name = escapeHtml(f.name);
        const escapedPath = escapeHtml(f.path);

        // 构建操作按钮（复用逻辑）
        let actionsHtml = '';
        if (isGrid) {
            actionsHtml = buildGridActions(f, escapedPath);
        } else {
            actionsHtml = buildListActions(f, escapedPath);
        }

        if (isGrid) {
            const sizeStr = f.is_dir ? '' : formatSize(f.size);
            item.innerHTML = `
                <div class="file-icon">${icon}</div>
                <div class="file-name">${name}</div>
                <div class="file-size">${sizeStr}</div>
                <div class="grid-actions">${actionsHtml}</div>
            `;
        } else {
            const sizeStr = f.is_dir ? '' : formatSize(f.size);
            item.innerHTML = `
                <div class="file-icon">${icon}</div>
                <div class="file-name">${name}</div>
                <div class="file-size">${sizeStr}</div>
                <div class="file-date">${f.mod_time || ''}</div>
                <div class="file-actions">${actionsHtml}</div>
            `;
        }

        // 绑定事件（统一处理）
        item.addEventListener('dblclick', () => navigateTo(f));
        item.addEventListener('click', (e) => {
            if (e.target.closest('.grid-actions, .file-actions')) return;
            if (e.ctrlKey || e.metaKey) {
                toggleSelect(f.path);
                item.classList.toggle('selected');
            } else {
                selectItem(item);
            }
        });
        item.addEventListener('contextmenu', (e) => {
            e.preventDefault();
            showContextMenu(e, f);
        });

        fragment.appendChild(item);
    });

    container.innerHTML = '';
    container.appendChild(fragment);
    updateFileCount(files.length);
}

// 构建网格视图操作按钮
function buildGridActions(f, escapedPath) {
    let html = '';
    if (f.is_dir) {
        html += `<button class="btn btn-sm" onclick="event.stopPropagation(); downloadFile('${escapedPath}')" title="下载文件夹 (ZIP)">⬇️</button>`;
    } else if (state.permissions.preview) {
        html += `<button class="btn btn-sm" onclick="event.stopPropagation(); previewFile('${escapedPath}')" title="预览">👁️</button>`;
    }
    if (state.permissions.download && !f.is_dir) {
        html += `<button class="btn btn-sm" onclick="event.stopPropagation(); downloadFile('${escapedPath}')" title="下载">⬇️</button>`;
    }
    if (state.permissions.share) {
        html += `<button class="btn btn-sm" onclick="event.stopPropagation(); shareDialog('${escapedPath}', ${f.is_dir})" title="分享">🔗</button>`;
    }
    if (state.permissions.delete) {
        html += `<button class="btn btn-sm btn-danger" onclick="event.stopPropagation(); deleteFile('${escapedPath}')" title="删除">🗑️</button>`;
    }
    return html;
}

// 构建列表视图操作按钮
function buildListActions(f, escapedPath) {
    let html = '';
    if (f.is_dir) {
        if (state.permissions.download) {
            html += `<button class="btn btn-sm" onclick="event.stopPropagation(); downloadFile('${escapedPath}')" title="下载文件夹 (ZIP)">⬇️📁</button>`;
        }
    } else if (state.permissions.preview) {
        html += `<button class="btn btn-sm" onclick="event.stopPropagation(); previewFile('${escapedPath}')" title="预览">👁️</button>`;
    }
    if (state.permissions.download) {
        html += `<button class="btn btn-sm" onclick="event.stopPropagation(); downloadFile('${escapedPath}')" title="下载">⬇️</button>`;
    }
    if (state.permissions.share) {
        html += `<button class="btn btn-sm" onclick="event.stopPropagation(); shareDialog('${escapedPath}', ${f.is_dir})" title="分享">🔗</button>`;
    }
    if (state.permissions.delete) {
        html += `<button class="btn btn-sm btn-danger" onclick="event.stopPropagation(); deleteFile('${escapedPath}')" title="删除">🗑️</button>`;
    }
    return html;
}

function selectItem(el) {
    state.selectedFiles.clear();
    const path = el.dataset.path;
    if (path) state.selectedFiles.add(path);
    document.querySelectorAll('.file-item.selected').forEach(i => i.classList.remove('selected'));
    el.classList.add('selected');
    updateSelectionBar();
}

function toggleSelect(path) {
    if (state.selectedFiles.has(path)) state.selectedFiles.delete(path);
    else state.selectedFiles.add(path);
    updateSelectionBar();
}

function clearSelection() {
    state.selectedFiles.clear();
    updateSelectionBar();
    // Re-render to remove selected class
    document.querySelectorAll('.file-item.selected').forEach(el => el.classList.remove('selected'));
}

function updateSelectionBar() {
    const bar = document.getElementById('selectionBar');
    const count = state.selectedFiles.size;
    if (count === 0) {
        bar.classList.remove('visible');
        return;
    }
    bar.classList.add('visible');
    document.getElementById('selectionCount').textContent = count;
}

function updateFileCount(count) {
    const el = document.getElementById('fileCount');
    if (el) el.textContent = `共 ${count} 个项目`;
}

async function batchDelete() {
    const count = state.selectedFiles.size;
    if (count === 0) return;
    if (!confirm(`确定删除选中的 ${count} 个项目？`)) return;
    const paths = Array.from(state.selectedFiles);
    let success = 0, fail = 0;
    for (const path of paths) {
        try {
            await API.delete('/api/delete', { path });
            success++;
        } catch (e) {
            fail++;
        }
    }
    state.selectedFiles.clear();
    updateSelectionBar();
    loadFileList();
    if (fail === 0) {
        showToast(`成功删除 ${success} 个项目`, 'success');
    } else {
        showToast(`删除完成：${success} 成功，${fail} 失败`, fail > 0 ? 'error' : 'success');
    }
}

function batchDownload() {
    const paths = Array.from(state.selectedFiles);
    if (paths.length === 0) return;
    if (paths.length === 1) {
        downloadFile(paths[0]);
    } else {
        paths.forEach(p => window.open(`/api/download?path=${encodeURIComponent(p)}`, '_blank'));
        showToast(`正在下载 ${paths.length} 个文件`, 'info');
    }
}

function navigateTo(f) {
    if (f.is_dir) {
        state.currentPath = f.path;
        loadFileList(state.currentPath);
    } else {
        previewFile(f.path);
    }
}

function renderBreadcrumb() {
    const el = document.getElementById('breadcrumb');
    const parts = state.currentPath === '.' ? [] : state.currentPath.replace(/\\/g, '/').split('/');
    let html = `<a href="#" data-path="." onclick="event.preventDefault(); loadFileList('.')">🏠 根目录</a>`;
    let cumulative = '';
    parts.forEach((part, i) => {
        cumulative = cumulative ? `${cumulative}/${part}` : part;
        if (i === parts.length - 1) {
            html += `<span class="sep">/</span><span>${escapeHtml(part)}</span>`;
        } else {
            html += `<span class="sep">/</span><a href="#" data-path="${cumulative}" onclick="event.preventDefault(); loadFileList('${cumulative}')">${escapeHtml(part)}</a>`;
        }
    });
    el.innerHTML = html;
}

function updatePathDisplay() {
    document.getElementById('currentPath').textContent = '/' + (state.currentPath === '.' ? '' : state.currentPath.replace(/\\/g, '/'));
}

function refreshList() { loadFileList(); }

// ===== 视图模式 =====
function setView(mode) {
    state.viewMode = mode;
    document.querySelectorAll('.view-toggle .btn').forEach(b => b.classList.toggle('active', b.dataset.view === mode));
    renderFiles();
}

// ===== 搜索 =====
function filterFiles() { renderFiles(); }

// ===== 文件操作 =====
async function downloadFile(path) {
    window.open(`/api/download?path=${encodeURIComponent(path)}`, '_blank');
}

async function deleteFile(path) {
    if (!confirm(`确定删除 "${path}" ？`)) return;
    try {
        await API.delete('/api/delete', { path });
        showToast('删除成功', 'success');
        loadFileList();
    } catch (e) {
        showToast(`删除失败: ${e.message}`, 'error');
    }
}

function newFolder() {
    document.getElementById('folderName').value = '';
    document.getElementById('folderModal').classList.add('active');
    setTimeout(() => document.getElementById('folderName').focus(), 100);
}

function closeFolderDialog() {
    document.getElementById('folderModal').classList.remove('active');
}

async function confirmNewFolder() {
    const name = document.getElementById('folderName').value.trim();
    if (!name) { showToast('请输入文件夹名称', 'error'); return; }
    const folderPath = state.currentPath === '.' ? name : `${state.currentPath}/${name}`;
    try {
        await API.post('/api/folder', { path: folderPath });
        showToast('文件夹已创建', 'success');
        closeFolderDialog();
        loadFileList();
    } catch (e) {
        showToast(`创建失败: ${e.message}`, 'error');
    }
}

// ===== 上传 =====
function setupDragDrop() {
    const zone = document.getElementById('dropZone');
    zone.addEventListener('dragover', (e) => { e.preventDefault(); zone.classList.add('drag-over'); });
    zone.addEventListener('dragleave', () => zone.classList.remove('drag-over'));
    zone.addEventListener('drop', (e) => {
        e.preventDefault();
        zone.classList.remove('drag-over');
        if (e.dataTransfer.files.length > 0) uploadFiles(e.dataTransfer.files, e.dataTransfer.items);
    });
    zone.addEventListener('click', () => document.getElementById('fileInput').click());
}

function setupFileInputs() {
    document.getElementById('fileInput').addEventListener('change', (e) => {
        if (e.target.files.length > 0) uploadFiles(e.target.files);
        e.target.value = '';
    });
    document.getElementById('folderInput').addEventListener('change', (e) => {
        if (e.target.files.length > 0) uploadFiles(e.target.files);
        e.target.value = '';
    });
}

async function uploadFiles(fileList) {
    const formData = new FormData();
    let fileCount = 0;

    for (const file of fileList) {
        let relativePath = file.webkitRelativePath || file.name;
        formData.append('files', file);
        formData.append('paths', relativePath);
        fileCount++;
    }

    if (fileCount === 0) return;

    showProgress(true, 0);

    try {
        const xhr = new XMLHttpRequest();
        xhr.open('POST', '/api/upload');

        xhr.upload.onprogress = (e) => {
            if (e.lengthComputable) {
                const pct = Math.round((e.loaded / e.total) * 100);
                showProgress(true, pct);
            }
        };

        await new Promise((resolve, reject) => {
            xhr.onload = () => {
                if (xhr.status >= 200 && xhr.status < 300) resolve();
                else reject(new Error(xhr.responseText || `HTTP ${xhr.status}`));
            };
            xhr.onerror = () => reject(new Error('网络错误'));
            xhr.send(formData);
        });

        showProgress(false);
        showToast(`成功上传 ${fileCount} 个文件`, 'success');
        loadFileList();
    } catch (e) {
        showProgress(false);
        showToast(`上传失败: ${e.message}`, 'error');
    }
}

function showProgress(visible, pct) {
    const el = document.getElementById('uploadProgress');
    const fill = document.getElementById('progressFill');
    const text = document.getElementById('progressText');
    if (!visible) {
        el.style.display = 'none';
        fill.style.width = '0%';
        return;
    }
    el.style.display = 'block';
    fill.style.width = `${pct}%`;
    text.textContent = `${pct}%`;
}

// ===== 预览（含图片画廊） =====
async function previewFile(path) {
    // 检查是否为目录（用于双击进入目录时）
    try {
        const data = await API.get(`/api/list?path=${encodeURIComponent(path)}`);
        if (data.files) return;
    } catch (e) { /* expected */ }

    state.previewFile = path;
    const name = path.split('/').pop() || path.split('\\').pop() || path;
    const ext = name.split('.').pop().toLowerCase();
    document.getElementById('previewTitle').textContent = `👁️ ${name}`;
    document.getElementById('previewModal').classList.add('active');

    // 查找图片列表中的索引
    state.previewIndex = state.imageList.indexOf(path);

    const body = document.getElementById('previewBody');
    const previewUrl = `/api/preview?path=${encodeURIComponent(path)}`;
    const escapedPath = escapeHtml(path);
    const escapedName = escapeHtml(name);

    // 图片预览（画廊模式）
    if (IMAGE_EXTS.has(ext)) {
        body.innerHTML = `<div class="gallery-wrapper">
            <img class="preview-image" src="${previewUrl}" alt="${escapedName}"
                 onerror="this.parentElement.innerHTML='<p>⚠️ 图片加载失败</p>'">
            <div class="gallery-counter" id="galleryCounter"></div>
        </div>`;
        document.getElementById('previewTitle').textContent = `🖼️ ${name}`;
        updateGalleryButtons();
    }
    // 视频预览
    else if (VIDEO_EXTS.has(ext)) {
        const videoType = ext === 'mkv' ? 'mp4' : ext;
        body.innerHTML = `<video class="preview-video" controls>
            <source src="${previewUrl}" type="video/${videoType}">
            您的浏览器不支持视频播放
        </video>`;
    }
    // 音频预览
    else if (AUDIO_EXTS.has(ext)) {
        body.innerHTML = `<div style="text-align:center;padding:40px 0">
            <p style="font-size:48px;margin-bottom:16px">🎵</p>
            <audio class="preview-audio" controls src="${previewUrl}">您的浏览器不支持音频播放</audio>
        </div>`;
    }
    // PDF 预览
    else if (ext === 'pdf') {
        body.innerHTML = `<iframe class="preview-pdf" src="${previewUrl}"></iframe>`;
    }
    // 压缩包预览
    else if (ARCHIVE_EXTS.has(ext)) {
        body.innerHTML = '<div class="loading">📦 正在读取压缩包...</div>';
        try {
            const res = await fetch(`/api/archive?path=${encodeURIComponent(path)}`);
            const data = await res.json();
            if (data.supported && data.entries && data.entries.length > 0) {
                let html = `<div class="archive-header">📦 <strong>${escapeHtml(data.name)}</strong> — ${escapeHtml(data.message)}</div>`;
                html += `<div class="archive-table-wrapper"><table class="archive-table">
                    <thead><tr><th></th><th>名称</th><th>大小</th><th>修改时间</th></tr></thead><tbody>`;
                data.entries.forEach(e => {
                    const icon = e.is_dir ? '📁' : getExtIcon(e.name);
                    const size = e.is_dir ? '—' : formatSize(e.size);
                    html += `<tr>
                        <td class="archive-icon">${icon}</td>
                        <td class="archive-name">${escapeHtml(e.name)}</td>
                        <td class="archive-size">${size}</td>
                        <td class="archive-date">${e.mod_time || '—'}</td>
                    </tr>`;
                });
                html += `</tbody></table></div>`;
                body.innerHTML = html;
            } else {
                body.innerHTML = `
                    <div class="loading" style="padding:30px">
                        <p style="font-size:48px;margin-bottom:12px">📦</p>
                        <p>${escapeHtml(data.message || '压缩包预览不可用')}</p>
                        <button class="btn btn-primary" onclick="downloadFile('${escapedPath}')" style="margin-top:16px">⬇️ 下载压缩包</button>
                    </div>`;
            }
        } catch (e) {
            body.innerHTML = `
                <div class="loading" style="padding:30px">
                    <p style="font-size:48px;margin-bottom:12px">📦</p>
                    <p>无法读取压缩包</p>
                    <button class="btn btn-primary" onclick="downloadFile('${escapedPath}')" style="margin-top:16px">⬇️ 下载压缩包</button>
                </div>`;
        }
    }
    // 文本/代码预览（highlight.js 语法高亮）
    else if (TEXT_EXTS.has(ext) || ext === '') {
        try {
            const res = await fetch(previewUrl);
            const text = await res.text();
            const langClass = LANG_MAP[ext] || '';
            body.innerHTML = `<div style="position:relative">
                <button class="btn btn-sm" onclick="copyPreviewCode()" style="position:absolute;top:8px;right:8px;z-index:1;background:var(--bg-card);font-size:12px;padding:4px 8px">📋 复制代码</button>
                <pre class="preview-text" style="padding-top:36px"><code class="language-${langClass}">${escapeHtml(text)}</code></pre>
            </div>`;
            if (typeof hljs !== 'undefined') {
                body.querySelectorAll('pre code').forEach(hljs.highlightElement);
            }
        } catch (e) {
            body.innerHTML = '<p>⚠️ 文本内容加载失败</p>';
        }
    }
    // 不支持的格式
    else {
        body.innerHTML = `
            <div class="loading">
                <p style="font-size:40px;margin-bottom:12px">📄</p>
                <p>此文件类型不支持预览</p>
                <button class="btn btn-primary" onclick="downloadFile('${escapedPath}')" style="margin-top:12px">⬇️ 下载文件</button>
            </div>`;
    }
}

// ===== 图片画廊导航 =====
function updateGalleryButtons() {
    const prevBtn = document.getElementById('prevBtn');
    const nextBtn = document.getElementById('nextBtn');
    const counter = document.getElementById('galleryCounter');
    if (!prevBtn || !nextBtn) return;

    const isImage = state.previewIndex >= 0 && state.imageList.length > 0;
    if (isImage) {
        prevBtn.style.display = '';
        nextBtn.style.display = '';
        prevBtn.disabled = state.previewIndex <= 0;
        nextBtn.disabled = state.previewIndex >= state.imageList.length - 1;
        if (counter) {
            counter.textContent = `${state.previewIndex + 1} / ${state.imageList.length}`;
        }
    } else {
        prevBtn.style.display = 'none';
        nextBtn.style.display = 'none';
        if (counter) counter.textContent = '';
    }
}

function prevImage() {
    if (state.previewIndex > 0 && state.imageList.length > 0) {
        previewFile(state.imageList[state.previewIndex - 1]);
    }
}

function nextImage() {
    if (state.previewIndex < state.imageList.length - 1) {
        previewFile(state.imageList[state.previewIndex + 1]);
    }
}

function closePreview() {
    document.getElementById('previewModal').classList.remove('active');
    state.previewFile = null;
    state.previewIndex = -1;
}

function downloadCurrent() {
    if (state.previewFile) downloadFile(state.previewFile);
}

function shareCurrent() {
    if (state.previewFile) {
        closePreview();
        shareDialog(state.previewFile, false);
    }
}

function copyPreviewCode() {
    const code = document.querySelector('.preview-text code');
    if (code) copyToClipboard(code.textContent);
}

// ===== 分享链接 =====
function shareDialog(path, isDir) {
    document.getElementById('sharePath').value = path;
    document.getElementById('sharePassword').value = '';
    document.getElementById('shareExpiration').value = '';
    document.getElementById('shareResult').style.display = 'none';
    document.getElementById('shareModal').classList.add('active');
}

function closeShareDialog() {
    document.getElementById('shareModal').classList.remove('active');
}

async function generateShareLink() {
    const path = document.getElementById('sharePath').value;
    const password = document.getElementById('sharePassword').value;
    const expiresIn = document.getElementById('shareExpiration').value;

    try {
        const link = await API.post('/api/share', { path, password, expires_in: expiresIn });
        const shareUrl = `${getShareBaseUrl()}/share/${link.token}`;
        document.getElementById('shareUrl').value = shareUrl;
        document.getElementById('shareResult').style.display = 'block';
        showToast('分享链接已创建！', 'success');
        loadShareLinks();
    } catch (e) {
        showToast(`创建分享链接失败: ${e.message}`, 'error');
    }
}

// 获取分享链接的基本 URL（优先使用局域网 IP）
function getShareBaseUrl() {
    if (state.serverInfo && state.serverInfo.local_ip) {
        return `http://${state.serverInfo.local_ip}:${state.serverInfo.port}`;
    }
    return window.location.origin;
}

// 通用剪贴板复制（带 fallback）
function copyToClipboard(text) {
    // 优先使用现代 Clipboard API
    if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(() => {
            showToast('链接已复制到剪贴板！', 'success');
        }).catch(() => {
            // fallback: 创建临时 textarea
            fallbackCopy(text);
        });
    } else {
        fallbackCopy(text);
    }
}

function fallbackCopy(text) {
    const textarea = document.createElement('textarea');
    textarea.value = text;
    textarea.style.position = 'fixed';
    textarea.style.opacity = '0';
    textarea.style.left = '-9999px';
    document.body.appendChild(textarea);
    textarea.select();
    try {
        document.execCommand('copy');
        showToast('链接已复制！', 'success');
    } catch (e) {
        showToast('复制失败，请手动选中链接复制', 'error');
    }
    document.body.removeChild(textarea);
}

function copyShareUrl() {
    const input = document.getElementById('shareUrl');
    copyToClipboard(input.value);
}

async function loadShareLinks() {
    try {
        const data = await API.get('/api/share');
        state.shareLinks = data.links || [];
        renderShareLinks();
    } catch (e) { /* silently fail */ }
}

function renderShareLinks() {
    const el = document.getElementById('shareLinks');
    if (state.shareLinks.length === 0) {
        el.innerHTML = '<p class="empty-hint">暂无分享链接</p>';
        return;
    }
    el.innerHTML = state.shareLinks.map(link => `
        <div class="share-link-item">
            <div class="link-path" title="${escapeHtml(link.path)}">${link.is_dir ? '📁' : '📄'} ${escapeHtml(link.path)}</div>
            <div class="link-actions">
                <button class="btn btn-sm" onclick="copyShareLink('${link.token}')" title="复制链接">📋</button>
                <button class="btn btn-sm btn-danger" onclick="deleteShareLink('${link.token}')" title="删除">✕</button>
            </div>
        </div>
    `).join('');
}

function copyShareLink(token) {
    const url = `${getShareBaseUrl()}/share/${token}`;
    copyToClipboard(url);
}

async function deleteShareLink(token) {
    try {
        await API.delete(`/api/share?token=${token}`, { token });
        showToast('分享链接已删除', 'success');
        loadShareLinks();
    } catch (e) {
        showToast(`删除失败: ${e.message}`, 'error');
    }
}

// ===== 右键菜单 =====
function showContextMenu(e, file) {
    removeContextMenu();
    const menu = document.createElement('div');
    menu.className = 'context-menu';
    menu.innerHTML = `
        ${file.is_dir
            ? `<div class="context-menu-item" onclick="navigateTo({name:'${escapeHtml(file.name)}',path:'${file.path}',is_dir:true}); removeContextMenu()">📂 打开</div>`
            : (state.permissions.preview ? `<div class="context-menu-item" onclick="previewFile('${file.path}'); removeContextMenu()">👁️ 预览</div>` : ``)}
        ${state.permissions.download ? `<div class="context-menu-item" onclick="downloadFile('${file.path}'); removeContextMenu()">⬇️ 下载</div>` : ``}
        ${state.permissions.share ? `<div class="context-menu-item" onclick="shareDialog('${file.path}', ${file.is_dir}); removeContextMenu()">🔗 分享</div>` : ``}
        ${state.permissions.delete ? `<hr style="border:none;border-top:1px solid var(--border);margin:4px 0"><div class="context-menu-item danger" onclick="deleteFile('${file.path}'); removeContextMenu()">🗑️ 删除</div>` : ``}
    `;
    menu.style.left = `${e.clientX}px`;
    menu.style.top = `${e.clientY}px`;
    document.body.appendChild(menu);
    setTimeout(() => document.addEventListener('click', removeContextMenu, { once: true }), 0);
}

function removeContextMenu() {
    document.querySelectorAll('.context-menu').forEach(m => m.remove());
}

// ===== 主题 =====
function toggleTheme() {
    const current = document.documentElement.getAttribute('data-theme');
    const next = current === 'dark' ? '' : 'dark';
    document.documentElement.setAttribute('data-theme', next);
    localStorage.setItem('theme', next || 'light');
}

function loadTheme() {
    const saved = localStorage.getItem('theme');
    if (saved) document.documentElement.setAttribute('data-theme', saved);
}

// ===== 关于 =====
function showAbout() {
    const info = state.serverInfo;
    const msg = `📁 局域网文件共享 v${info ? info.version : '1.0.0'}\n\n` +
        `📍 服务地址: ${info ? `${info.local_ip}:${info.port}` : '未知'}\n` +
        `📂 最大上传: ${info ? `${info.max_upload}MB` : '500MB'}\n` +
        `🔒 密码保护: ${info && info.has_auth ? '已启用' : '未启用（开放访问）'}\n\n` +
        `拖拽文件到窗口即可共享！\n同局域网内所有设备均可通过浏览器访问。`;
    alert(msg);
}

// ===== 帮助页面 =====
function showHelp() {
    const modal = document.getElementById('helpModal');
    if (modal) modal.classList.add('active');
}

function closeHelp() {
    const modal = document.getElementById('helpModal');
    if (modal) modal.classList.remove('active');
}

// Apply UI restrictions based on current permission state
function applyUIPermissions() {
    const p = state.permissions;

    // New folder button
    const newBtn = document.getElementById('newFolderBtn');
    if (newBtn) {
        newBtn.style.display = p.upload ? '' : 'none';
    }

    // Upload section visibility
    const uploadSection = document.querySelector('.upload-section');
    if (uploadSection) {
        uploadSection.style.display = p.upload ? '' : 'none';
    }

    // Toolbar upload button visibility
    const uploadBtn = document.getElementById('uploadBtn');
    if (uploadBtn) {
        uploadBtn.style.display = p.upload ? '' : 'none';
    }

    // Share links section visibility
    const shareSection = document.querySelector('.sidebar-section:has(#shareLinks)');
    if (shareSection) {
        shareSection.style.display = (p.share || p.isAdmin) ? '' : 'none';
    }

    // Settings button visibility (admin only when auth enabled)
    const settingsBtn = document.querySelector('[onclick="showSettings()"]');
    if (settingsBtn) {
        settingsBtn.style.display = p.isAdmin ? '' : 'none';
    }

    // Re-render the file list to update action buttons
    renderFiles();
    renderShareLinks();
}

// Override API.put since it doesn't exist yet
API.put = async function(path, body) {
    const res = await fetch(path, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
    });
    if (!res.ok) {
        const err = await res.text();
        throw new Error(err || `HTTP ${res.status}`);
    }
    return res.json();
};

// ===== 快捷键 =====
function setupKeyboard() {
    document.addEventListener('keydown', (e) => {
        // Escape 关闭弹窗
        if (e.key === 'Escape') {
            closePreview();
            closeShareDialog();
            closeFolderDialog();
            closeHelp();
            removeContextMenu();
        }
        // 图片画廊 ← →
        if (e.key === 'ArrowLeft' && state.previewFile && state.previewIndex >= 0) {
            e.preventDefault();
            prevImage();
        }
        if (e.key === 'ArrowRight' && state.previewFile && state.previewIndex >= 0) {
            e.preventDefault();
            nextImage();
        }
        // Ctrl+R 刷新
        if ((e.ctrlKey || e.metaKey) && e.key === 'r') {
            if (document.activeElement?.tagName !== 'INPUT' && document.activeElement?.tagName !== 'TEXTAREA') {
                e.preventDefault();
                refreshList();
            }
        }
        // Ctrl+A 全选
        if ((e.ctrlKey || e.metaKey) && e.key === 'a') {
            if (document.activeElement?.tagName !== 'INPUT' && document.activeElement?.tagName !== 'TEXTAREA') {
                e.preventDefault();
                state.files.forEach(f => state.selectedFiles.add(f.path));
                updateSelectionBar();
                renderFiles();
            }
        }
        // Delete 删除选中
        if (e.key === 'Delete' && state.selectedFiles.size > 0) {
            if (document.activeElement?.tagName !== 'INPUT' && document.activeElement?.tagName !== 'TEXTAREA') {
                e.preventDefault();
                batchDelete();
            }
        }
    });
}

// ===== 工具函数 =====
function getFileIcon(f) {
    if (f.is_dir) return '📁';
    return getExtIcon(f.name);
}

function getExtIcon(filename) {
    const ext = filename.split('.').pop().toLowerCase();
    return EXT_ICONS[ext] || '📄';
}

function formatSize(bytes) {
    if (bytes === 0) return '0 B';
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    return (bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0) + ' ' + units[i];
}

// 优化的 HTML 转义函数（复用 DOM 元素）
function escapeHtml(text) {
    _escapeDiv.textContent = text;
    return _escapeDiv.innerHTML;
}

function showToast(message, type = 'info') {
    const container = document.getElementById('toastContainer');
    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    toast.textContent = message;
    container.appendChild(toast);
    setTimeout(() => {
        toast.style.opacity = '0';
        toast.style.transition = 'opacity 0.3s';
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}
