package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileInfo represents a file or directory entry in the listing.
type FileInfo struct {
	Name    string `json:"name"`
	Path    string `json:"path"` // relative path
	Size    int64  `json:"size"`
	IsDir   bool   `json:"is_dir"`
	ModTime string `json:"mod_time"`
}

// validatePath sanitizes a relative path and validates it's within the shared directory.
// Returns the absolute path and info if valid, or an error response if not.
func (s *FileServer) validatePath(w http.ResponseWriter, relPath string) (string, os.FileInfo, bool) {
	cleanPath := sanitizePath(relPath)
	if cleanPath == "" {
		http.Error(w, "Path is required", http.StatusBadRequest)
		return "", nil, false
	}

	fullPath := filepath.Join(s.config.SharedDir, cleanPath)
	absShared, _ := filepath.Abs(s.config.SharedDir)
	absFull, _ := filepath.Abs(fullPath)

	if !strings.HasPrefix(absFull, absShared) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return "", nil, false
	}

	info, err := os.Stat(absFull)
	if err != nil {
		http.Error(w, "Path not found", http.StatusNotFound)
		return "", nil, false
	}

	return absFull, info, true
}

// mimeTypes maps file extensions to MIME types (cached, not recreated per request).
var mimeTypes = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".svg":  "image/svg+xml",
	".webp": "image/webp",
	".bmp":  "image/bmp",
	".ico":  "image/x-icon",
	".mp4":  "video/mp4",
	".webm": "video/webm",
	".avi":  "video/x-msvideo",
	".mov":  "video/quicktime",
	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
	".ogg":  "audio/ogg",
	".flac": "audio/flac",
	".pdf":  "application/pdf",
	".txt":  "text/plain; charset=utf-8",
	".md":   "text/markdown; charset=utf-8",
	".html": "text/html; charset=utf-8",
	".htm":  "text/html; charset=utf-8",
	".css":  "text/css; charset=utf-8",
	".js":   "application/javascript; charset=utf-8",
	".json": "application/json; charset=utf-8",
	".xml":  "application/xml; charset=utf-8",
	".yaml": "text/plain; charset=utf-8",
	".yml":  "text/plain; charset=utf-8",
	".go":   "text/plain; charset=utf-8",
	".py":   "text/plain; charset=utf-8",
	".java": "text/plain; charset=utf-8",
	".c":    "text/plain; charset=utf-8",
	".cpp":  "text/plain; charset=utf-8",
	".h":    "text/plain; charset=utf-8",
	".sh":   "text/plain; charset=utf-8",
	".bat":  "text/plain; charset=utf-8",
	".ps1":  "text/plain; charset=utf-8",
	".log":  "text/plain; charset=utf-8",
	".csv":  "text/csv; charset=utf-8",
	".toml": "text/plain; charset=utf-8",
	".conf": "text/plain; charset=utf-8",
	".ini":  "text/plain; charset=utf-8",
}

// ListHandler returns the directory listing.
func (s *FileServer) ListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !s.HasPermission(s.isAuthenticated(r), PermBrowse) {
		http.Error(w, "Forbidden: browsing not allowed", http.StatusForbidden)
		return
	}

	relPath := r.URL.Query().Get("path")
	if relPath == "" {
		relPath = "."
	}
	relPath = sanitizePath(relPath)

	fullPath := filepath.Join(s.config.SharedDir, relPath)

	// Security: ensure we don't escape the shared dir
	absShared, _ := filepath.Abs(s.config.SharedDir)
	absFull, _ := filepath.Abs(fullPath)
	if !strings.HasPrefix(absFull, absShared) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	info, err := os.Stat(absFull)
	if err != nil {
		http.Error(w, "Path not found", http.StatusNotFound)
		return
	}

	// If it's a single file, return redirect for download
	if !info.IsDir() {
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, info.Name()))
		http.ServeFile(w, r, absFull)
		return
	}

	entries, err := os.ReadDir(absFull)
	if err != nil {
		http.Error(w, "Failed to read directory", http.StatusInternalServerError)
		return
	}

	var files = make([]FileInfo, 0, len(entries))
	for _, entry := range entries {
		fi, err := entry.Info()
		if err != nil {
			continue
		}
		entryRelPath := filepath.Join(relPath, entry.Name())
		if relPath == "." {
			entryRelPath = entry.Name()
		}
		files = append(files, FileInfo{
			Name:    entry.Name(),
			Path:    entryRelPath,
			Size:    fi.Size(),
			IsDir:   entry.IsDir(),
			ModTime: fi.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	// Sort: directories first, then by name
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	resp := map[string]interface{}{
		"path":  relPath,
		"files": files,
	}
	writeJSON(w, http.StatusOK, resp)
}

// DownloadHandler handles downloading a file or folder as zip.
func (s *FileServer) DownloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !s.HasPermission(s.isAuthenticated(r), PermDownload) {
		http.Error(w, "Forbidden: download not allowed", http.StatusForbidden)
		return
	}

	relPath := r.URL.Query().Get("path")
	absFull, info, ok := s.validatePath(w, relPath)
	if !ok {
		return
	}

	if info.IsDir() {
		// Zip and download the folder
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.zip"`, info.Name()))
		if err := zipDirectory(absFull, w); err != nil {
			http.Error(w, "Failed to create zip", http.StatusInternalServerError)
		}
		return
	}

	// Single file download
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, info.Name()))
	http.ServeFile(w, r, absFull)
}

// PreviewHandler returns file content for preview (text, images served directly).
func (s *FileServer) PreviewHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !s.HasPermission(s.isAuthenticated(r), PermPreview) {
		http.Error(w, "Forbidden: preview not allowed", http.StatusForbidden)
		return
	}

	relPath := r.URL.Query().Get("path")
	absFull, info, ok := s.validatePath(w, relPath)
	if !ok {
		return
	}

	if info.IsDir() {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Set content type based on extension
	ext := strings.ToLower(filepath.Ext(info.Name()))

	if mime, ok := mimeTypes[ext]; ok {
		w.Header().Set("Content-Type", mime)
	} else {
		// Force download for unknown types
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, info.Name()))
	}

	http.ServeFile(w, r, absFull)
}

// DeleteHandler handles file/folder deletion.
func (s *FileServer) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !s.HasPermission(s.isAuthenticated(r), PermDelete) {
		http.Error(w, "Forbidden: delete not allowed", http.StatusForbidden)
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	absFull, _, ok := s.validatePath(w, req.Path)
	if !ok {
		return
	}

	if err := os.RemoveAll(absFull); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// CreateFolderHandler creates a new folder.
func (s *FileServer) CreateFolderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !s.HasPermission(s.isAuthenticated(r), PermUpload) {
		http.Error(w, "Forbidden: upload not allowed", http.StatusForbidden)
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	relPath := sanitizePath(req.Path)
	if relPath == "" {
		http.Error(w, "Path is required", http.StatusBadRequest)
		return
	}

	// For folder creation, we need to validate the parent path
	parentPath := filepath.Dir(relPath)
	if parentPath != "." {
		if _, _, ok := s.validatePath(w, parentPath); !ok {
			return
		}
	}

	fullPath := filepath.Join(s.config.SharedDir, relPath)
	absFull, _ := filepath.Abs(fullPath)

	if err := os.MkdirAll(absFull, 0755); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create folder: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}
