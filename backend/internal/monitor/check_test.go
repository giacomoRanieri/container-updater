package monitor

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"container-updater/backend/internal/db"
)

type fakeRegistryAPI struct {
	digests       map[string]string
	digestErrs    map[string]error
	tags          map[string][]string
	listTagsErr   error
	labels        map[string]map[string]string
	labelsErr     error
	listTagsCalls int
}

func (f *fakeRegistryAPI) FetchDigest(ctx context.Context, imageRef, username, password string) (string, error) {
	if err, ok := f.digestErrs[imageRef]; ok {
		return "", err
	}
	if d, ok := f.digests[imageRef]; ok {
		return d, nil
	}
	return "", fmt.Errorf("digest not found for %s", imageRef)
}

func (f *fakeRegistryAPI) ListTags(ctx context.Context, imageRef, username, password string) ([]string, error) {
	f.listTagsCalls++
	if f.listTagsErr != nil {
		return nil, f.listTagsErr
	}
	if t, ok := f.tags[imageRef]; ok {
		return t, nil
	}
	repo, _ := splitImageRepoTag(imageRef)
	if t, ok := f.tags[repo]; ok {
		return t, nil
	}
	return nil, errors.New("tags not found")
}

func (f *fakeRegistryAPI) FetchImageLabels(ctx context.Context, imageRef, username, password string) (map[string]string, error) {
	if f.labelsErr != nil {
		return nil, f.labelsErr
	}
	if l, ok := f.labels[imageRef]; ok {
		return l, nil
	}
	return map[string]string{}, nil
}

func strPtr(s string) *string {
	return &s
}

func TestCheckWorkloadWithRegistry_TrackedChannelUpToDateSkipsListTags(t *testing.T) {
	fake := &fakeRegistryAPI{
		digests: map[string]string{
			"n8nio/n8n:2.48.7": "sha256:stable2487",
			"n8nio/n8n:stable": "sha256:stable2487",
			"n8nio/n8n:2.49.3": "sha256:next2493",
		},
		tags: map[string][]string{
			"n8nio/n8n": {"2.48.7", "2.49.3", "stable", "next"},
		},
	}

	w := &db.Workload{
		ID:            "k8s:default:deploy:n8n:n8n",
		Name:          "n8n",
		CurrentImage:  "n8nio/n8n:2.48.7",
		CurrentDigest: "sha256:stable2487",
		TrackTag:      strPtr("stable"),
	}

	updated, err := checkWorkloadWithRegistry(context.Background(), fake, nil, w, "docker.io", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated {
		t.Fatalf("expected no update when stable digest matches 2.48.7, got NewImage=%v", w.NewImage)
	}
	if w.UpdateStatus != "up_to_date" {
		t.Errorf("UpdateStatus = %q, want up_to_date", w.UpdateStatus)
	}
	if fake.listTagsCalls != 0 {
		t.Errorf("expected 0 ListTags calls when tracked channel matches, got %d", fake.listTagsCalls)
	}
}

func TestCheckWorkloadWithRegistry_TrackedChannelResolvesOCILabel(t *testing.T) {
	fake := &fakeRegistryAPI{
		digests: map[string]string{
			"n8nio/n8n:2.48.7": "sha256:old2487",
			"n8nio/n8n:stable": "sha256:new2488",
			"n8nio/n8n:2.48.8": "sha256:new2488",
		},
		labels: map[string]map[string]string{
			"n8nio/n8n:stable": {
				"org.opencontainers.image.version": "2.48.8",
			},
		},
	}

	w := &db.Workload{
		ID:            "k8s:default:deploy:n8n:n8n",
		Name:          "n8n",
		CurrentImage:  "n8nio/n8n:2.48.7",
		CurrentDigest: "sha256:old2487",
		TrackTag:      strPtr("stable"),
	}

	updated, err := checkWorkloadWithRegistry(context.Background(), fake, nil, w, "docker.io", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated || w.NewImage == nil || *w.NewImage != "n8nio/n8n:2.48.8" {
		t.Fatalf("expected NewImage 'n8nio/n8n:2.48.8', got updated=%v NewImage=%v", updated, w.NewImage)
	}
	if w.NewDigest == nil || *w.NewDigest != "new2488" {
		t.Errorf("NewDigest = %v, want new2488", w.NewDigest)
	}
	if fake.listTagsCalls != 0 {
		t.Errorf("expected 0 ListTags calls when OCI label resolves version, got %d", fake.listTagsCalls)
	}
}

func TestCheckWorkloadWithRegistry_TrackedChannelLinuxServerRevision(t *testing.T) {
	fake := &fakeRegistryAPI{
		digests: map[string]string{
			"lscr.io/linuxserver/wireguard:1.0.20260223-r0-ls117": "sha256:ls117",
			"lscr.io/linuxserver/wireguard:latest":                "sha256:ls130",
			"lscr.io/linuxserver/wireguard:1.0.20260223-r0-ls130": "sha256:ls130",
		},
		labels: map[string]map[string]string{
			"lscr.io/linuxserver/wireguard:latest": {
				"org.opencontainers.image.version": "1.0.20260223-r0-ls130",
			},
		},
	}

	w := &db.Workload{
		ID:            "k8s:default:deploy:wireguard:wireguard",
		Name:          "wireguard",
		CurrentImage:  "lscr.io/linuxserver/wireguard:1.0.20260223-r0-ls117",
		CurrentDigest: "sha256:ls117",
		TrackTag:      strPtr("latest"),
	}

	updated, err := checkWorkloadWithRegistry(context.Background(), fake, nil, w, "lscr.io", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated || w.NewImage == nil || *w.NewImage != "lscr.io/linuxserver/wireguard:1.0.20260223-r0-ls130" {
		t.Fatalf("expected ls130 update, got updated=%v NewImage=%v", updated, w.NewImage)
	}
}

func TestCheckWorkloadWithRegistry_TrackedChannelDowngradeProtection(t *testing.T) {
	fake := &fakeRegistryAPI{
		digests: map[string]string{
			"n8nio/n8n:2.49.3": "sha256:next2493",
			"n8nio/n8n:stable": "sha256:stable2487",
			"n8nio/n8n:2.48.7": "sha256:stable2487",
		},
		labels: map[string]map[string]string{
			"n8nio/n8n:stable": {
				"org.opencontainers.image.version": "2.48.7",
			},
		},
	}

	w := &db.Workload{
		ID:            "k8s:default:deploy:n8n:n8n",
		Name:          "n8n",
		CurrentImage:  "n8nio/n8n:2.49.3",
		CurrentDigest: "sha256:next2493",
		TrackTag:      strPtr("stable"),
	}

	updated, err := checkWorkloadWithRegistry(context.Background(), fake, nil, w, "docker.io", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated {
		t.Fatalf("expected downgrade to be blocked, got NewImage=%v", w.NewImage)
	}
	if w.UpdateStatus != "up_to_date" {
		t.Errorf("UpdateStatus = %q, want up_to_date", w.UpdateStatus)
	}
}

func TestCheckWorkloadWithRegistry_TrackedChannelFallbackListTagsDigestMatch(t *testing.T) {
	fake := &fakeRegistryAPI{
		digests: map[string]string{
			"app/demo:1.0.0":  "sha256:old100",
			"app/demo:stable": "sha256:new110",
			"app/demo:1.2.0":  "sha256:next120",
			"app/demo:1.1.0":  "sha256:new110",
		},
		labels: map[string]map[string]string{
			"app/demo:stable": {}, // No OCI version label
		},
		tags: map[string][]string{
			"app/demo": {"1.0.0", "1.1.0", "1.2.0", "stable"},
		},
	}

	w := &db.Workload{
		ID:            "docker-123456789012",
		Name:          "demo",
		CurrentImage:  "app/demo:1.0.0",
		CurrentDigest: "sha256:old100",
		TrackTag:      strPtr("stable"),
	}

	updated, err := checkWorkloadWithRegistry(context.Background(), fake, nil, w, "docker.io", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated || w.NewImage == nil || *w.NewImage != "app/demo:1.1.0" {
		t.Fatalf("expected fallback to 'app/demo:1.1.0', got updated=%v NewImage=%v", updated, w.NewImage)
	}
}

func TestCheckWorkloadWithRegistry_TrackedChannelFallbackToChannelTag(t *testing.T) {
	fake := &fakeRegistryAPI{
		digests: map[string]string{
			"app/custom:1.0.0":  "sha256:old100",
			"app/custom:stable": "sha256:newdigest",
		},
		labels:      map[string]map[string]string{},
		listTagsErr: errors.New("tags/list unsupported"),
	}

	w := &db.Workload{
		ID:            "docker-custom",
		Name:          "custom",
		CurrentImage:  "app/custom:1.0.0",
		CurrentDigest: "sha256:old100",
		TrackTag:      strPtr("stable"),
	}

	updated, err := checkWorkloadWithRegistry(context.Background(), fake, nil, w, "docker.io", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated || w.NewImage == nil || *w.NewImage != "app/custom:stable" {
		t.Fatalf("expected last-resort fallback 'app/custom:stable', got updated=%v NewImage=%v", updated, w.NewImage)
	}
}

func TestCheckWorkloadWithRegistry_TrackedChannelMissingReturnsError(t *testing.T) {
	fake := &fakeRegistryAPI{
		digestErrs: map[string]error{
			"n8nio/n8n:nonexistent": errors.New("registry returned HTTP 404"),
		},
		tags: map[string][]string{
			"n8nio/n8n": {"2.48.7", "2.49.3"},
		},
	}

	w := &db.Workload{
		ID:            "k8s:default:deploy:n8n:n8n",
		Name:          "n8n",
		CurrentImage:  "n8nio/n8n:2.48.7",
		CurrentDigest: "sha256:old",
		TrackTag:      strPtr("nonexistent"),
	}

	updated, err := checkWorkloadWithRegistry(context.Background(), fake, nil, w, "docker.io", "", "")
	if err == nil {
		t.Fatalf("expected error for missing track-tag, got updated=%v NewImage=%v", updated, w.NewImage)
	}
	if fake.listTagsCalls != 0 {
		t.Errorf("expected no silent fallback to SemVer ListTags on track-tag error, got %d calls", fake.listTagsCalls)
	}
}

func TestCheckWorkloadWithRegistry_FloatingBetaTagSkipsListTags(t *testing.T) {
	fake := &fakeRegistryAPI{
		digests: map[string]string{
			"ghcr.io/giacomoranieri/container-updater-backend:beta": "sha256:newbeta",
		},
		tags: map[string][]string{
			"ghcr.io/giacomoranieri/container-updater-backend": {"beta", "latest", "v1.0.0"},
		},
	}

	w := &db.Workload{
		ID:            "k8s:default:deploy:container-updater-backend:backend",
		Name:          "container-updater-backend",
		CurrentImage:  "ghcr.io/giacomoranieri/container-updater-backend:beta",
		CurrentDigest: "sha256:oldbeta",
	}

	updated, err := checkWorkloadWithRegistry(context.Background(), fake, nil, w, "ghcr.io", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated || w.NewDigest == nil || *w.NewDigest != "newbeta" {
		t.Fatalf("expected digest update on floating beta tag, got updated=%v NewDigest=%v", updated, w.NewDigest)
	}
	if fake.listTagsCalls != 0 {
		t.Errorf("expected 0 ListTags calls for non-version tag 'beta', got %d", fake.listTagsCalls)
	}
}

func TestCheckWorkloadWithRegistry_TrackedChannelVPrefixAdaptation(t *testing.T) {
	fake := &fakeRegistryAPI{
		digests: map[string]string{
			"ghcr.io/org/app:v1.0.0": "sha256:old",
			"ghcr.io/org/app:stable": "sha256:new110",
			"ghcr.io/org/app:v1.1.0": "sha256:new110",
		},
		labels: map[string]map[string]string{
			"ghcr.io/org/app:stable": {
				"org.opencontainers.image.version": "1.1.0",
			},
		},
	}

	w := &db.Workload{
		ID:            "k8s:default:deploy:app:app",
		Name:          "app",
		CurrentImage:  "ghcr.io/org/app:v1.0.0",
		CurrentDigest: "sha256:old",
		TrackTag:      strPtr("stable"),
	}

	updated, err := checkWorkloadWithRegistry(context.Background(), fake, nil, w, "ghcr.io", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated || w.NewImage == nil || *w.NewImage != "ghcr.io/org/app:v1.1.0" {
		t.Fatalf("expected v-prefix preserved 'ghcr.io/org/app:v1.1.0', got updated=%v NewImage=%v", updated, w.NewImage)
	}
}
