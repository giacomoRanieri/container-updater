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
