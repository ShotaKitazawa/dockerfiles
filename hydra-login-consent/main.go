package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type appConfig struct {
	hydraAdminURL     string
	auth0ClientID     string
	auth0ClientSecret string
	auth0CallbackURL  string
	listenAddr        string
}

func loadConfig() appConfig {
	return appConfig{
		hydraAdminURL:     getenv("HYDRA_ADMIN_URL", "http://localhost:4445"),
		auth0ClientID:     mustEnv("AUTH0_CLIENT_ID"),
		auth0ClientSecret: mustEnv("AUTH0_CLIENT_SECRET"),
		auth0CallbackURL:  mustEnv("AUTH0_CALLBACK_URL"),
		listenAddr:        getenv("LISTEN_ADDR", ":8080"),
	}
}

func main() {
	cfg := loadConfig()

	auth0Domain := mustEnv("AUTH0_DOMAIN")
	auth0Issuer := "https://" + auth0Domain + "/"

	provider, err := oidc.NewProvider(context.Background(), auth0Issuer)
	if err != nil {
		slog.Error("create oidc provider", "err", err)
		os.Exit(1)
	}

	oauth2Cfg := &oauth2.Config{
		ClientID:     cfg.auth0ClientID,
		ClientSecret: cfg.auth0ClientSecret,
		RedirectURL:  cfg.auth0CallbackURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "email"},
	}
	idTokenVerifier := provider.Verifier(&oidc.Config{ClientID: cfg.auth0ClientID})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /login", loginHandler(cfg, oauth2Cfg))
	mux.HandleFunc("GET /login/callback", loginCallbackHandler(cfg, oauth2Cfg, idTokenVerifier))
	mux.HandleFunc("GET /consent", consentHandler(cfg))
	mux.HandleFunc("GET /error", errorHandler())
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	slog.Info("starting", "addr", cfg.listenAddr)
	if err := http.ListenAndServe(cfg.listenAddr, mux); err != nil {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}

func loginHandler(cfg appConfig, oauth2Cfg *oauth2.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		challenge := r.URL.Query().Get("login_challenge")
		if challenge == "" {
			http.Error(w, "missing login_challenge", http.StatusBadRequest)
			return
		}

		loginReq, err := hydraGet(cfg.hydraAdminURL+"/admin/oauth2/auth/requests/login", "login_challenge", challenge)
		if err != nil {
			slog.Error("get login request", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		// Already authenticated: skip login UI and accept immediately.
		if skip, _ := loginReq["skip"].(bool); skip {
			subject, _ := loginReq["subject"].(string)
			redirectTo, err := hydraAccept(cfg.hydraAdminURL+"/admin/oauth2/auth/requests/login/accept", "login_challenge", challenge, map[string]any{
				"subject":      subject,
				"remember":     false,
				"remember_for": 0,
			})
			if err != nil {
				slog.Error("accept login request (skip)", "err", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			http.Redirect(w, r, redirectTo, http.StatusFound)
			return
		}

		// Use login_challenge as OAuth2 state. It is a one-time server-generated
		// value, so it serves as a CSRF token without needing a session store.
		http.Redirect(w, r, oauth2Cfg.AuthCodeURL(challenge), http.StatusFound)
	}
}

func loginCallbackHandler(cfg appConfig, oauth2Cfg *oauth2.Config, verifier *oidc.IDTokenVerifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		challenge := r.URL.Query().Get("state")
		code := r.URL.Query().Get("code")
		if challenge == "" || code == "" {
			http.Error(w, "missing state or code", http.StatusBadRequest)
			return
		}

		token, err := oauth2Cfg.Exchange(r.Context(), code)
		if err != nil {
			slog.Error("exchange code", "err", err)
			http.Error(w, "token exchange failed", http.StatusInternalServerError)
			return
		}

		rawIDToken, ok := token.Extra("id_token").(string)
		if !ok {
			http.Error(w, "missing id_token", http.StatusInternalServerError)
			return
		}

		idToken, err := verifier.Verify(r.Context(), rawIDToken)
		if err != nil {
			slog.Error("verify id token", "err", err)
			http.Error(w, "invalid id token", http.StatusInternalServerError)
			return
		}

		var claims struct {
			Email string `json:"email"`
			Sub   string `json:"sub"`
		}
		if err := idToken.Claims(&claims); err != nil {
			slog.Error("parse claims", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		subject := claims.Email
		if subject == "" {
			subject = claims.Sub
		}

		redirectTo, err := hydraAccept(cfg.hydraAdminURL+"/admin/oauth2/auth/requests/login/accept", "login_challenge", challenge, map[string]any{
			"subject":      subject,
			"remember":     false,
			"remember_for": 0,
		})
		if err != nil {
			slog.Error("accept login request", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, redirectTo, http.StatusFound)
	}
}

func consentHandler(cfg appConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		challenge := r.URL.Query().Get("consent_challenge")
		if challenge == "" {
			http.Error(w, "missing consent_challenge", http.StatusBadRequest)
			return
		}

		consentReq, err := hydraGet(cfg.hydraAdminURL+"/admin/oauth2/auth/requests/consent", "consent_challenge", challenge)
		if err != nil {
			slog.Error("get consent request", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		scopes := toStringSlice(consentReq["requested_scope"])
		audience := toStringSlice(consentReq["requested_access_token_audience"])

		redirectTo, err := hydraAccept(cfg.hydraAdminURL+"/admin/oauth2/auth/requests/consent/accept", "consent_challenge", challenge, map[string]any{
			"grant_scope":                  scopes,
			"grant_access_token_audience":  audience,
			"remember":                     false,
			"remember_for":                 0,
		})
		if err != nil {
			slog.Error("accept consent request", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, redirectTo, http.StatusFound)
	}
}

func errorHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		errMsg := r.URL.Query().Get("error")
		errDesc := r.URL.Query().Get("error_description")
		slog.Error("hydra error", "error", errMsg, "description", errDesc)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "OAuth2 error: %s\n%s\n", errMsg, errDesc)
	}
}

// hydraGet calls a Hydra Admin GET endpoint with the given challenge query param.
func hydraGet(endpoint, challengeParam, challenge string) (map[string]any, error) {
	u := endpoint + "?" + challengeParam + "=" + url.QueryEscape(challenge)
	resp, err := http.Get(u) //nolint:noctx
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("hydra returned %d", resp.StatusCode)
	}
	var result map[string]any
	return result, json.NewDecoder(resp.Body).Decode(&result)
}

// hydraAccept calls a Hydra Admin PUT accept endpoint and returns redirect_to.
func hydraAccept(endpoint, challengeParam, challenge string, body map[string]any) (string, error) {
	b, _ := json.Marshal(body)
	u := endpoint + "?" + challengeParam + "=" + url.QueryEscape(challenge)
	req, _ := http.NewRequest(http.MethodPut, u, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("hydra returned %d", resp.StatusCode)
	}
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	redirectTo, ok := result["redirect_to"].(string)
	if !ok {
		return "", fmt.Errorf("no redirect_to in response: %v", result)
	}
	return redirectTo, nil
}

func toStringSlice(v any) []string {
	arr, _ := v.([]any)
	result := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("required env var not set", "key", key)
		os.Exit(1)
	}
	return v
}
