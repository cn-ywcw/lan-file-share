package server

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// AdminHandler handles administrative operations (config read/write).
func (s *FileServer) AdminHandler(w http.ResponseWriter, r *http.Request) {
	// All admin operations require authentication
	if !s.isAdminRequest(r) {
		AuthChallenge(w)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.getAdminConfig(w, r)
	case http.MethodPut:
		s.updateAdminConfig(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// AdminStatusHandler returns server status info.
func (s *FileServer) AdminStatusHandler(w http.ResponseWriter, r *http.Request) {
	if !s.isAdminRequest(r) {
		AuthChallenge(w)
		return
	}

	cfg := s.config
	info := map[string]interface{}{
		"port":           cfg.Port,
		"shared_dir":     cfg.SharedDir,
		"max_upload_mb":  cfg.MaxUpload,
		"has_auth":       cfg.AuthUser != "",
		"auth_user":      cfg.AuthUser,
		"config_file":    cfg.ConfigFile,
		"local_ip":       LocalIP(),
		"guest_settings": cfg.GuestPermissions,
	}
	writeJSON(w, http.StatusOK, info)
}

func (s *FileServer) getAdminConfig(w http.ResponseWriter, r *http.Request) {
	// Return a sanitised config (no passwords in response)
	cfg := s.config
	resp := map[string]interface{}{
		"port":              cfg.Port,
		"max_upload_mb":     cfg.MaxUpload,
		"shared_dir":        cfg.SharedDir,
		"auth_user":         cfg.AuthUser,
		"has_auth":          cfg.AuthUser != "",
		"config_file":       cfg.ConfigFile != "",
		"guest_permissions": cfg.GuestPermissions,
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *FileServer) updateAdminConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Port              *int              `json:"port,omitempty"`
		MaxUploadMB       *int64            `json:"max_upload_mb,omitempty"`
		AuthUser          *string           `json:"auth_user,omitempty"`
		AuthPass          *string           `json:"auth_pass,omitempty"`
		GuestPermissions  *GuestPermissions `json:"guest_permissions,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	changes := make([]string, 0)

	if req.Port != nil {
		s.config.Port = *req.Port
		changes = append(changes, fmt.Sprintf("port → %d", *req.Port))
	}
	if req.MaxUploadMB != nil {
		s.config.MaxUpload = *req.MaxUploadMB
		changes = append(changes, fmt.Sprintf("max_upload → %dMB", *req.MaxUploadMB))
	}
	if req.AuthUser != nil {
		s.config.AuthUser = *req.AuthUser
		changes = append(changes, fmt.Sprintf("auth_user → %s", *req.AuthUser))
	}
	if req.AuthPass != nil {
		s.config.AuthPass = *req.AuthPass
		changes = append(changes, "auth_pass → (updated)")
	}
	if req.GuestPermissions != nil {
		s.config.GuestPermissions = req.GuestPermissions
		changes = append(changes, "guest_permissions → (updated)")
	}

	// Persist to config file if available
	configSaved := false
	if s.config.ConfigFile != "" {
		if err := s.config.SaveConfig(); err == nil {
			configSaved = true
		}
	}

	resp := map[string]interface{}{
		"success":      true,
		"changes":      changes,
		"config_saved": configSaved,
		"note":         "Port changes require a server restart to take effect",
	}
	writeJSON(w, http.StatusOK, resp)
}

// isAdminRequest checks if the request is authenticated as admin.
// When no auth is configured, all requests are considered admin.
func (s *FileServer) isAdminRequest(r *http.Request) bool {
	if s.config.AuthUser == "" {
		return true
	}
	user, pass, ok := r.BasicAuth()
	if !ok {
		return false
	}
	return user == s.config.AuthUser && pass == s.config.AuthPass
}
