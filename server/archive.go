package server

import (
	"archive/zip"
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
)

// ArchiveEntry represents a file inside an archive.
type ArchiveEntry struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	IsDir   bool   `json:"is_dir"`
	ModTime string `json:"mod_time"`
	Index   int    `json:"index"` // index within archive for extraction
}

// ArchiveInfo returns metadata about an archive.
type ArchiveInfo struct {
	Format    string         `json:"format"`    // "zip", "rar", "7z", "tar", "gz", etc.
	Path      string         `json:"path"`      // relative path
	Name      string         `json:"name"`      // filename
	Size      int64          `json:"size"`      // archive file size
	Entries   []ArchiveEntry `json:"entries"`   // contents listing
	Supported bool           `json:"supported"` // whether listing is supported
	Message   string         `json:"message,omitempty"`
}

// ArchivePreviewHandler returns the contents of an archive file.
// GET /api/archive?path=xxx.zip
func (s *FileServer) ArchivePreviewHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !s.CheckPermission(w, r, PermBrowse) {
		return
	}

	relPath := sanitizePath(r.URL.Query().Get("path"))
	absPath, info, ok := s.validatePath(w, relPath)
	if !ok {
		return
	}

	if info.IsDir() {
		http.Error(w, "Not a file", http.StatusBadRequest)
		return
	}

	ext := strings.ToLower(filepath.Ext(info.Name()))
	name := info.Name()

	result := &ArchiveInfo{
		Format:  ext,
		Path:    relPath,
		Name:    name,
		Size:    info.Size(),
		Entries: make([]ArchiveEntry, 0),
	}

	switch ext {
	case ".zip":
		result.Supported = true
		result.Message = fmt.Sprintf("ZIP archive — %d files total", 0)
		entries, total, err := listZipContents(absPath)
		if err != nil {
			result.Message = fmt.Sprintf("Failed to read ZIP: %v", err)
			result.Supported = false
		} else {
			result.Entries = entries
			result.Message = fmt.Sprintf("ZIP archive — %d files total", total)
		}
	case ".rar":
		result.Supported = false
		result.Message = "RAR format preview not supported. Download and use WinRAR/7-Zip to extract."
	case ".7z":
		result.Supported = false
		result.Message = "7z format preview not supported. Download and use 7-Zip to extract."
	case ".tar", ".tar.gz", ".tgz", ".gz":
		result.Supported = false
		result.Message = "TAR/GZip format preview not supported in browser. Download to extract."
	default:
		result.Supported = false
		result.Message = "Archive format not recognized. Download to extract."
	}

	// Count total (non-dir) entries
	totalFiles := 0
	for _, e := range result.Entries {
		if !e.IsDir {
			totalFiles++
		}
	}
	if totalFiles > 0 && result.Message != "" && ext == ".zip" {
		result.Message = fmt.Sprintf("ZIP archive — %d files", totalFiles)
	}

	writeJSON(w, http.StatusOK, result)
}

// listZipContents reads a zip file and returns its directory listing.
func listZipContents(path string) ([]ArchiveEntry, int, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, 0, err
	}
	defer reader.Close()

	entries := make([]ArchiveEntry, 0, len(reader.File))
	totalFiles := 0

	for _, f := range reader.File {
		modTime := f.Modified
		if modTime.IsZero() {
			modTime = f.ModTime()
		}
		entry := ArchiveEntry{
			Name:    f.Name,
			Size:    int64(f.UncompressedSize64),
			IsDir:   f.FileInfo().IsDir(),
			ModTime: modTime.Format("2006-01-02 15:04:05"),
		}
		if !entry.IsDir {
			totalFiles++
		}
		entries = append(entries, entry)
	}

	// Sort: dirs first, then alphabetical
	sortArchiveEntries(entries)

	return entries, totalFiles, nil
}

func sortArchiveEntries(entries []ArchiveEntry) {
	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.IsDir != b.IsDir {
			return a.IsDir // directories first
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})
}
