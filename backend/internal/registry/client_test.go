package registry

import (
	"reflect"
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
