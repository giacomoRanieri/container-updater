package db

import "testing"

func TestMaskRegistryPassword(t *testing.T) {
	if got := MaskRegistryPassword("super-secret"); got != "[set]" {
		t.Fatalf("expected masked password placeholder, got %q", got)
	}

	if got := MaskRegistryPassword(""); got != "" {
		t.Fatalf("expected empty password to remain empty, got %q", got)
	}
}
