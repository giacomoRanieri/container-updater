package registry

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestParseHeaderParams_DockerHub(t *testing.T) {
	// Real Www-Authenticate header from Docker Hub
	header := `realm="https://auth.docker.io/token",service="registry.docker.io",scope="repository:home-assistant/home-assistant:pull"`
	// Strip leading "Bearer "
	params := parseHeaderParams(header)

	want := map[string]string{
		"realm":   "https://auth.docker.io/token",
		"service": "registry.docker.io",
		"scope":   "repository:home-assistant/home-assistant:pull",
	}

	if !reflect.DeepEqual(params, want) {
		t.Errorf("parseHeaderParams() = %v, want %v", params, want)
	}
}

func TestParseHeaderParams_SimpleNoQuotes(t *testing.T) {
	header := `realm="https://ghcr.io/token",service="ghcr.io"`
	params := parseHeaderParams(header)

	if params["realm"] != "https://ghcr.io/token" {
		t.Errorf("realm = %q, want %q", params["realm"], "https://ghcr.io/token")
	}
	if params["service"] != "ghcr.io" {
		t.Errorf("service = %q, want %q", params["service"], "ghcr.io")
	}
}

func TestParseHeaderParams_ScopeWithColons(t *testing.T) {
	// Scope with multiple colons inside quotes must not be split
	header := `realm="https://auth.example.com/token",service="example.com",scope="repository:foo/bar:pull,push"`
	params := parseHeaderParams(header)

	if params["scope"] != "repository:foo/bar:pull,push" {
		t.Errorf("scope = %q, want %q", params["scope"], "repository:foo/bar:pull,push")
	}
}

func TestParseLinkNext_ValidRelative(t *testing.T) {
	link := `</v2/home-assistant/home-assistant/tags/list?last=2021.6.0&n=100>; rel="next"`
	got := parseLinkNext(link, "ghcr.io")
	want := "https://ghcr.io/v2/home-assistant/home-assistant/tags/list?last=2021.6.0&n=100"
	if got != want {
		t.Errorf("parseLinkNext() = %q, want %q", got, want)
	}
}

func TestParseLinkNext_ValidAbsolute(t *testing.T) {
	link := `<https://ghcr.io/v2/home-assistant/home-assistant/tags/list?last=tag100&n=100>; rel="next"`
	got := parseLinkNext(link, "ghcr.io")
	want := "https://ghcr.io/v2/home-assistant/home-assistant/tags/list?last=tag100&n=100"
	if got != want {
		t.Errorf("parseLinkNext() = %q, want %q", got, want)
	}
}

func TestParseLinkNext_Empty(t *testing.T) {
	if got := parseLinkNext("", "ghcr.io"); got != "" {
		t.Errorf("parseLinkNext(\"\") = %q, want empty", got)
	}
}

func TestParseLinkNext_NoNextRel(t *testing.T) {
	link := `</v2/repo/tags/list?last=foo&n=100>; rel="last"`
	if got := parseLinkNext(link, "ghcr.io"); got != "" {
		t.Errorf("parseLinkNext with rel=last = %q, want empty", got)
	}
}

type rewriteHostTransport struct {
	targets map[string]*url.URL
	base    http.RoundTripper
}

func (rt *rewriteHostTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	if target, ok := rt.targets[req.URL.Host]; ok {
		cloned.URL.Scheme = target.Scheme
		cloned.URL.Host = target.Host
	}
	return rt.base.RoundTrip(cloned)
}

func newTestRegistryClient(t *testing.T, routes map[string]http.HandlerFunc) (*Client, func()) {
	t.Helper()
	servers := make(map[string]*httptest.Server)
	targets := make(map[string]*url.URL)
	for host, handler := range routes {
		srv := httptest.NewServer(handler)
		servers[host] = srv
		u, err := url.Parse(srv.URL)
		if err != nil {
			t.Fatalf("parse server url: %v", err)
		}
		targets[host] = u
	}

	httpClient := &http.Client{
		Transport: &rewriteHostTransport{
			targets: targets,
			base:    http.DefaultTransport,
		},
	}
	cleanup := func() {
		for _, srv := range servers {
			srv.Close()
		}
	}
	return NewClientWithHTTPClient(httpClient), cleanup
}

func TestFetchImageLabels_MultiArchWithAttestationAndBearerAuth(t *testing.T) {
	const (
		tokenValue    = "secret-registry-token"
		childDigest   = "sha256:amd64manifest"
		configDigest  = "sha256:configblob"
		attestDigest  = "sha256:attestationmanifest"
	)

	var blobSawAuth bool
	client, cleanup := newTestRegistryClient(t, map[string]http.HandlerFunc{
		"registry.test": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/token" {
				json.NewEncoder(w).Encode(map[string]string{"token": tokenValue})
				return
			}

			if r.Header.Get("Authorization") != "Bearer "+tokenValue {
				w.Header().Set("Www-Authenticate", `Bearer realm="https://registry.test/token",service="registry.test",scope="repository:n8nio/n8n:pull"`)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			switch r.URL.Path {
			case "/v2/n8nio/n8n/manifests/stable":
				json.NewEncoder(w).Encode(map[string]any{
					"mediaType": "application/vnd.oci.image.index.v1+json",
					"manifests": []any{
						map[string]any{
							"digest": attestDigest,
							"annotations": map[string]string{
								"vnd.docker.reference.type": "attestation-manifest",
							},
							"platform": map[string]string{
								"os":           "unknown",
								"architecture": "unknown",
							},
						},
						map[string]any{
							"digest": childDigest,
							"platform": map[string]string{
								"os":           "linux",
								"architecture": "amd64",
							},
						},
					},
				})
			case "/v2/n8nio/n8n/manifests/" + childDigest:
				json.NewEncoder(w).Encode(map[string]any{
					"mediaType": "application/vnd.oci.image.manifest.v1+json",
					"config": map[string]string{
						"digest": configDigest,
					},
				})
			case "/v2/n8nio/n8n/blobs/" + configDigest:
				http.Redirect(w, r, "https://cdn.test/blobs/"+configDigest, http.StatusTemporaryRedirect)
			default:
				http.NotFound(w, r)
			}
		},
		"cdn.test": func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "" {
				blobSawAuth = true
			}
			json.NewEncoder(w).Encode(map[string]any{
				"config": map[string]any{
					"Labels": map[string]string{
						"org.opencontainers.image.version": "2.48.7",
						"org.opencontainers.image.source":  "https://github.com/n8n-io/n8n",
					},
				},
			})
		},
	})
	defer cleanup()

	labels, err := client.FetchImageLabels(context.Background(), "registry.test/n8nio/n8n:stable", "", "")
	if err != nil {
		t.Fatalf("FetchImageLabels failed: %v", err)
	}
	if blobSawAuth {
		t.Errorf("expected Authorization header to be stripped on cross-host CDN redirect")
	}
	if labels["org.opencontainers.image.version"] != "2.48.7" {
		t.Errorf("version label = %q, want %q", labels["org.opencontainers.image.version"], "2.48.7")
	}
	if !strings.Contains(labels["org.opencontainers.image.source"], "n8n-io/n8n") {
		t.Errorf("unexpected source label: %q", labels["org.opencontainers.image.source"])
	}
}

func TestFetchImageLabels_SingleArchManifest(t *testing.T) {
	const configDigest = "sha256:singleconfig"
	client, cleanup := newTestRegistryClient(t, map[string]http.HandlerFunc{
		"registry.test": func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/v2/app/demo/manifests/latest":
				json.NewEncoder(w).Encode(map[string]any{
					"mediaType": "application/vnd.oci.image.manifest.v1+json",
					"config":    map[string]string{"digest": configDigest},
				})
			case "/v2/app/demo/blobs/" + configDigest:
				json.NewEncoder(w).Encode(map[string]any{
					"config": map[string]any{
						"Labels": map[string]string{
							"org.opencontainers.image.version": "1.2.3",
						},
					},
				})
			default:
				http.NotFound(w, r)
			}
		},
	})
	defer cleanup()

	labels, err := client.FetchImageLabels(context.Background(), "registry.test/app/demo:latest", "", "")
	if err != nil {
		t.Fatalf("FetchImageLabels failed: %v", err)
	}
	if labels["org.opencontainers.image.version"] != "1.2.3" {
		t.Errorf("version label = %q, want %q", labels["org.opencontainers.image.version"], "1.2.3")
	}
}

func TestFetchImageLabels_MissingVersionLabel(t *testing.T) {
	const configDigest = "sha256:noversioncfg"
	client, cleanup := newTestRegistryClient(t, map[string]http.HandlerFunc{
		"registry.test": func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/v2/app/plain/manifests/stable":
				json.NewEncoder(w).Encode(map[string]any{
					"config": map[string]string{"digest": configDigest},
				})
			case "/v2/app/plain/blobs/" + configDigest:
				json.NewEncoder(w).Encode(map[string]any{
					"config": map[string]any{
						"Labels": map[string]string{
							"maintainer": "example",
						},
					},
				})
			default:
				http.NotFound(w, r)
			}
		},
	})
	defer cleanup()

	labels, err := client.FetchImageLabels(context.Background(), "registry.test/app/plain:stable", "", "")
	if err != nil {
		t.Fatalf("expected no error when version label absent, got: %v", err)
	}
	if _, ok := labels["org.opencontainers.image.version"]; ok {
		t.Errorf("expected no version label, got %v", labels)
	}
	if labels["maintainer"] != "example" {
		t.Errorf("maintainer label = %q, want %q", labels["maintainer"], "example")
	}
}
