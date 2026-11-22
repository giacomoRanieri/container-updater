package registry

import (
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input      string
		valid      bool
		scheme     VersionScheme
		hasV       bool
		parts      int
		major      int
		minor      int
		patch      int
		isPre      bool
		flavor     string
		hasRev     bool
		revision   int
	}{
		{"1.25.0", true, SchemeSemVer, false, 3, 1, 25, 0, false, "", false, 0},
		{"2026.6.4", true, SchemeCalVer, false, 3, 2026, 6, 4, false, "", false, 0},
		{"v2.10.1", true, SchemeSemVer, true, 3, 2, 10, 1, false, "", false, 0},
		{"2026.6.5-rc1", true, SchemeCalVer, false, 3, 2026, 6, 5, true, "", false, 0},
		{"2.1.2-alpine", true, SchemeSemVer, false, 3, 2, 1, 2, false, "alpine", false, 0},
		{"2.1.5-rc1-alpine", true, SchemeSemVer, false, 3, 2, 1, 5, true, "alpine", false, 0},
		{"2.0.0-beta.2-alpine", true, SchemeSemVer, false, 3, 2, 0, 0, true, "alpine", false, 0},
		{"1.0.20260223-r0-ls117", true, SchemeSemVer, false, 3, 1, 0, 20260223, false, "r0-ls", true, 117},
		{"20230811", true, SchemeCompactDate, false, 1, 20230811, 0, 0, false, "", false, 0},
		{"16", true, SchemeSingleMajor, false, 1, 16, 0, 0, false, "", false, 0},
		{"2.1", true, SchemeSemVer, false, 2, 2, 1, 0, false, "", false, 0},
		// Rejected inputs
		{"latest", false, SchemeUnknown, false, 0, 0, 0, 0, false, "", false, 0},
		{"44760626283", false, SchemeUnknown, false, 0, 0, 0, 0, false, "", false, 0},
		{"20239999", false, SchemeUnknown, false, 0, 0, 0, 0, false, "", false, 0},
		{"2026.11.0b1", false, SchemeUnknown, false, 0, 0, 0, 0, false, "", false, 0},
		{"2026.12.0.dev20261120", false, SchemeUnknown, false, 0, 0, 0, 0, false, "", false, 0},
	}

	for _, tt := range tests {
		v, ok := ParseVersion(tt.input)
		if ok != tt.valid {
			t.Errorf("ParseVersion(%q) valid = %v, want %v", tt.input, ok, tt.valid)
			continue
		}
		if ok {
			if v.Scheme != tt.scheme || v.HasVPrefix != tt.hasV || v.Parts != tt.parts ||
				v.Major != tt.major || v.Minor != tt.minor || v.Patch != tt.patch ||
				v.IsPre != tt.isPre || v.Flavor != tt.flavor ||
				v.HasRevision != tt.hasRev || v.Revision != tt.revision {
				t.Errorf("ParseVersion(%q) = %+v, want scheme=%v hasV=%v parts=%d %d.%d.%d isPre=%v flavor=%q hasRev=%v rev=%d",
					tt.input, v, tt.scheme, tt.hasV, tt.parts, tt.major, tt.minor, tt.patch, tt.isPre, tt.flavor, tt.hasRev, tt.revision)
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

func TestFindLatestMatchingTag_RealWorldRegressions(t *testing.T) {
	tests := []struct {
		name       string
		currentTag string
		tags       []string
		wantTag    string
		wantFound  bool
	}{
		{
			name:       "matterbridge standalone ignores HA addon CalVer and dev tags",
			currentTag: "3.11.4",
			tags:       []string{"latest", "stable", "3.11.4", "dev-3.11.4", "alpine-3.11.4", "2026.11.3", "2026.11.3-legacy"},
			wantTag:    "",
			wantFound:  false,
		},
		{
			name:       "matterbridge standalone picks newer SemVer and ignores CalVer",
			currentTag: "3.11.2",
			tags:       []string{"3.11.2", "3.11.4", "2026.11.3", "2026.11.3-legacy"},
			wantTag:    "3.11.4",
			wantFound:  true,
		},
		{
			name:       "n8n ignores 20230811 compact date and suffixed variants",
			currentTag: "2.48.7",
			tags:       []string{"latest", "stable", "2.48.7", "20230811", "2.48.7-pc", "2.48.7-amd64", "2.48.7-86dff76", "2.48.8-debian"},
			wantTag:    "",
			wantFound:  false,
		},
		{
			name:       "n8n picks clean newer SemVer over 20230811 and suffixed tags",
			currentTag: "2.48.7",
			tags:       []string{"2.48.7", "20230811", "2.48.8-debian", "2.48.8", "2.49.0-rc1"},
			wantTag:    "2.48.8",
			wantFound:  true,
		},
		{
			name:       "eclipse-mosquitto alpine stays on alpine flavor and 3-part precision",
			currentTag: "2.1.2-alpine",
			tags:       []string{"latest", "2.1.2-alpine", "2.1.4-alpine", "2.1-alpine", "2.1.4-openssl", "2.1.4", "2.1.5-rc1-alpine"},
			wantTag:    "2.1.4-alpine",
			wantFound:  true,
		},
		{
			name:       "caronc/apprise 2-part tag stays on 2-part precision",
			currentTag: "2.1",
			tags:       []string{"latest", "2", "2.1", "2.1.0", "2.2", "2.2.0", "v2.3"},
			wantTag:    "2.2",
			wantFound:  true,
		},
		{
			name:       "victoriametrics preserves v-prefix and skips enterprise/scratch/rc",
			currentTag: "v1.109.1",
			tags:       []string{"latest", "v1.109.1", "v1.156.0", "v1.156.0-scratch", "v1.156.0-enterprise", "v1.157.0-rc0", "1.158.0"},
			wantTag:    "v1.156.0",
			wantFound:  true,
		},
		{
			name:       "home-assistant CalVer upgrades within CalVer and ignores SemVer/beta/dev",
			currentTag: "2026.11.0",
			tags:       []string{"stable", "latest", "2026.11.0", "2026.11.2", "2026.11.3b1", "2026.12.0.dev20261120", "3.11.4"},
			wantTag:    "2026.11.2",
			wantFound:  true,
		},
		{
			name:       "linuxserver wireguard increments -ls revision",
			currentTag: "1.0.20260223-r0-ls117",
			tags:       []string{"latest", "1.0.20260223-r0-ls117", "1.0.20260223-r0-ls130", "amd64-1.0.20260223-r0-ls131", "v1.0.20210315-ls21"},
			wantTag:    "1.0.20260223-r0-ls130",
			wantFound:  true,
		},
		{
			name:       "single major stays in single major scheme",
			currentTag: "16",
			tags:       []string{"16", "17", "17.2", "17-alpine", "20230811"},
			wantTag:    "17",
			wantFound:  true,
		},
		{
			name:       "compact date stays in compact date scheme",
			currentTag: "20230811",
			tags:       []string{"20230811", "20240115", "2.48.7", "99999999"},
			wantTag:    "20240115",
			wantFound:  true,
		},
		{
			name:       "prerelease upgrades to matching stable release",
			currentTag: "2.0.0-rc1",
			tags:       []string{"2.0.0-rc1", "2.0.0-rc2", "2.0.0"},
			wantTag:    "2.0.0",
			wantFound:  true,
		},
		{
			name:       "prerelease upgrades to higher prerelease when stable absent",
			currentTag: "2.0.0-rc1",
			tags:       []string{"2.0.0-rc1", "2.0.0-rc2"},
			wantTag:    "2.0.0-rc2",
			wantFound:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTag, gotFound := FindLatestMatchingTag(tt.currentTag, tt.tags)
			if gotFound != tt.wantFound || gotTag != tt.wantTag {
				t.Errorf("FindLatestMatchingTag(%q) = (%q, %v), want (%q, %v)",
					tt.currentTag, gotTag, gotFound, tt.wantTag, tt.wantFound)
			}
		})
	}
}
