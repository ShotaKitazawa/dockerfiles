package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type config struct {
	listenAddr                string
	upstreamURL               string
	allowedCIDR               string // empty = no IP restriction
	trustedProxyCIDR          string // empty = use RemoteAddr directly
	allowedAudiences          []string
	allowedRedirectURIOrigins []string
	clientSecretExpiresIn     time.Duration // 0 = disabled
}

func loadConfig() config {
	parse := func(v string) []string {
		var out []string
		for _, s := range strings.Split(v, ",") {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	cfg := config{
		listenAddr:                getenv("LISTEN_ADDR", ":8080"),
		upstreamURL:               getenv("UPSTREAM_URL", "http://release-name-hydra-public:4444"),
		allowedCIDR:               os.Getenv("ALLOWED_CIDR"),
		trustedProxyCIDR:          os.Getenv("TRUSTED_PROXY_CIDR"),
		allowedAudiences:          parse(os.Getenv("ALLOWED_AUDIENCES")),
		allowedRedirectURIOrigins: parse(os.Getenv("ALLOWED_REDIRECT_URI_ORIGINS")),
	}
	if raw := getenv("CLIENT_SECRET_EXPIRES_IN", "168h"); raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil {
			slog.Error("invalid CLIENT_SECRET_EXPIRES_IN", "err", err)
			os.Exit(1)
		}
		cfg.clientSecretExpiresIn = d
	}
	return cfg
}

// getClientIP extracts the real client IP from X-Forwarded-For by stripping
// trusted proxy IPs from right to left, mirroring nginx real_ip_recursive behavior.
// Falls back to RemoteAddr when the header is absent or all IPs are trusted.
func getClientIP(r *http.Request, trustedNet *net.IPNet) net.IP {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		for i := len(parts) - 1; i >= 0; i-- {
			ip := net.ParseIP(strings.TrimSpace(parts[i]))
			if ip == nil || !trustedNet.Contains(ip) {
				return ip
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return net.ParseIP(r.RemoteAddr)
	}
	return net.ParseIP(host)
}

// clientIPFromRemoteAddr extracts the client IP from RemoteAddr without trusted-proxy logic.
func clientIPFromRemoteAddr(r *http.Request) net.IP {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return net.ParseIP(r.RemoteAddr)
	}
	return net.ParseIP(host)
}

// sanitizeResponse removes Hydra-specific quirks from DCR POST response bodies:
//   - "contacts":null → field deleted
//   - "client_uri":"" / "logo_uri":"" / "tos_uri":"" / "policy_uri":"" → field deleted
func sanitizeResponse(resp *http.Response) error {
	if resp.Request.Method != http.MethodPost {
		return nil
	}
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return nil
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(body, &obj); err != nil {
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return nil
	}

	modified := false
	if v, ok := obj["contacts"]; ok && bytes.Equal(v, []byte("null")) {
		delete(obj, "contacts")
		modified = true
	}
	for _, field := range []string{"client_uri", "logo_uri", "tos_uri", "policy_uri"} {
		if v, ok := obj[field]; ok {
			var s string
			if json.Unmarshal(v, &s) == nil && s == "" {
				delete(obj, field)
				modified = true
			}
		}
	}

	if modified {
		if newBody, err := json.Marshal(obj); err == nil {
			resp.Body = io.NopCloser(bytes.NewReader(newBody))
			resp.ContentLength = -1
			resp.Header.Del("Content-Length")
			return nil
		}
	}

	resp.Body = io.NopCloser(bytes.NewReader(body))
	return nil
}

// validateAudiences returns an error if any audience in the DCR request body
// is not in allowed. Passes through if allowed is empty or body is non-JSON.
func validateAudiences(body []byte, allowed []string) error {
	if len(allowed) == 0 {
		return nil
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil
	}
	raw, ok := obj["audience"]
	if !ok {
		return nil
	}
	var audiences []string
	if err := json.Unmarshal(raw, &audiences); err != nil {
		return nil
	}
	allowedSet := make(map[string]bool, len(allowed))
	for _, a := range allowed {
		allowedSet[a] = true
	}
	for _, a := range audiences {
		if !allowedSet[a] {
			return fmt.Errorf("audience not allowed: %s", a)
		}
	}
	return nil
}

// validateRedirectURIs returns an error if any redirect_uri in the DCR request
// body is neither a loopback URI nor an origin listed in allowedOrigins.
// When allowedOrigins is empty only loopback URIs are accepted.
func validateRedirectURIs(body []byte, allowedOrigins []string) error {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil
	}
	raw, ok := obj["redirect_uris"]
	if !ok {
		return nil
	}
	var uris []string
	if err := json.Unmarshal(raw, &uris); err != nil {
		return nil
	}
	for _, uri := range uris {
		if !isAllowedRedirectURI(uri, allowedOrigins) {
			return fmt.Errorf("redirect_uri not allowed: %s", uri)
		}
	}
	return nil
}

func isAllowedRedirectURI(rawURI string, allowedOrigins []string) bool {
	u, err := url.Parse(rawURI)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if host == "localhost" || host == "::1" || strings.HasPrefix(host, "127.") {
		return true
	}
	origin := u.Scheme + "://" + u.Host
	for _, a := range allowedOrigins {
		if origin == a {
			return true
		}
	}
	return false
}

// injectClientExpiry sets client_secret_expires_at to now+ttl, overriding any
// caller-supplied value to enforce the maximum lifetime for DCR clients.
func injectClientExpiry(body []byte, ttl time.Duration) ([]byte, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(body, &obj); err != nil {
		return body, nil
	}
	expiresAt := time.Now().Add(ttl).Unix()
	obj["client_secret_expires_at"] = json.RawMessage(strconv.FormatInt(expiresAt, 10))
	return json.Marshal(obj)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg}) //nolint:errcheck
}

func main() {
	cfg := loadConfig()

	var (
		allowedNet *net.IPNet
		trustedNet *net.IPNet
	)
	if cfg.allowedCIDR != "" {
		var err error
		_, allowedNet, err = net.ParseCIDR(cfg.allowedCIDR)
		if err != nil {
			slog.Error("invalid ALLOWED_CIDR", "err", err)
			os.Exit(1)
		}
		if cfg.trustedProxyCIDR != "" {
			_, trustedNet, err = net.ParseCIDR(cfg.trustedProxyCIDR)
			if err != nil {
				slog.Error("invalid TRUSTED_PROXY_CIDR", "err", err)
				os.Exit(1)
			}
		}
	}

	target, err := url.Parse(cfg.upstreamURL)
	if err != nil {
		slog.Error("invalid UPSTREAM_URL", "err", err)
		os.Exit(1)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ModifyResponse = sanitizeResponse
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		slog.Error("upstream error", "err", err)
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if allowedNet != nil {
			var clientIP net.IP
			if trustedNet != nil {
				clientIP = getClientIP(r, trustedNet)
			} else {
				clientIP = clientIPFromRemoteAddr(r)
			}
			if clientIP == nil || !allowedNet.Contains(clientIP) {
				slog.Warn("denied", "ip", clientIP)
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
		}

		if r.Method == http.MethodPost && r.Body != nil {
			body, err := io.ReadAll(r.Body)
			r.Body.Close()
			if err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			if err := validateAudiences(body, cfg.allowedAudiences); err != nil {
				writeJSONError(w, http.StatusBadRequest, err.Error())
				return
			}
			if err := validateRedirectURIs(body, cfg.allowedRedirectURIOrigins); err != nil {
				writeJSONError(w, http.StatusBadRequest, err.Error())
				return
			}
			if cfg.clientSecretExpiresIn > 0 {
				var err error
				body, err = injectClientExpiry(body, cfg.clientSecretExpiresIn)
				if err != nil {
					http.Error(w, "internal error", http.StatusInternalServerError)
					return
				}
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			r.ContentLength = int64(len(body))
		}

		proxy.ServeHTTP(w, r)
	})

	slog.Info("starting dcr-proxy", "addr", cfg.listenAddr, "upstream", cfg.upstreamURL)
	if err := http.ListenAndServe(cfg.listenAddr, mux); err != nil {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
