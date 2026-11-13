package main

import (
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (a *app) handleFile(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.handleFileGet(w, r)
	case http.MethodPost:
		a.handleFileUpload(w, r)
	default:
		apiMethodNotAllowed(w)
	}
}

func (a *app) handleFileGet(w http.ResponseWriter, r *http.Request) {
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
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": info.Name()}))
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}

func (a *app) handleFileUpload(w http.ResponseWriter, r *http.Request) {
	// parsing req
	realFolder, err := getAbsPath(r.URL.Query().Get("folder"))
	if err != nil {
		apiWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		apiWriteError(w, http.StatusBadRequest, "invalid upload form")
		return
	}
	defer r.MultipartForm.RemoveAll()

	// defensive chk that path is correct
	value := strings.TrimSpace(r.FormValue("path"))
	if value == "" || filepath.IsAbs(value) {
		apiWriteError(w, http.StatusBadRequest, "path must be relative to folder")
		return
	}
	relative := filepath.Clean(filepath.FromSlash(value))
	if relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		apiWriteError(w, http.StatusBadRequest, "path must stay inside folder")
		return
	}
	root, err := os.OpenRoot(realFolder)
	if err != nil {
		apiWriteError(w, http.StatusBadRequest, "could not open folder")
		return
	}
	defer root.Close()

	// setup the directories, and chk if filepath alr exist
	if err := root.MkdirAll(filepath.Dir(relative), 0755); err != nil {
		apiWriteError(w, http.StatusBadRequest, "could not create destination folder")
		return
	}

	info, err := root.Lstat(relative)
	if err == nil {
		if !info.Mode().IsRegular() {
			apiWriteError(w, http.StatusBadRequest, "destination is not a regular file")
			return
		}
		if r.FormValue("overwrite") != "true" {
			apiWriteError(w, http.StatusConflict, "destination already exists")
			return
		}
	} else if !os.IsNotExist(err) {
		apiWriteError(w, http.StatusBadRequest, "could not check destination")
		return
	}

	// read and save file
	file, _, err := r.FormFile("file")
	if err != nil {
		apiWriteError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	if r.FormValue("overwrite") == "true" && info != nil {
		if err := root.Remove(relative); err != nil {
			apiWriteError(w, http.StatusBadRequest, "could not replace destination")
			return
		}
	}
	output, err := root.OpenFile(relative, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		if os.IsExist(err) {
			apiWriteError(w, http.StatusConflict, "destination already exists")
		} else {
			apiWriteError(w, http.StatusBadRequest, "could not write destination")
		}
		return
	}
	if _, err := io.Copy(output, file); err != nil {
		output.Close()
		apiWriteError(w, http.StatusInternalServerError, "could not save file")
		return
	}
	if err := output.Close(); err != nil {
		apiWriteError(w, http.StatusInternalServerError, "could not save file")
		return
	}
	apiWriteJSON(w, http.StatusCreated, map[string]string{"path": filepath.ToSlash(relative)})
}
