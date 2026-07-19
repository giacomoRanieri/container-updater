package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"container-updater/backend/internal/config"
	"container-updater/backend/internal/logger"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

func getOAuth2Config(ctx context.Context) (*oauth2.Config, *oidc.IDTokenVerifier, error) {
	if !config.GlobalConfig.OIDC.Enabled {
		return nil, nil, nil
	}

	provider, err := oidc.NewProvider(ctx, config.GlobalConfig.OIDC.Issuer)
	if err != nil {
		return nil, nil, err
	}

	oauth2Config := &oauth2.Config{
		ClientID:     config.GlobalConfig.OIDC.ClientID,
		ClientSecret: config.GlobalConfig.OIDC.ClientSecret,
		RedirectURL:  config.GlobalConfig.OIDC.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: config.GlobalConfig.OIDC.ClientID})
	return oauth2Config, verifier, nil
}

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// If OIDC is disabled, bypass and redirect directly to mock callback
	if !config.GlobalConfig.OIDC.Enabled {
		logger.Log.Info("OIDC disabled, initiating mock login redirect")
		http.Redirect(w, r, "/api/auth/callback?code=mock-code&state=mock-state", http.StatusTemporaryRedirect)
		return
	}

	oauth2Config, _, err := getOAuth2Config(ctx)
	if err != nil {
		logger.Log.Error("failed to build OIDC configuration", "error", err)
		http.Error(w, "Authentication configuration error", http.StatusInternalServerError)
		return
	}

	// Generate state token
	b := make([]byte, 16)
	rand.Read(b)
	state := hex.EncodeToString(b)

	// Save state token in short-lived secure cookie
	cookie := &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		Expires:  time.Now().Add(10 * time.Minute),
		HttpOnly: true,
		Secure:   false, // Set true in production if TLS is enabled
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)

	authURL := oauth2Config.AuthCodeURL(state)
	logger.Log.Info("redirecting to OIDC provider login page", "url", authURL)
	http.Redirect(w, r, authURL, http.StatusFound)
}

func HandleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	code := r.FormValue("code")
	state := r.FormValue("state")

	// 1. Mock Login Mode (OIDC Disabled)
	if !config.GlobalConfig.OIDC.Enabled {
		logger.Log.Info("OIDC disabled, processing mock login callback")
		token, err := GenerateToken("usr-mock-admin", "admin", "admin@container-updater.local")
		if err != nil {
			logger.Log.Error("failed to generate mock session token", "error", err)
			http.Error(w, "Mock auth error", http.StatusInternalServerError)
			return
		}

		setSessionCookie(w, token)
		
		// Redirect back to frontend homepage (Next.js)
		frontendURL := getFrontendRedirectURL(r)
		logger.Log.Info("Mock auth successful, redirecting to frontend", "url", frontendURL)
		http.Redirect(w, r, frontendURL, http.StatusFound)
		return
	}

	// 2. Standard OIDC Mode
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil || stateCookie.Value != state {
		logger.Log.Error("OIDC callback validation failed: missing or mismatched state cookie")
		http.Error(w, "Mismatched state parameter", http.StatusBadRequest)
		return
	}

	// Clear state cookie
	http.SetCookie(w, &http.Cookie{
		Name:   "oauth_state",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	oauth2Config, verifier, err := getOAuth2Config(ctx)
	if err != nil {
		logger.Log.Error("failed to reload OIDC configuration during callback", "error", err)
		http.Error(w, "Authentication error", http.StatusInternalServerError)
		return
	}

	// Exchange authorization code for token
	oauth2Token, err := oauth2Config.Exchange(ctx, code)
	if err != nil {
		logger.Log.Error("failed to exchange code for OIDC token", "error", err)
		http.Error(w, "Failed to exchange code", http.StatusInternalServerError)
		return
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		logger.Log.Error("OIDC token did not contain an id_token")
		http.Error(w, "Missing ID token", http.StatusInternalServerError)
		return
	}

	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		logger.Log.Error("failed to verify raw OIDC ID token", "error", err)
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Extract standard OIDC claims
	var claims struct {
		Subject  string `json:"sub"`
		Name     string `json:"preferred_username"`
		Email    string `json:"email"`
	}
	if err := idToken.Claims(&claims); err != nil {
		logger.Log.Error("failed to parse OIDC claims", "error", err)
		http.Error(w, "Failed to extract claims", http.StatusInternalServerError)
		return
	}

	if claims.Name == "" {
		claims.Name = claims.Email
	}

	// Generate session JWT token
	token, err := GenerateToken(claims.Subject, claims.Name, claims.Email)
	if err != nil {
		logger.Log.Error("failed to generate session token", "user", claims.Name, "error", err)
		http.Error(w, "Token generation failed", http.StatusInternalServerError)
		return
	}

	setSessionCookie(w, token)

	// Redirect to frontend
	frontendURL := getFrontendRedirectURL(r)
	logger.Log.Info("OIDC authentication successful", "user", claims.Name, "redirect", frontendURL)
	http.Redirect(w, r, frontendURL, http.StatusFound)
}

func getFrontendRedirectURL(r *http.Request) string {
	// 1. Explicit environment variable: FRONTEND_URL
	if feURL := config.GlobalConfig.FrontendURL; feURL != "" {
		return feURL
	}

	// 2. Derive base host from Referer header if present
	if ref := r.Header.Get("Referer"); ref != "" {
		if parsed, err := url.Parse(ref); err == nil && parsed.Scheme != "" && parsed.Host != "" {
			return fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
		}
	}

	// 3. Fallback to request scheme and Host header
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	if r.Host != "" {
		return fmt.Sprintf("%s://%s", scheme, r.Host)
	}

	// 4. Default fallback
	return "http://localhost:3000"
}

func HandleSession(w http.ResponseWriter, r *http.Request) {
	// Retrieve claims from context injected by RequireAuth middleware
	val := r.Context().Value(UserContextKey)
	if val == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"authenticated": false})
		return
	}

	claims, ok := val.(*UserClaims)
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"authenticated": false})
		return
	}

	response := map[string]any{
		"authenticated": true,
		"user": map[string]string{
			"id":       claims.UserID,
			"username": claims.Username,
			"email":    claims.Email,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func setSessionCookie(w http.ResponseWriter, token string) {
	cookie := &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Secure:   false, // Set true in production over HTTPS
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
}
