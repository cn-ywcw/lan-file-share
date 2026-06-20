package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// UploadHandler handles file and folder uploads.
func (s *FileServer) UploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !s.CheckPermission(w, r, PermUpload) {
		return
	}

	// Limit upload size
	r.Body = http.MaxBytesReader(w, r.Body, s.config.MaxUpload*1024*1024)

	contentType := r.Header.Get("Content-Type")

	var uploaded []UploadedFile

	if strings.HasPrefix(contentType, "multipart/form-data") {
		// Multipart upload (supports folder structure via relative paths)
		if err := r.ParseMultipartForm(s.config.MaxUpload * 1024 * 1024); err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse form: %v", err), http.StatusBadRequest)
			return
		}

		files := r.MultipartForm.File["files"]
		for _, fh := range files {
			// Get the relative path from the form field "paths" or use filename
			relativePath := fh.Filename
			if paths, ok := r.MultipartForm.Value["paths"]; ok && len(paths) > 0 {
				// Server-side, we can match by index
			}

			src, err := fh.Open()
			if err != nil {
				continue
			}

			// Sanitize path: prevent directory traversal
			cleanPath := sanitizePath(relativePath)
			if cleanPath == "" {
				src.Close()
				continue
			}

			destPath := filepath.Join(s.config.SharedDir, cleanPath)
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				src.Close()
				continue
			}

			dst, err := os.Create(destPath)
			if err != nil {
				src.Close()
				continue
			}

			written, _ := io.Copy(dst, src)
			dst.Close()
			src.Close()
			uploaded = append(uploaded, UploadedFile{
				Name: fh.Filename,
				Path: cleanPath,
				Size: written,
			})
		}
	} else if strings.HasPrefix(contentType, "application/octet-stream") || strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		// Single file upload via body (alternative method)
		filename := r.URL.Query().Get("filename")
		if filename == "" {
			filename = "unnamed"
		}
		cleanPath := sanitizePath(filename)
		if cleanPath == "" {
			http.Error(w, "Invalid filename", http.StatusBadRequest)
			return
		}
		destPath := filepath.Join(s.config.SharedDir, cleanPath)
		dst, err := os.Create(destPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer dst.Close()
		written, _ := io.Copy(dst, r.Body)
		uploaded = append(uploaded, UploadedFile{
			Name: filename,
			Path: cleanPath,
			Size: written,
		})
	} else {
		http.Error(w, "Unsupported Content-Type", http.StatusBadRequest)
		return
	}

	resp := map[string]interface{}{
		"success": true,
		"files":   uploaded,
	}
	writeJSON(w, http.StatusOK, resp)
}

// sanitizePath prevents directory traversal by cleaning the path.
func sanitizePath(path string) string {
	// Convert to forward slashes, clean, remove leading ".."
	clean := filepath.Clean(strings.ReplaceAll(path, "\\", "/"))
	// Remove leading slash
	clean = strings.TrimPrefix(clean, "/")
	// Remove any remaining ".." components
	parts := strings.Split(clean, string(filepath.Separator))
	var result []string
	for _, p := range parts {
		if p == ".." {
			if len(result) > 0 {
				result = result[:len(result)-1]
			}
			continue
		}
		if p != "" && p != "." {
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return "."
	}
	return filepath.Join(result...)
}

// UploadedFile represents a successfully uploaded file.
type UploadedFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
