package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"lan-file-share/models"
)

// FileServer is the main HTTP file sharing server.
type FileServer struct {
	config     *Config
	shareStore *models.ShareStore
	embedFS    embed.FS
	paused     atomic.Bool // service paused (non-admin requests blocked), thread-safe
	indexTmpl  *template.Template // cached index template
}

// New creates a new FileServer.
func New(cfg *Config, efs embed.FS) *FileServer {
	return &FileServer{
		config:     cfg,
		shareStore: models.NewShareStore(),
		embedFS:    efs,
	}
}

// Start starts the HTTP server.
func (s *FileServer) Start() error {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/list", s.ListHandler)
	mux.HandleFunc("/api/upload", s.UploadHandler)
	mux.HandleFunc("/api/download", s.DownloadHandler)
	mux.HandleFunc("/api/preview", s.PreviewHandler)
	mux.HandleFunc("/api/delete", s.DeleteHandler)
	mux.HandleFunc("/api/folder", s.CreateFolderHandler)
	mux.HandleFunc("/api/share", s.ShareHandler)
	mux.HandleFunc("/api/archive", s.ArchivePreviewHandler)
	mux.HandleFunc("/api/info", s.InfoHandler)
	mux.HandleFunc("/api/admin/config", s.AdminHandler)
	mux.HandleFunc("/api/admin/status", s.AdminStatusHandler)
	mux.HandleFunc("/api/admin/toggle", s.AdminToggleHandler)
	mux.HandleFunc("/api/permissions", s.PermissionsHandler)

	// Shared file access via token
	mux.HandleFunc("/share/", s.AccessSharedHandler)

	// Serve embedded static files
	staticFS, err := fs.Sub(s.embedFS, "static")
	if err != nil {
		return fmt.Errorf("failed to get static sub-fs: %w", err)
	}
	fileServer := http.FileServer(http.FS(staticFS))

	// Handle /static/ prefix - need StripPrefix because FileServer sees full URL path
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	// Main page
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			s.renderIndex(w, r)
			return
		}
		if r.URL.Path == "/admin.html" {
			s.renderAdmin(w, r)
			return
		}
		// Favicon
		if r.URL.Path == "/favicon.ico" {
			http.NotFound(w, r)
			return
		}
		// Try serving from shared directory
		sharedPath := filepath.Join(s.config.SharedDir, r.URL.Path)
		http.ServeFile(w, r, sharedPath)
	})

	// Wrap with middleware (CORS + logging only, auth is handled per-handler)
	var h http.Handler = CORS(mux)
	h = Logging(h)
	// Wrap with pause check — when paused, only admin routes work
	h = s.pauseMiddleware(h)

	// Start cleanup goroutine for expired share links
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			s.shareStore.Cleanup()
		}
	}()

	addr := fmt.Sprintf(":%d", s.config.Port)
	fmt.Printf("Server starting on http://0.0.0.0%s\n", addr)
	fmt.Printf("Local access: http://localhost%s\n", addr)
	fmt.Printf("LAN access:   http://%s%s\n", LocalIP(), addr)
	if s.config.AuthUser != "" {
		fmt.Println("Authentication: enabled")
	} else {
		fmt.Println("Authentication: disabled (open to LAN)")
	}
	fmt.Printf("Shared directory: %s\n", s.config.SharedDir)

	return http.ListenAndServe(addr, h)
}

// InfoHandler returns server info for the UI.
func (s *FileServer) InfoHandler(w http.ResponseWriter, r *http.Request) {
	// Include guest permissions for non-admin users
	var gp *GuestPermissions
	if s.config.AuthUser != "" {
		gp = s.config.GuestPermissions
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"server":            "LAN File Share",
		"version":           "1.0.0",
		"local_ip":          LocalIP(),
		"port":              s.config.Port,
		"max_upload":        s.config.MaxUpload,
		"has_auth":          s.config.AuthUser != "",
		"guest_permissions": gp,
		"perm_categories":   AllCategories(),
	})
}

// pauseMiddleware blocks non-admin requests when the service is paused.
func (s *FileServer) pauseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.paused.Load() {
			// Always allow admin routes and static assets
			if strings.HasPrefix(r.URL.Path, "/api/admin/") || r.URL.Path == "/admin.html" ||
				strings.HasPrefix(r.URL.Path, "/static/") {
				next.ServeHTTP(w, r)
				return
			}
			// Block all other requests
			if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/" {
				w.Header().Set("Content-Type", "application/json")
				writeJSON(w, http.StatusServiceUnavailable, map[string]interface{}{
					"error":   "paused",
					"message": "文件共享服务已暂停，请联系管理员",
				})
				return
			}
			// For other page requests, show a simple HTML page
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `<!DOCTYPE html><html><head><meta charset="utf-8"><title>服务已暂停</title>
			<style>body{font-family:sans-serif;text-align:center;padding:80px 20px;background:#1a1b2e;color:#e4e6f0}
			h1{font-size:60px;margin-bottom:10px}p{font-size:18px;color:#8f91b0}
			</style></head><body><h1>⏸️</h1><h2>文件共享服务已暂停</h2><p>请联系管理员恢复服务</p></body></html>`)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// AdminToggleHandler toggles the service pause state.
func (s *FileServer) AdminToggleHandler(w http.ResponseWriter, r *http.Request) {
	if !s.isAdminRequest(r) {
		AuthChallenge(w)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Paused *bool `json:"paused"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if req.Paused != nil {
		s.paused.Store(*req.Paused)
	}
	paused := s.paused.Load()
	state := "running"
	if paused {
		state = "paused"
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"paused":  paused,
		"state":   state,
	})
}

// renderAdmin renders the admin management page.
func (s *FileServer) renderAdmin(w http.ResponseWriter, r *http.Request) {
	content, err := fs.ReadFile(s.embedFS, "static/admin.html")
	if err != nil {
		http.Error(w, "Admin page not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(content)
}

func (s *FileServer) renderIndex(w http.ResponseWriter, r *http.Request) {
	// Cache the template on first use
	if s.indexTmpl == nil {
		tmplContent, err := fs.ReadFile(s.embedFS, "static/index.html")
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		tmpl, err := template.New("index").Parse(string(tmplContent))
		if err != nil {
			http.Error(w, "Template error", http.StatusInternalServerError)
			return
		}
		s.indexTmpl = tmpl
	}

	data := map[string]interface{}{
		"ServerName":  "LAN File Share",
		"LocalIP":     LocalIP(),
		"Port":        s.config.Port,
		"MaxUploadMB": s.config.MaxUpload,
		"HasAuth":     s.config.AuthUser != "",
	}
	s.indexTmpl.Execute(w, data)
}
