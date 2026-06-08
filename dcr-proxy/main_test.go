package main

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var (
	_, homeCIDR,    _ = net.ParseCIDR("192.168.0.0/24")
	_, podCIDR,     _ = net.ParseCIDR("10.0.0.0/8")
)

func TestGetClientIP_HomeNetworkAllowed(t *testing.T) {
	r := httptest.NewRequest("POST", "/", nil)
	r.Header.Set("X-Forwarded-For", "192.168.0.5")
	ip := getClientIP(r, podCIDR)
	if !homeCIDR.Contains(ip) {
		t.Fatalf("expected home IP, got %v", ip)
	}
}

func TestGetClientIP_InternetDenied(t *testing.T) {
	r := httptest.NewRequest("POST", "/", nil)
	r.Header.Set("X-Forwarded-For", "8.8.8.8")
	ip := getClientIP(r, podCIDR)
	if homeCIDR.Contains(ip) {
		t.Fatalf("expected non-home IP, got %v", ip)
	}
}

func TestGetClientIP_TrustedProxyStripped(t *testing.T) {
	r := httptest.NewRequest("POST", "/", nil)
	// Pod proxy added 10.x to the right; real client is on the left.
	r.Header.Set("X-Forwarded-For", "192.168.0.42, 10.0.1.20")
	ip := getClientIP(r, podCIDR)
	if !homeCIDR.Contains(ip) {
		t.Fatalf("expected home IP after stripping pod IP, got %v", ip)
	}
}

func TestGetClientIP_AllTrustedFallsBackToRemoteAddr(t *testing.T) {
	r := httptest.NewRequest("POST", "/", nil)
	r.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.1.2")
	r.RemoteAddr = "192.168.0.99:12345"
	ip := getClientIP(r, podCIDR)
	if ip == nil || ip.String() != "192.168.0.99" {
		t.Fatalf("expected RemoteAddr fallback 192.168.0.99, got %v", ip)
	}
}

func TestStripContactsNull_RemovesNull(t *testing.T) {
	body := `{"client_id":"abc","contacts":null,"grant_types":["authorization_code"]}`
	resp := makeJSONPostResp(body)
	if err := stripContactsNull(resp); err != nil {
		t.Fatal(err)
	}
	got := readBody(t, resp)
	if strings.Contains(got, "contacts") {
		t.Fatalf("contacts still present: %s", got)
	}
	if !strings.Contains(got, "client_id") {
		t.Fatalf("other fields missing: %s", got)
	}
}

func TestStripContactsNull_PreservesArray(t *testing.T) {
	body := `{"client_id":"abc","contacts":["admin@example.com"]}`
	resp := makeJSONPostResp(body)
	if err := stripContactsNull(resp); err != nil {
		t.Fatal(err)
	}
	got := readBody(t, resp)
	if !strings.Contains(got, `"contacts"`) {
		t.Fatalf("non-null contacts should be preserved: %s", got)
	}
}

func TestStripContactsNull_NonPost(t *testing.T) {
	body := `{"contacts":null}`
	resp := makeJSONPostResp(body)
	resp.Request.Method = http.MethodGet
	if err := stripContactsNull(resp); err != nil {
		t.Fatal(err)
	}
	got := readBody(t, resp)
	// GET responses pass through unchanged.
	if got != body {
		t.Fatalf("expected unchanged body for GET, got %s", got)
	}
}

func TestStripContactsNull_NonJSON(t *testing.T) {
	body := "not json"
	resp := makeJSONPostResp(body)
	resp.Header.Set("Content-Type", "text/plain")
	if err := stripContactsNull(resp); err != nil {
		t.Fatal(err)
	}
	got := readBody(t, resp)
	if got != body {
		t.Fatalf("expected unchanged body for non-JSON, got %s", got)
	}
}

func makeJSONPostResp(body string) *http.Response {
	req := httptest.NewRequest(http.MethodPost, "/oauth2/register", nil)
	return &http.Response{
		Request: req,
		Header:  http.Header{"Content-Type": []string{"application/json"}},
		Body:    io.NopCloser(bytes.NewBufferString(body)),
	}
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
