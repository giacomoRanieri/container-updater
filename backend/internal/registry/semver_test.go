package registry

import (
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input    string
		valid    bool
		major    int
		minor    int
		patch    int
		isPre    bool
	}{
		{"1.25.0", true, 1, 25, 0, false},
		{"2026.6.4", true, 2026, 6, 4, false},
		{"v2.10.1", true, 2, 10, 1, false},
		{"2026.6.5-rc1", true, 2026, 6, 5, true},
		{"latest", false, 0, 0, 0, false},
	}

	for _, tt := range tests {
		v, ok := ParseVersion(tt.input)
		if ok != tt.valid {
			t.Errorf("ParseVersion(%q) valid = %v, want %v", tt.input, ok, tt.valid)
			continue
		}
		if ok {
			if v.Major != tt.major || v.Minor != tt.minor || v.Patch != tt.patch || v.IsPre != tt.isPre {
				t.Errorf("ParseVersion(%q) = %+v, want Major:%d Minor:%d Patch:%d IsPre:%v",
					tt.input, v, tt.major, tt.minor, tt.patch, tt.isPre)
			}
		}
	}
}

func TestFindLatestMatchingTag(t *testing.T) {
	tags := []string{"2026.6.0", "2026.6.1", "2026.6.4", "2026.6.5", "2026.6.6-beta1", "2026.7.0", "latest"}

	latest, found := FindLatestMatchingTag("2026.6.4", tags)
	if !found || latest != "2026.7.0" {
		t.Errorf("FindLatestMatchingTag('2026.6.4') = (%q, %v), want ('2026.7.0', true)", latest, found)
	}

	latest, found = FindLatestMatchingTag("2026.7.0", tags)
	if found {
		t.Errorf("FindLatestMatchingTag('2026.7.0') expected no update, got %q", latest)
	}
}
