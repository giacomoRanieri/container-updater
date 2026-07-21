package k8s

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindAndEditManifestInWorkspaceRoot(t *testing.T) {
	tempDir := t.TempDir()
	workspaceDir := filepath.Join(tempDir, "workspace")
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		t.Fatalf("create workspace dir: %v", err)
	}

	manifestPath := filepath.Join(workspaceDir, "deployment.yaml")
	manifestContent := []byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: demo
spec:
  template:
    spec:
      containers:
        - name: app
          image: nginx:1.25.0
`)
	if err := os.WriteFile(manifestPath, manifestContent, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	found, err := findAndEditManifestInRoots([]string{workspaceDir}, "Deployment", "demo", "app", "nginx:1.25.1")
	if err != nil {
		t.Fatalf("edit manifest: %v", err)
	}
	if !found {
		t.Fatalf("expected manifest to be updated")
	}

	updated, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read updated manifest: %v", err)
	}
	if string(updated) == string(manifestContent) {
		t.Fatalf("expected manifest content to change")
	}
	if !containsString(string(updated), "nginx:1.25.1") {
		t.Fatalf("expected updated image in manifest, got %s", string(updated))
	}
}

func containsString(haystack, needle string) bool {
	return len(needle) == 0 || (len(haystack) >= len(needle) && (haystack == needle || len(haystack) > len(needle) && (containsString(haystack[1:], needle) || haystack[:len(needle)] == needle)))
}
