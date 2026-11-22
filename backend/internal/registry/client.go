package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxOCIResponseBytes = 2 << 20 // 2 MiB cap for manifest/config JSON payloads

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

// NewClientWithHTTPClient creates a registry Client using a caller-supplied http.Client (primarily for tests).
func NewClientWithHTTPClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		return NewClient()
	}
	return &Client{httpClient: httpClient}
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

// ListTags queries the OCI Registry v2 API for all tags, following
// RFC 5988 Link headers for pagination (registries return at most 100 per page).
func (c *Client) ListTags(ctx context.Context, imageRef, username, password string) ([]string, error) {
	host, repo, _ := parseImageRef(imageRef)
	nextURL := fmt.Sprintf("https://%s/v2/%s/tags/list", host, repo)

	// Obtain a bearer token once via the first 401 challenge
	var bearerToken string

	var allTags []string
	for nextURL != "" {
		tags, next, token, err := c.fetchTagPage(ctx, nextURL, host, username, password, bearerToken)
		if err != nil {
			return nil, err
		}
		if token != "" {
			bearerToken = token // reuse for subsequent pages
		}
		allTags = append(allTags, tags...)
		nextURL = next
	}
	return allTags, nil
}

// fetchTagPage fetches a single page of tags and returns (tags, nextURL, bearerToken, error).
// nextURL is non-empty when the registry signals more pages via a Link header.
// bearerToken is returned (non-empty) only on the first page when auth is negotiated.
func (c *Client) fetchTagPage(ctx context.Context, pageURL, host, username, password, bearerToken string) ([]string, string, string, error) {
	resp, bearerToken, err := c.doAuthenticatedGET(ctx, pageURL, "", host, username, password, bearerToken)
	if err != nil {
		return nil, "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", "", fmt.Errorf("registry returned HTTP %d for %s", resp.StatusCode, pageURL)
	}

	var tagResp struct {
		Name string   `json:"name"`
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tagResp); err != nil {
		return nil, "", "", err
	}

	// Parse RFC 5988 Link header for the next page URL
	// Example: Link: </v2/repo/tags/list?last=tag100&n=100>; rel="next"
	nextURL := parseLinkNext(resp.Header.Get("Link"), host)

	return tagResp.Tags, nextURL, bearerToken, nil
}

type ociManifestEntry struct {
	Digest      string            `json:"digest"`
	MediaType   string            `json:"mediaType"`
	Annotations map[string]string `json:"annotations"`
	Platform    *struct {
		Architecture string `json:"architecture"`
		OS           string `json:"os"`
	} `json:"platform"`
}

type ociManifestDoc struct {
	MediaType   string            `json:"mediaType"`
	Annotations map[string]string `json:"annotations"`
	Config      *struct {
		Digest string `json:"digest"`
	} `json:"config"`
	Manifests []ociManifestEntry `json:"manifests"`
}

const (
	indexAndManifestAccept = "application/vnd.docker.distribution.manifest.v2+json, application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.oci.image.index.v1+json"
	singleManifestAccept   = "application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json"
	configBlobAccept       = "application/vnd.oci.image.config.v1+json, application/vnd.docker.container.image.v1+json, application/json, */*"
)

// FetchImageLabels retrieves OCI image config labels (and manifest annotation fallbacks)
// for imageRef, handling multi-arch indexes, Buildx attestations, and Bearer auth.
func (c *Client) FetchImageLabels(ctx context.Context, imageRef, username, password string) (map[string]string, error) {
	host, repo, tag := parseImageRef(imageRef)
	manifestURL := fmt.Sprintf("https://%s/v2/%s/manifests/%s", host, repo, tag)

	var doc ociManifestDoc
	bearerToken, err := c.fetchJSONWithAuth(ctx, manifestURL, indexAndManifestAccept, host, username, password, "", &doc)
	if err != nil {
		return nil, err
	}

	fallbackLabels := make(map[string]string)
	mergeLabels(fallbackLabels, doc.Annotations)

	var configDigest string
	if doc.Config != nil && doc.Config.Digest != "" {
		configDigest = doc.Config.Digest
	} else if len(doc.Manifests) > 0 {
		selected, ok := selectBestManifest(doc.Manifests)
		if ok {
			mergeLabels(fallbackLabels, selected.Annotations)
			childURL := fmt.Sprintf("https://%s/v2/%s/manifests/%s", host, repo, selected.Digest)
			var childDoc ociManifestDoc
			bearerToken, err = c.fetchJSONWithAuth(ctx, childURL, singleManifestAccept, host, username, password, bearerToken, &childDoc)
			if err != nil {
				return nil, err
			}
			mergeLabels(fallbackLabels, childDoc.Annotations)
			if childDoc.Config != nil {
				configDigest = childDoc.Config.Digest
			}
		}
	}

	if configDigest == "" {
		if len(fallbackLabels) > 0 {
			return fallbackLabels, nil
		}
		return nil, fmt.Errorf("missing config digest in manifest for %s", imageRef)
	}

	blobURL := fmt.Sprintf("https://%s/v2/%s/blobs/%s", host, repo, configDigest)
	var cfgDoc struct {
		Config struct {
			Labels map[string]string `json:"Labels"`
		} `json:"config"`
	}
	if _, err := c.fetchJSONWithAuth(ctx, blobURL, configBlobAccept, host, username, password, bearerToken, &cfgDoc); err != nil {
		return nil, err
	}

	labels := make(map[string]string, len(fallbackLabels)+len(cfgDoc.Config.Labels))
	mergeLabels(labels, fallbackLabels)
	mergeLabels(labels, cfgDoc.Config.Labels)
	return labels, nil
}

func (c *Client) fetchJSONWithAuth(ctx context.Context, reqURL, acceptHeader, host, username, password, bearerToken string, dest any) (string, error) {
	resp, token, err := c.doAuthenticatedGET(ctx, reqURL, acceptHeader, host, username, password, bearerToken)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("registry returned HTTP %d for %s", resp.StatusCode, reqURL)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxOCIResponseBytes))
	if err != nil {
		return "", fmt.Errorf("failed reading response from %s: %w", reqURL, err)
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return "", fmt.Errorf("failed decoding JSON from %s: %w", reqURL, err)
	}
	return token, nil
}

func (c *Client) doAuthenticatedGET(ctx context.Context, reqURL, acceptHeader, host, username, password, bearerToken string) (*http.Response, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, "", err
	}
	if acceptHeader != "" {
		req.Header.Set("Accept", acceptHeader)
	}
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed HTTP GET request to %s: %w", host, err)
	}

	if resp.StatusCode == http.StatusUnauthorized && bearerToken == "" {
		authHeader := resp.Header.Get("Www-Authenticate")
		resp.Body.Close()

		token, err := c.obtainToken(ctx, authHeader, username, password)
		if err != nil {
			return nil, "", fmt.Errorf("registry auth failed for %s: %w", host, err)
		}
		bearerToken = token

		req, err = http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, "", err
		}
		if acceptHeader != "" {
			req.Header.Set("Accept", acceptHeader)
		}
		req.Header.Set("Authorization", "Bearer "+bearerToken)

		resp, err = c.httpClient.Do(req)
		if err != nil {
			return nil, "", fmt.Errorf("failed authenticated HTTP GET request to %s: %w", host, err)
		}
	}

	return resp, bearerToken, nil
}

func selectBestManifest(manifests []ociManifestEntry) (ociManifestEntry, bool) {
	var valid []ociManifestEntry
	for _, m := range manifests {
		if m.Digest == "" {
			continue
		}
		if m.Annotations["vnd.docker.reference.type"] == "attestation-manifest" {
			continue
		}
		if m.Platform != nil && (m.Platform.OS == "unknown" || m.Platform.Architecture == "unknown") {
			continue
		}
		valid = append(valid, m)
	}
	if len(valid) == 0 {
		return ociManifestEntry{}, false
	}

	for _, m := range valid {
		if m.Platform != nil && m.Platform.OS == "linux" && m.Platform.Architecture == "amd64" {
			return m, true
		}
	}
	for _, m := range valid {
		if m.Platform != nil && m.Platform.OS == "linux" && m.Platform.Architecture == "arm64" {
			return m, true
		}
	}
	for _, m := range valid {
		if m.Platform != nil && m.Platform.OS == "linux" {
			return m, true
		}
	}
	return valid[0], true
}

func mergeLabels(dst, src map[string]string) {
	for k, v := range src {
		dst[k] = v
	}
}

// parseLinkNext extracts the absolute URL from a RFC 5988 Link header with rel="next".
func parseLinkNext(linkHeader, host string) string {
	if linkHeader == "" {
		return ""
	}
	// Header format: </v2/repo/tags/list?last=...&n=100>; rel="next"
	parts := strings.Split(linkHeader, ";")
	if len(parts) < 2 {
		return ""
	}
	rel := strings.TrimSpace(parts[1])
	if !strings.Contains(rel, `rel="next"`) {
		return ""
	}
	urlPart := strings.TrimSpace(parts[0])
	urlPart = strings.Trim(urlPart, "<>")
	if strings.HasPrefix(urlPart, "/") {
		return "https://" + host + urlPart
	}
	return urlPart
}


func setManifestHeaders(req *http.Request) {
	req.Header.Set("Accept", indexAndManifestAccept)
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
