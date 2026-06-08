package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

type config struct {
	listenAddr       string
	upstreamURL      string
	allowedCIDR      string
	trustedProxyCIDR string
}

func loadConfig() config {
	return config{
		listenAddr:       getenv("LISTEN_ADDR", ":8080"),
		upstreamURL:      getenv("UPSTREAM_URL", "http://release-name-hydra-public:4444"),
		allowedCIDR:      getenv("ALLOWED_CIDR", "192.168.0.0/24"),
		trustedProxyCIDR: getenv("TRUSTED_PROXY_CIDR", "10.0.0.0/8"),
	}
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

// stripContactsNull removes "contacts":null from DCR POST response bodies.
// Hydra serializes the contacts field as null instead of omitting it, which
// causes the MCP TypeScript SDK's Zod schema to reject the response.
func stripContactsNull(resp *http.Response) error {
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

	if v, ok := obj["contacts"]; ok && bytes.Equal(v, []byte("null")) {
		delete(obj, "contacts")
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

func main() {
	cfg := loadConfig()

	_, allowedNet, err := net.ParseCIDR(cfg.allowedCIDR)
	if err != nil {
		slog.Error("invalid ALLOWED_CIDR", "err", err)
		os.Exit(1)
	}

	_, trustedNet, err := net.ParseCIDR(cfg.trustedProxyCIDR)
	if err != nil {
		slog.Error("invalid TRUSTED_PROXY_CIDR", "err", err)
		os.Exit(1)
	}

	target, err := url.Parse(cfg.upstreamURL)
	if err != nil {
		slog.Error("invalid UPSTREAM_URL", "err", err)
		os.Exit(1)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ModifyResponse = stripContactsNull
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		slog.Error("upstream error", "err", err)
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIP(r, trustedNet)
		if clientIP == nil || !allowedNet.Contains(clientIP) {
			slog.Warn("denied", "ip", clientIP)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
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
