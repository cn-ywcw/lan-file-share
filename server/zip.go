package server

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// zipDirectory compresses a directory and writes the zip to w.
func zipDirectory(srcDir string, w io.Writer) error {
	zw := zip.NewWriter(w)
	defer zw.Close()

	absSrc, err := filepath.Abs(srcDir)
	if err != nil {
		return err
	}

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		absPath, _ := filepath.Abs(path)
		relPath := strings.TrimPrefix(absPath, absSrc)
		relPath = strings.TrimPrefix(relPath, string(filepath.Separator))
		if relPath == "" {
			return nil
		}
		// Normalize to forward slashes for zip
		relPath = strings.ReplaceAll(relPath, "\\", "/")
		if info.IsDir() {
			relPath += "/"
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = relPath
		header.Method = zip.Deflate

		writer, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})
}
