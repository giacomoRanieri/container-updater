package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// FetchDigest queries the OCI Registry v2 API directly via HTTP/HTTPS
// to retrieve the remote image digest for a given image reference (e.g. "nginx:latest", "ghcr.io/user/repo:beta").
func (c *Client) FetchDigest(ctx context.Context, imageRef, username, password string) (string, error) {
	host, repo, tag := parseImageRef(imageRef)

	// Build HTTPS target URL
	manifestURL := fmt.Sprintf("https://%s/v2/%s/manifests/%s", host, repo, tag)

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, manifestURL, nil)
	if err != nil {
		return "", err
	}

	setManifestHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed HTTP request to registry %s: %w", host, err)
	}
	defer resp.Body.Close()

	// If 401 Unauthorized, perform OCI Bearer / Basic Auth token exchange
	if resp.StatusCode == http.StatusUnauthorized {
		authHeader := resp.Header.Get("Www-Authenticate")
		token, err := c.obtainToken(ctx, authHeader, username, password)
		if err != nil {
			return "", fmt.Errorf("registry auth failed for %s: %w", host, err)
		}

		// Retry HEAD request with Bearer token
		req, err = http.NewRequestWithContext(ctx, http.MethodHead, manifestURL, nil)
		if err != nil {
			return "", err
		}
		setManifestHeaders(req)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err = c.httpClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("failed authenticated HTTP request to registry %s: %w", host, err)
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("registry returned HTTP %d for %s", resp.StatusCode, manifestURL)
	}

	digest := resp.Header.Get("Docker-Content-Digest")
	if digest == "" {
		// Fallback GET request to calculate digest if header missing in HEAD
		return c.fetchDigestViaGet(ctx, manifestURL, username, password)
	}

	return strings.TrimPrefix(digest, "sha256:"), nil
}

func (c *Client) fetchDigestViaGet(ctx context.Context, manifestURL, username, password string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURL, nil)
	if err != nil {
		return "", err
	}
	setManifestHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		authHeader := resp.Header.Get("Www-Authenticate")
		token, err := c.obtainToken(ctx, authHeader, username, password)
		if err != nil {
			return "", err
		}
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, manifestURL, nil)
		if err != nil {
			return "", err
		}
		setManifestHeaders(req)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err = c.httpClient.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
	}

	digest := resp.Header.Get("Docker-Content-Digest")
	if digest == "" {
		return "", fmt.Errorf("empty Docker-Content-Digest header from %s", manifestURL)
	}
	return strings.TrimPrefix(digest, "sha256:"), nil
}

// ListTags queries the OCI Registry v2 API endpoint GET /v2/<name>/tags/list
func (c *Client) ListTags(ctx context.Context, imageRef, username, password string) ([]string, error) {
	host, repo, _ := parseImageRef(imageRef)
	tagsURL := fmt.Sprintf("https://%s/v2/%s/tags/list", host, repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tagsURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed HTTP request to list tags from %s: %w", host, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		authHeader := resp.Header.Get("Www-Authenticate")
		token, err := c.obtainToken(ctx, authHeader, username, password)
		if err != nil {
			return nil, fmt.Errorf("registry auth failed for %s: %w", host, err)
		}

		req, err = http.NewRequestWithContext(ctx, http.MethodGet, tagsURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err = c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed authenticated tag list request to %s: %w", host, err)
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned HTTP %d for %s", resp.StatusCode, tagsURL)
	}

	var tagResp struct {
		Name string   `json:"name"`
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tagResp); err != nil {
		return nil, err
	}

	return tagResp.Tags, nil
}

func setManifestHeaders(req *http.Request) {
	req.Header.Set("Accept", strings.Join([]string{
		"application/vnd.docker.distribution.manifest.v2+json",
		"application/vnd.oci.image.manifest.v1+json",
		"application/vnd.docker.distribution.manifest.list.v2+json",
		"application/vnd.oci.image.index.v1+json",
	}, ", "))
}

func (c *Client) obtainToken(ctx context.Context, wwwAuthHeader, username, password string) (string, error) {
	if wwwAuthHeader == "" {
		return "", fmt.Errorf("missing Www-Authenticate header")
	}

	if !strings.HasPrefix(wwwAuthHeader, "Bearer ") {
		return "", fmt.Errorf("unsupported auth scheme in %s", wwwAuthHeader)
	}

	params := parseHeaderParams(wwwAuthHeader[7:])
	realm := params["realm"]
	if realm == "" {
		return "", fmt.Errorf("missing realm in Www-Authenticate header")
	}

	tokenURL, err := url.Parse(realm)
	if err != nil {
		return "", err
	}

	q := tokenURL.Query()
	if service := params["service"]; service != "" {
		q.Set("service", service)
	}
	if scope := params["scope"]; scope != "" {
		q.Set("scope", scope)
	}
	tokenURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL.String(), nil)
	if err != nil {
		return "", err
	}

	if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint returned HTTP %d", resp.StatusCode)
	}

	var tokenResp struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	if tokenResp.Token != "" {
		return tokenResp.Token, nil
	}
	if tokenResp.AccessToken != "" {
		return tokenResp.AccessToken, nil
	}

	return "", fmt.Errorf("empty token received")
}

// parseHeaderParams parses a Bearer Www-Authenticate parameter string
// respecting RFC 7235 quoted strings (commas inside quotes are not delimiters).
func parseHeaderParams(header string) map[string]string {
	result := make(map[string]string)
	inQuote := false
	start := 0
	for i := 0; i < len(header); i++ {
		switch header[i] {
		case '"':
			inQuote = !inQuote
		case ',':
			if !inQuote {
				parseHeaderKV(result, strings.TrimSpace(header[start:i]))
				start = i + 1
			}
		}
	}
	parseHeaderKV(result, strings.TrimSpace(header[start:]))
	return result
}

func parseHeaderKV(m map[string]string, s string) {
	kv := strings.SplitN(s, "=", 2)
	if len(kv) == 2 {
		key := strings.TrimSpace(kv[0])
		val := strings.Trim(strings.TrimSpace(kv[1]), "\"")
		m[key] = val
	}
}

func parseImageRef(imageRef string) (host, repo, tag string) {
	tag = "latest"
	if idx := strings.LastIndex(imageRef, ":"); idx != -1 && !strings.Contains(imageRef[idx:], "/") {
		tag = imageRef[idx+1:]
		imageRef = imageRef[:idx]
	}

	parts := strings.Split(imageRef, "/")
	if len(parts) == 1 {
		host = "registry-1.docker.io"
		repo = "library/" + parts[0]
	} else if !strings.Contains(parts[0], ".") && !strings.Contains(parts[0], ":") && parts[0] != "localhost" {
		host = "registry-1.docker.io"
		repo = imageRef
	} else {
		host = parts[0]
		repo = strings.Join(parts[1:], "/")
		if host == "docker.io" || host == "index.docker.io" {
			host = "registry-1.docker.io"
		}
	}

	return host, repo, tag
}
