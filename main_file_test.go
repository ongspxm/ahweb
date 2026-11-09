package main

import (
	"encoding/json"
	"net/http"
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
