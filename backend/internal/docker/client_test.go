package docker

import "testing"

func TestResolveDockerTrackTag(t *testing.T) {
	if got := resolveDockerTrackTag(nil); got != nil {
		t.Fatalf("expected nil for nil labels, got %q", *got)
	}
	if got := resolveDockerTrackTag(map[string]string{"container-updater.track-tag": "  "}); got != nil {
		t.Fatalf("expected nil for blank label, got %q", *got)
	}
	got := resolveDockerTrackTag(map[string]string{"container-updater.track-tag": " stable "})
	if got == nil || *got != "stable" {
		t.Fatalf("expected 'stable', got %v", got)
	}
}
