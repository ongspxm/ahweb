package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (a *app) handleFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		apiMethodNotAllowed(w)
		return
	}
	realFolder, err := getAbsPath(r.URL.Query().Get("folder"))
	if err != nil {
		apiWriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	// resolve the folder-relative path, keeping symlinks inside the folder.
	value := strings.TrimSpace(r.URL.Query().Get("path"))
	if value == "" {
		value = "."
	}
	if filepath.IsAbs(value) {
		apiWriteError(w, http.StatusNotFound, "path must be relative to folder")
		return
	}
	fullPath, err := filepath.EvalSymlinks(filepath.Join(realFolder, filepath.FromSlash(value)))
	if err != nil {
		apiWriteError(w, http.StatusNotFound, "path does not exist")
		return
	}
	if fullPath != realFolder && !strings.HasPrefix(fullPath, realFolder+string(os.PathSeparator)) {
		apiWriteError(w, http.StatusNotFound, "path is outside folder")
		return
	}
	relPath, err := filepath.Rel(realFolder, fullPath)
	if err != nil {
		apiWriteError(w, http.StatusNotFound, "invalid path")
		return
	}
	relPath = filepath.ToSlash(relPath)
	info, err := os.Stat(fullPath)
	if err != nil {
		apiWriteError(w, http.StatusNotFound, "path does not exist")
		return
	}

	// serve directory listing
	if info.IsDir() {
		entries, err := os.ReadDir(fullPath)
		if err != nil {
			apiWriteError(w, http.StatusBadRequest, "could not read folder")
			return
		}
		dirs, files := []string{}, []string{}
		for _, entry := range entries {
			path := strings.TrimPrefix(relPath+"/"+entry.Name(), "./")
			info, err := os.Stat(filepath.Join(fullPath, entry.Name()))
			if err == nil && info.IsDir() {
				dirs = append(dirs, path)
			} else {
				files = append(files, path)
			}
		}
		apiWriteJSON(w, http.StatusOK, struct {
			Dirs  []string `json:"dirs"`
			Files []string `json:"files"`
		}{dirs, files})
		return
	}

	// serve files
	if !info.Mode().IsRegular() {
		apiWriteError(w, http.StatusBadRequest, "path is not a regular file")
		return
	}
	file, err := os.Open(fullPath)
	if err != nil {
		apiWriteError(w, http.StatusNotFound, "could not open file")
		return
	}
	defer file.Close()
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}
