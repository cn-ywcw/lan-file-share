package server

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// ShareHandler handles share link CRUD operations.
func (s *FileServer) ShareHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		if !s.CheckPermission(w, r, PermShare) {
			return
		}
		s.createShareLink(w, r)
	case http.MethodGet:
		if !s.CheckPermission(w, r, PermBrowse) {
			return
		}
		s.listShareLinks(w, r)
	case http.MethodDelete:
		if !s.CheckPermission(w, r, PermShare) {
			return
		}
		s.deleteShareLink(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// AccessSharedHandler handles accessing a shared file via token.
func (s *FileServer) AccessSharedHandler(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.URL.Path, "/share/")
	token = strings.Split(token, "?")[0]

	if token == "" {
		http.Error(w, "Token is required", http.StatusBadRequest)
		return
	}

	link, ok := s.shareStore.Get(token)
	if !ok {
		http.Error(w, "Share link not found", http.StatusNotFound)
		return
	}

	// Check expiration
	if !link.ExpiresAt.IsZero() && time.Now().After(link.ExpiresAt) {
		s.shareStore.Delete(token)
		http.Error(w, "Share link has expired", http.StatusGone)
		return
	}

	// Check password if required
	if link.Password != "" {
		password := r.URL.Query().Get("password")
		if password == "" {
			password = r.Header.Get("X-Share-Password")
		}

		if password == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
				"error":    "password_required",
				"message":  "This share is password protected",
				"requires": "password",
			})
			return
		}

		pHash := sha256.Sum256([]byte(password))
		linkHash := sha256.Sum256([]byte(link.Password))
		if subtle.ConstantTimeCompare(pHash[:], linkHash[:]) != 1 {
			writeJSON(w, http.StatusForbidden, map[string]interface{}{
				"error":   "invalid_password",
				"message": "Invalid password",
			})
			return
		}
	}

	// Serve the file/directory
	fullPath := filepath.Join(s.config.SharedDir, link.Path)
	if link.IsDir {
		http.Redirect(w, r, "/?path="+link.Path, http.StatusSeeOther)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=\""+filepath.Base(link.Path)+"\"")
	http.ServeFile(w, r, fullPath)
}

func (s *FileServer) createShareLink(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path      string `json:"path"`
		Password  string `json:"password,omitempty"`
		ExpiresIn string `json:"expires_in,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	absPath, stat, ok := s.validatePath(w, req.Path)
	if !ok {
		return
	}

	// Parse expiration
	var expiresIn time.Duration
	switch {
	case strings.HasSuffix(req.ExpiresIn, "d"):
		var days int
		if _, err := fmt.Sscanf(req.ExpiresIn, "%dd", &days); err == nil && days > 0 {
			expiresIn = time.Duration(days) * 24 * time.Hour
		}
	case strings.HasSuffix(req.ExpiresIn, "h"):
		d, err := time.ParseDuration(req.ExpiresIn)
		if err == nil {
			expiresIn = d
		}
	case strings.HasSuffix(req.ExpiresIn, "m"):
		d, err := time.ParseDuration(req.ExpiresIn)
		if err == nil {
			expiresIn = d
		}
	default:
		d, err := time.ParseDuration(req.ExpiresIn)
		if err == nil {
			expiresIn = d
		}
	}

	// Get relative path for storage
	absShared, _ := filepath.Abs(s.config.SharedDir)
	relPath := strings.TrimPrefix(absPath, absShared)
	relPath = strings.TrimPrefix(relPath, string(filepath.Separator))
	if relPath == "" {
		relPath = "."
	}

	link, err := s.shareStore.Create(relPath, stat.IsDir(), req.Password, expiresIn)
	if err != nil {
		http.Error(w, "Failed to create share link", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, link)
}

func (s *FileServer) listShareLinks(w http.ResponseWriter, r *http.Request) {
	links := s.shareStore.List()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"links": links,
	})
}

func (s *FileServer) deleteShareLink(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		var req struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Token is required", http.StatusBadRequest)
			return
		}
		token = req.Token
	}

	if token == "" {
		http.Error(w, "Token is required", http.StatusBadRequest)
		return
	}

	s.shareStore.Delete(token)
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}
