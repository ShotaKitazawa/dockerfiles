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
	_, homeCIDR, _ = net.ParseCIDR("192.168.0.0/24")
	_, podCIDR, _  = net.ParseCIDR("10.0.0.0/8")
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

// --- sanitizeResponse ---

func TestSanitizeResponse_RemovesContactsNull(t *testing.T) {
	body := `{"client_id":"abc","contacts":null,"grant_types":["authorization_code"]}`
	resp := makeJSONPostResp(body)
	if err := sanitizeResponse(resp); err != nil {
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

func TestSanitizeResponse_PreservesContactsArray(t *testing.T) {
	body := `{"client_id":"abc","contacts":["admin@example.com"]}`
	resp := makeJSONPostResp(body)
	if err := sanitizeResponse(resp); err != nil {
		t.Fatal(err)
	}
	got := readBody(t, resp)
	if !strings.Contains(got, `"contacts"`) {
		t.Fatalf("non-null contacts should be preserved: %s", got)
	}
}

func TestSanitizeResponse_RemovesEmptyURLFields(t *testing.T) {
	body := `{"client_id":"abc","client_uri":"","logo_uri":"","tos_uri":"","policy_uri":""}`
	resp := makeJSONPostResp(body)
	if err := sanitizeResponse(resp); err != nil {
		t.Fatal(err)
	}
	got := readBody(t, resp)
	for _, field := range []string{"client_uri", "logo_uri", "tos_uri", "policy_uri"} {
		if strings.Contains(got, `"`+field+`"`) {
			t.Fatalf("empty %s should be removed: %s", field, got)
		}
	}
	if !strings.Contains(got, "client_id") {
		t.Fatalf("other fields missing: %s", got)
	}
}

func TestSanitizeResponse_PreservesNonEmptyURLFields(t *testing.T) {
	body := `{"client_uri":"https://example.com","logo_uri":"https://example.com/logo.png"}`
	resp := makeJSONPostResp(body)
	if err := sanitizeResponse(resp); err != nil {
		t.Fatal(err)
	}
	got := readBody(t, resp)
	if !strings.Contains(got, "client_uri") {
		t.Fatalf("non-empty client_uri should be preserved: %s", got)
	}
}

func TestSanitizeResponse_NonPost(t *testing.T) {
	body := `{"contacts":null}`
	resp := makeJSONPostResp(body)
	resp.Request.Method = http.MethodGet
	if err := sanitizeResponse(resp); err != nil {
		t.Fatal(err)
	}
	got := readBody(t, resp)
	if got != body {
		t.Fatalf("expected unchanged body for GET, got %s", got)
	}
}

func TestSanitizeResponse_NonJSON(t *testing.T) {
	body := "not json"
	resp := makeJSONPostResp(body)
	resp.Header.Set("Content-Type", "text/plain")
	if err := sanitizeResponse(resp); err != nil {
		t.Fatal(err)
	}
	got := readBody(t, resp)
	if got != body {
		t.Fatalf("expected unchanged body for non-JSON, got %s", got)
	}
}

// --- validateAudiences ---

func TestValidateAudiences_AllowedPasses(t *testing.T) {
	body := `{"audience":["https://api.example.com"]}`
	if err := validateAudiences([]byte(body), []string{"https://api.example.com"}); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidateAudiences_DisallowedRejects(t *testing.T) {
	body := `{"audience":["https://evil.example.com"]}`
	if err := validateAudiences([]byte(body), []string{"https://api.example.com"}); err == nil {
		t.Fatal("expected error for disallowed audience")
	}
}

func TestValidateAudiences_EmptyAllowedSkips(t *testing.T) {
	body := `{"audience":["https://anything.example.com"]}`
	if err := validateAudiences([]byte(body), nil); err != nil {
		t.Fatalf("empty allowed list should skip validation, got %v", err)
	}
}

func TestValidateAudiences_NoFieldPasses(t *testing.T) {
	body := `{"client_name":"test"}`
	if err := validateAudiences([]byte(body), []string{"https://api.example.com"}); err != nil {
		t.Fatalf("missing audience field should pass, got %v", err)
	}
}

func TestValidateAudiences_NonJSONPasses(t *testing.T) {
	if err := validateAudiences([]byte("not json"), []string{"https://api.example.com"}); err != nil {
		t.Fatalf("non-JSON should pass, got %v", err)
	}
}

// --- validateRedirectURIs ---

func TestValidateRedirectURIs_LoopbackAlwaysAllowed(t *testing.T) {
	cases := []string{
		`{"redirect_uris":["http://localhost:8080/callback"]}`,
		`{"redirect_uris":["http://127.0.0.1:9090/cb"]}`,
		`{"redirect_uris":["http://127.42.0.1/cb"]}`,
	}
	for _, body := range cases {
		if err := validateRedirectURIs([]byte(body), nil); err != nil {
			t.Fatalf("loopback URI should be allowed: %v (body: %s)", err, body)
		}
	}
}

func TestValidateRedirectURIs_AllowedOriginPasses(t *testing.T) {
	body := `{"redirect_uris":["https://korpus.kanatakita.com/callback"]}`
	if err := validateRedirectURIs([]byte(body), []string{"https://korpus.kanatakita.com"}); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidateRedirectURIs_DisallowedRejects(t *testing.T) {
	body := `{"redirect_uris":["https://evil.example.com/callback"]}`
	if err := validateRedirectURIs([]byte(body), nil); err == nil {
		t.Fatal("expected error for disallowed redirect_uri")
	}
}

func TestValidateRedirectURIs_NoFieldPasses(t *testing.T) {
	body := `{"client_name":"test"}`
	if err := validateRedirectURIs([]byte(body), nil); err != nil {
		t.Fatalf("missing redirect_uris field should pass, got %v", err)
	}
}

func TestValidateRedirectURIs_NonJSONPasses(t *testing.T) {
	if err := validateRedirectURIs([]byte("not json"), nil); err != nil {
		t.Fatalf("non-JSON should pass, got %v", err)
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
