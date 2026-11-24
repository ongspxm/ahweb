package main

import (
	"encoding/json"
	"net/http"
	"strings"
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

func TestStatusAPIShouldIncludeUpdateCheckFields(t *testing.T) {
	a := &app{}
	res := testRequest(a.handleStatus, http.MethodGet, "/api/status", "")
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	var body struct {
		UpdateCheckFailed *bool  `json:"update_check_failed"`
		UpdateCommand     string `json:"update_command"`
		Update            string `json:"update"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.UpdateCheckFailed == nil {
		t.Fatal("response is missing update_check_failed")
	}
	if body.UpdateCommand != "go install github.com/ongspxm/ahweb@latest && ahweb install && ahweb restart" {
		t.Fatalf("update_command = %q", body.UpdateCommand)
	}
	if body.Update != "" {
		if !strings.HasPrefix(body.Update, "Update available: ") {
			t.Fatalf("update = %q, want availability notice", body.Update)
		}
		if strings.Contains(body.Update, body.UpdateCommand) {
			t.Fatalf("update = %q, should not contain update_command", body.Update)
		}
	}
}

func TestFormatUpdateNotice(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    string
	}{
		{name: "newer release", current: "v1.2.3", latest: "v1.2.4", want: "Update available: v1.2.4."},
		{name: "equal versions", current: "v1.2.3", latest: "v1.2.3", want: ""},
		{name: "older latest", current: "v1.2.4", latest: "v1.2.3", want: ""},
		{name: "invalid current", current: "invalid", latest: "v1.2.4", want: ""},
		{name: "empty current", current: "", latest: "v1.2.4", want: ""},
		{name: "invalid latest", current: "v1.2.3", latest: "invalid", want: ""},
		{name: "empty latest", current: "v1.2.3", latest: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatUpdateNotice(tt.current, tt.latest); got != tt.want {
				t.Fatalf("formatUpdateNotice(%q, %q) = %q, want %q", tt.current, tt.latest, got, tt.want)
			}
		})
	}
}
