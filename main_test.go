package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testRequest(handler http.HandlerFunc, method, target, body string) *httptest.ResponseRecorder {
	res := httptest.NewRecorder()
	handler(res, httptest.NewRequest(method, target, strings.NewReader(body)))
	return res
}

func TestHostHandlerReturnsPlainTextIdentity(t *testing.T) {
	a := &app{hostIdentity: "server_user_8080_linux_amd64"}
	res := testRequest(a.handleHost, http.MethodGet, "/host", "")
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	if got := res.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	if got := res.Body.String(); got != a.hostIdentity {
		t.Fatalf("body = %q, want %q", got, a.hostIdentity)
	}
}
