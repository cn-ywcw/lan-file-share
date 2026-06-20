package server

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
)

// Permission constants.
const (
	PermBrowse   = "browse"
	PermUpload   = "upload"
	PermDownload = "download"
	PermPreview  = "preview"
	PermDelete   = "delete"
	PermShare    = "share"
)

// PermissionCategory groups related permissions under a label.
type PermissionCategory struct {
	ID      string              `json:"id"`
	Label   string              `json:"label"`
	Icon    string              `json:"icon"`
	Entries []PermissionEntry   `json:"entries"`
}

// PermissionEntry describes one toggleable permission.
type PermissionEntry struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Icon  string `json:"icon"`
}

// AllCategories returns the permission grouping used by the settings UI.
func AllCategories() []PermissionCategory {
	return []PermissionCategory{
		{
			ID: "browsing", Label: "文件浏览", Icon: "📂",
			Entries: []PermissionEntry{
				{Key: PermBrowse, Label: "浏览目录", Icon: "📂"},
			},
		},
		{
			ID: "operations", Label: "文件操作", Icon: "📄",
			Entries: []PermissionEntry{
				{Key: PermUpload, Label: "上传文件", Icon: "📤"},
				{Key: PermDownload, Label: "下载文件", Icon: "⬇️"},
				{Key: PermPreview, Label: "在线预览", Icon: "👁️"},
				{Key: PermDelete, Label: "删除文件", Icon: "🗑️"},
			},
		},
		{
			ID: "sharing", Label: "分享管理", Icon: "🔗",
			Entries: []PermissionEntry{
				{Key: PermShare, Label: "创建分享链接", Icon: "🔗"},
			},
		},
	}
}

// AllPermissionKeys returns every permission key name.
func AllPermissionKeys() []string {
	return []string{PermBrowse, PermUpload, PermDownload, PermPreview, PermDelete, PermShare}
}

// PermissionsHandler returns the caller's effective permissions + category defs.
// Public endpoint (no auth required) — guests can see their own limitations.
func (s *FileServer) PermissionsHandler(w http.ResponseWriter, r *http.Request) {
	perms := s.EffectivePermissions(r)
	categories := AllCategories()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"permissions": perms,
		"categories":  categories,
		"has_auth":    s.config.AuthUser != "",
	})
}

// HasPermission checks if the current request is allowed the given action.
func (s *FileServer) HasPermission(isAuth bool, permission string) bool {
	if s.config.AuthUser == "" {
		return true
	}
	if isAuth {
		return true
	}
	if s.config.GuestPermissions == nil {
		return false
	}
	switch permission {
	case PermBrowse:
		return s.config.GuestPermissions.Browse
	case PermUpload:
		return s.config.GuestPermissions.Upload
	case PermDownload:
		return s.config.GuestPermissions.Download
	case PermPreview:
		return s.config.GuestPermissions.Preview
	case PermDelete:
		return s.config.GuestPermissions.Delete
	case PermShare:
		return s.config.GuestPermissions.Share
	default:
		return false
	}
}

// EffectivePermissions returns the caller's effective permissions as a flat map.
// Used by /api/permissions for the frontend.
func (s *FileServer) EffectivePermissions(r *http.Request) map[string]bool {
	isAdmin := s.isAuthenticated(r) || s.config.AuthUser == ""
	result := make(map[string]bool)
	for _, k := range AllPermissionKeys() {
		result[k] = isAdmin || s.HasPermission(false, k)
	}
	result["isAdmin"] = isAdmin
	return result
}

// CheckPermission sends the appropriate error response when a permission check fails.
func (s *FileServer) CheckPermission(w http.ResponseWriter, r *http.Request, perm string) bool {
	if s.config.AuthUser != "" && !s.isAuthenticated(r) {
		AuthChallenge(w)
		return false
	}
	if !s.HasPermission(s.isAuthenticated(r), perm) {
		http.Error(w, "Forbidden: no permission", http.StatusForbidden)
		return false
	}
	return true
}

// isAuthenticated checks if the request carries valid admin credentials.
// Uses constant-time comparison with SHA-256 hashing to prevent timing attacks.
func (s *FileServer) isAuthenticated(r *http.Request) bool {
	if s.config.AuthUser == "" {
		return true
	}
	user, pass, ok := r.BasicAuth()
	if !ok {
		return false
	}
	// Hash credentials for constant-time comparison (same as BasicAuth middleware)
	userHash := sha256.Sum256([]byte(user))
	passHash := sha256.Sum256([]byte(pass))
	expectedUserHash := sha256.Sum256([]byte(s.config.AuthUser))
	expectedPassHash := sha256.Sum256([]byte(s.config.AuthPass))
	return subtle.ConstantTimeCompare(userHash[:], expectedUserHash[:]) == 1 &&
		subtle.ConstantTimeCompare(passHash[:], expectedPassHash[:]) == 1
}
