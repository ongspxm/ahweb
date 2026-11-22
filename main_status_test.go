package main

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestCheckUpdateShouldReportDevBuild(t *testing.T) {
	version, notice, failed := checkUpdateNotice()
	if version != "(devel)" {
		t.Skipf("test binary version = %q, want development build", version)
	}
	if notice != "" || !failed {
		t.Fatalf("development build check = (%q, %t), want no notice and failure", notice, failed)
	}
}

func TestStatusAPIShouldIncludeCheckFailtureBool(t *testing.T) {
	a := &app{}
	res := testRequest(a.handleStatus, http.MethodGet, "/api/status", "")
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	var body struct {
		UpdateCheckFailed *bool `json:"update_check_failed"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.UpdateCheckFailed == nil {
		t.Fatal("response is missing update_check_failed")
	}
}
