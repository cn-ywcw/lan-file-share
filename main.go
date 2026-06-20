package main

import (
	"embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"lan-file-share/server"
)

//go:embed static/*
var embedFS embed.FS

func main() {
	port := flag.Int("port", 8080, "HTTP server port")
	sharedDir := flag.String("dir", "", "Path to shared directory (default: ./shared)")
	maxUpload := flag.Int64("max-upload", 500, "Maximum upload size in MB")
	authUser := flag.String("user", "", "Basic auth username (empty = no auth)")
	authPass := flag.String("pass", "", "Basic auth password")
	configFile := flag.String("config", "", "Path to JSON config file")
	flag.Parse()

	cfg := server.DefaultConfig()

	if *configFile != "" {
		cfg = server.LoadConfig(*configFile)
	}

	// CLI flags override config file values
	if *port != 8080 {
		cfg.Port = *port
	}
	if *sharedDir != "" {
		cfg.SharedDir = *sharedDir
	}
	if *maxUpload != 500 {
		cfg.MaxUpload = *maxUpload
	}
	if *authUser != "" {
		cfg.AuthUser = *authUser
	}
	if *authPass != "" {
		cfg.AuthPass = *authPass
	}

	// Resolve shared directory
	if cfg.SharedDir == "" {
		exe, _ := os.Executable()
		cfg.SharedDir = filepath.Join(filepath.Dir(exe), "shared")
	}
	absDir, err := filepath.Abs(cfg.SharedDir)
	if err == nil {
		cfg.SharedDir = absDir
	}
	// Create shared dir if it doesn't exist
	if err := os.MkdirAll(cfg.SharedDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create shared directory %s: %v\n", cfg.SharedDir, err)
		os.Exit(1)
	}

	srv := server.New(cfg, embedFS)
	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
