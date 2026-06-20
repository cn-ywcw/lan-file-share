package server

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
)

// Config holds the server configuration.
type Config struct {
	Port             int               `json:"port"`
	SharedDir        string            `json:"shared_dir"`
	MaxUpload        int64             `json:"max_upload_mb"` // MB
	AuthUser         string            `json:"auth_user,omitempty"`
	AuthPass         string            `json:"auth_pass,omitempty"`
	GuestPermissions *GuestPermissions `json:"guest_permissions,omitempty"`
	ConfigFile       string            `json:"-"`
}

// GuestPermissions controls what unauthenticated guests can do.
// Only meaningful when AuthUser is set; when auth is off these are ignored (all true).
type GuestPermissions struct {
	Browse   bool `json:"browse"`   // list & view directory
	Upload   bool `json:"upload"`   // upload files
	Download bool `json:"download"` // download files
	Preview  bool `json:"preview"`  // preview files inline
	Delete   bool `json:"delete"`   // delete files
	Share    bool `json:"share"`    // create share links
}

// DefaultConfig returns a default configuration.
func DefaultConfig() *Config {
	exe, _ := os.Executable()
	base := filepath.Dir(exe)
	shared := filepath.Join(base, "shared")
	// If the exe dir doesn't work, fall back to CWD
	if _, err := os.Stat(shared); os.IsNotExist(err) {
		os.MkdirAll(shared, 0755)
	}
	return &Config{
		Port:      8080,
		SharedDir: shared,
		MaxUpload: 500, // 500 MB default
		GuestPermissions: &GuestPermissions{
			Browse:   true,
			Upload:   false,
			Download: false,
			Preview:  false,
			Delete:   false,
			Share:    false,
		},
	}
}

// LocalIP returns the first non-loopback IPv4 address.
func LocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			return ipnet.IP.String()
		}
	}
	return "127.0.0.1"
}

// LoadConfig loads config from a JSON file, merging with defaults.
func LoadConfig(path string) *Config {
	cfg := DefaultConfig()
	cfg.ConfigFile = path
	if path == "" {
		return cfg
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg // use defaults
	}
	var fileCfg Config
	if err := json.Unmarshal(data, &fileCfg); err != nil {
		fmt.Printf("warning: failed to parse config %s: %v\n", path, err)
		return cfg
	}
	if fileCfg.Port != 0 {
		cfg.Port = fileCfg.Port
	}
	if fileCfg.SharedDir != "" {
		cfg.SharedDir = fileCfg.SharedDir
	}
	if fileCfg.MaxUpload != 0 {
		cfg.MaxUpload = fileCfg.MaxUpload
	}
	if fileCfg.AuthUser != "" {
		cfg.AuthUser = fileCfg.AuthUser
	}
	if fileCfg.AuthPass != "" {
		cfg.AuthPass = fileCfg.AuthPass
	}
	if fileCfg.GuestPermissions != nil {
		cfg.GuestPermissions = fileCfg.GuestPermissions
	}
	return cfg
}

// SaveConfig writes the config to its JSON file.
func (c *Config) SaveConfig() error {
	if c.ConfigFile == "" {
		return fmt.Errorf("no config file path set")
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.ConfigFile, data, 0644)
}
