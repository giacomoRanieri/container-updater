package db

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

func strPtr(s string) *string {
	return &s
}

func TestSaveAndGetWorkload_WithAndWithoutTrackTag(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.sqlite")
	if err := Init(dbPath); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer DB.Close()

	now := time.Now().UTC().Truncate(time.Second)

	w1 := &Workload{
		ID:               "k8s:default:deploy:n8n:n8n",
		Name:             "n8n",
		NamespaceProject: "default",
		OrchestratorType: "kubernetes",
		CurrentImage:     "n8nio/n8n:2.48.7",
		CurrentDigest:    "abc123",
		TrackTag:         strPtr("stable"),
		UpdateStatus:     "up_to_date",
		LastCheckedAt:    &now,
	}
	if err := SaveWorkload(w1); err != nil {
		t.Fatalf("SaveWorkload w1 failed: %v", err)
	}

	w2 := &Workload{
		ID:               "k8s:default:deploy:authelia:authelia",
		Name:             "authelia",
		NamespaceProject: "default",
		OrchestratorType: "kubernetes",
		CurrentImage:     "authelia/authelia:4.39.28",
		CurrentDigest:    "def456",
		TrackTag:         nil,
		UpdateStatus:     "up_to_date",
		LastCheckedAt:    &now,
	}
	if err := SaveWorkload(w2); err != nil {
		t.Fatalf("SaveWorkload w2 failed: %v", err)
	}

	got1, err := GetWorkload(w1.ID)
	if err != nil || got1 == nil {
		t.Fatalf("GetWorkload w1 = (%v, %v)", got1, err)
	}
	if got1.TrackTag == nil || *got1.TrackTag != "stable" {
		t.Fatalf("w1 TrackTag = %v, want 'stable'", got1.TrackTag)
	}

	got2, err := GetWorkload(w2.ID)
	if err != nil || got2 == nil {
		t.Fatalf("GetWorkload w2 = (%v, %v)", got2, err)
	}
	if got2.TrackTag != nil {
		t.Fatalf("w2 TrackTag = %v, want nil", *got2.TrackTag)
	}

	// Clearing TrackTag on update should persist nil
	w1.TrackTag = nil
	if err := SaveWorkload(w1); err != nil {
		t.Fatalf("SaveWorkload clear TrackTag failed: %v", err)
	}
	got1Cleared, err := GetWorkload(w1.ID)
	if err != nil || got1Cleared == nil {
		t.Fatalf("GetWorkload w1 cleared = (%v, %v)", got1Cleared, err)
	}
	if got1Cleared.TrackTag != nil {
		t.Fatalf("expected cleared TrackTag to be nil, got %q", *got1Cleared.TrackTag)
	}

	list, err := ListWorkloads()
	if err != nil {
		t.Fatalf("ListWorkloads failed: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("ListWorkloads len = %d, want 2", len(list))
	}
}

func TestInitMigratesLegacyWorkloadsTable(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy.sqlite")

	// Create legacy DB without track_tag column
	rawDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open legacy sqlite: %v", err)
	}
	_, err = rawDB.Exec(`
		CREATE TABLE workloads (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			namespace_project TEXT NOT NULL,
			orchestrator_type TEXT NOT NULL,
			current_image TEXT NOT NULL,
			current_digest TEXT NOT NULL,
			new_image TEXT,
			new_digest TEXT,
			update_status TEXT NOT NULL DEFAULT 'up_to_date',
			last_checked_at DATETIME,
			last_updated_at DATETIME
		);
		INSERT INTO workloads (id, name, namespace_project, orchestrator_type, current_image, current_digest, update_status)
		VALUES ('k8s:default:deploy:legacy:legacy', 'legacy', 'default', 'kubernetes', 'nginx:1.25.0', 'sha256:111', 'up_to_date');
	`)
	if err != nil {
		t.Fatalf("seed legacy db: %v", err)
	}
	rawDB.Close()

	// Run Init which should migrate workloads cleanly without losing data
	if err := Init(dbPath); err != nil {
		t.Fatalf("Init migration failed: %v", err)
	}
	defer DB.Close()

	got, err := GetWorkload("k8s:default:deploy:legacy:legacy")
	if err != nil || got == nil {
		t.Fatalf("GetWorkload after migration = (%v, %v)", got, err)
	}
	if got.CurrentImage != "nginx:1.25.0" || got.TrackTag != nil {
		t.Fatalf("unexpected migrated workload: %+v", got)
	}

	// Verify writing TrackTag works post-migration
	got.TrackTag = strPtr("stable")
	if err := SaveWorkload(got); err != nil {
		t.Fatalf("SaveWorkload post-migration failed: %v", err)
	}
	reloaded, err := GetWorkload(got.ID)
	if err != nil || reloaded == nil || reloaded.TrackTag == nil || *reloaded.TrackTag != "stable" {
		t.Fatalf("expected persisted TrackTag 'stable' post-migration, got %+v (err=%v)", reloaded, err)
	}
}
