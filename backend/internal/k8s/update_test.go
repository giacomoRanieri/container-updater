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

<<<<<<< feat/k8s-async-rollout-monitoring-ws-events
func TestFindAndEditManifest_MultiDocument(t *testing.T) {
=======
func TestFindAndEditManifest_PreserveIndentationAndComments(t *testing.T) {
>>>>>>> local
	tempDir := t.TempDir()
	workspaceDir := filepath.Join(tempDir, "workspace")
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		t.Fatalf("create workspace dir: %v", err)
	}

<<<<<<< feat/k8s-async-rollout-monitoring-ws-events
	manifestPath := filepath.Join(workspaceDir, "zigbee2mqtt.yaml")
	manifestContent := []byte(`apiVersion: v1
kind: Service
metadata:
  name: zigbee2mqtt
spec:
  ports:
    - port: 8080
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: zigbee2mqtt
spec:
  template:
    spec:
      containers:
        - name: zigbee2mqtt
          image: koenkk/zigbee2mqtt:1.35.0
=======
	manifestPath := filepath.Join(workspaceDir, "custom-indent.yaml")
	// 4-space indentation and comments
	manifestContent := []byte(`# Top level comment
apiVersion: apps/v1
kind: Deployment
metadata:
    name: custom-app
spec:
    template:
        spec:
            containers:
                - name: app
                  image: redis:7.0.0
>>>>>>> local
`)
	if err := os.WriteFile(manifestPath, manifestContent, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

<<<<<<< feat/k8s-async-rollout-monitoring-ws-events
	found, err := findAndEditManifestInRoots([]string{workspaceDir}, "Deployment", "zigbee2mqtt", "zigbee2mqtt", "koenkk/zigbee2mqtt:1.36.0")
	if err != nil {
		t.Fatalf("edit multi-doc manifest: %v", err)
	}
	if !found {
		t.Fatalf("expected multi-doc manifest to be updated")
=======
	found, err := findAndEditManifestInRoots([]string{workspaceDir}, "Deployment", "custom-app", "app", "redis:7.2.0")
	if err != nil {
		t.Fatalf("edit manifest: %v", err)
	}
	if !found {
		t.Fatalf("expected manifest to be updated")
>>>>>>> local
	}

	updated, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read updated manifest: %v", err)
	}

	updatedStr := string(updated)
<<<<<<< feat/k8s-async-rollout-monitoring-ws-events
	if !containsString(updatedStr, "kind: Service") {
		t.Fatalf("expected Service document to be preserved in multi-doc file, got: %s", updatedStr)
	}
	if !containsString(updatedStr, "kind: Deployment") {
		t.Fatalf("expected Deployment document to be present in multi-doc file, got: %s", updatedStr)
	}
	if !containsString(updatedStr, "koenkk/zigbee2mqtt:1.36.0") {
		t.Fatalf("expected updated container image in multi-doc file, got: %s", updatedStr)
=======
	if !containsString(updatedStr, "redis:7.2.0") {
		t.Fatalf("expected updated container image in manifest, got: %s", updatedStr)
	}
	if !containsString(updatedStr, "Top level comment") {
		t.Fatalf("expected comments to be preserved in AST node edit, got: %s", updatedStr)
	}
	if !containsString(updatedStr, "    name: custom-app") {
		t.Fatalf("expected 4-space indentation to be preserved, got: %s", updatedStr)
>>>>>>> local
	}
}
