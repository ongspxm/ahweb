package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestFileHandlerRejectsSymlinkOutsideRequestedFolder(t *testing.T) {
	folder, outside := t.TempDir(), t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(folder, "secret.txt")); err != nil {
		t.Fatal(err)
	}

	q := url.Values{"folder": {folder}, "path": {"secret.txt"}}
	res := testRequest((&app{}).handleFile, http.MethodGet, "/api/file?"+q.Encode(), "")
	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusNotFound)
	}
}

func TestFileHandlerAcceptsTildeHomeFolder(t *testing.T) {
	q := url.Values{"folder": {"~/"}, "path": {"."}}
	res := testRequest((&app{}).handleFile, http.MethodGet, "/api/file?"+q.Encode(), "")
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", res.Code, http.StatusOK, res.Body.String())
	}
}

func TestFileHandlerListsHiddenEntries(t *testing.T) {
	folder := t.TempDir()
	if err := os.WriteFile(filepath.Join(folder, ".hidden-file"), []byte("text"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(folder, ".hidden-dir"), 0700); err != nil {
		t.Fatal(err)
	}

	q := url.Values{"folder": {folder}, "path": {"."}}
	res := testRequest((&app{}).handleFile, http.MethodGet, "/api/file?"+q.Encode(), "")
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", res.Code, http.StatusOK, res.Body.String())
	}
	var listing struct {
		Dirs  []string `json:"dirs"`
		Files []string `json:"files"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &listing); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(listing.Dirs, ".hidden-dir") || !slices.Contains(listing.Files, ".hidden-file") {
		t.Fatalf("hidden entries missing: %+v", listing)
	}
}

func TestFileHandlerUploadsFilesAndRequiresOverwrite(t *testing.T) {
	folder := t.TempDir()
	upload := func(path, content string, overwrite bool) int {
		t.Helper()
		var body bytes.Buffer
		form := multipart.NewWriter(&body)
		if err := form.WriteField("path", path); err != nil {
			t.Fatal(err)
		}
		if overwrite {
			if err := form.WriteField("overwrite", "true"); err != nil {
				t.Fatal(err)
			}
		}
		file, err := form.CreateFormFile("file", filepath.Base(path))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(file, content); err != nil {
			t.Fatal(err)
		}
		if err := form.Close(); err != nil {
			t.Fatal(err)
		}
		q := url.Values{"folder": {folder}}
		req := httptest.NewRequest(http.MethodPost, "/api/file?"+q.Encode(), &body)
		req.Header.Set("Content-Type", form.FormDataContentType())
		res := httptest.NewRecorder()
		(&app{}).handleFile(res, req)
		return res.Code
	}

	res := upload("../escape.txt", "nope", false)
	if res != http.StatusBadRequest {
		t.Fatalf("outside path status = %d, want %d", res, http.StatusBadRequest)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(folder, "outside-link")); err != nil {
		t.Fatal(err)
	}
	res = upload("outside-link/escaped.txt", "nope", false)
	if res != http.StatusBadRequest {
		t.Fatalf("outside symlink status = %d, want %d", res, http.StatusBadRequest)
	}
	if _, err := os.Stat(filepath.Join(outside, "escaped.txt")); !os.IsNotExist(err) {
		t.Fatalf("outside file exists or stat failed: %v", err)
	}
	res = upload("nested/deep/new.txt", "uploaded", false)
	if res != http.StatusCreated {
		t.Fatalf("upload status = %d, want %d", res, http.StatusCreated)
	}
	data, err := os.ReadFile(filepath.Join(folder, "nested", "deep", "new.txt"))
	if err != nil || string(data) != "uploaded" {
		t.Fatalf("uploaded contents = %q, err = %v", data, err)
	}
	if res = upload("nested/deep", "nope", false); res != http.StatusBadRequest {
		t.Fatalf("directory destination status = %d, want %d", res, http.StatusBadRequest)
	}

	res = upload("nested/deep/new.txt", "ignored", false)
	if res != http.StatusConflict {
		t.Fatalf("existing file status = %d, want %d", res, http.StatusConflict)
	}
	res = upload("nested/deep/new.txt", "replaced", true)
	if res != http.StatusCreated {
		t.Fatalf("overwrite status = %d, want %d", res, http.StatusCreated)
	}
	data, err = os.ReadFile(filepath.Join(folder, "nested", "deep", "new.txt"))
	if err != nil || string(data) != "replaced" {
		t.Fatalf("overwritten contents = %q, err = %v", data, err)
	}
}
