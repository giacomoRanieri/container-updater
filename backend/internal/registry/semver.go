package registry

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// VersionScheme identifies the versioning family of a parsed image tag so incompatible
// schemes (e.g. SemVer 3.11.4 vs CalVer 2026.11.3 vs compact date 20230811) are never mixed.
type VersionScheme int

const (
	SchemeUnknown     VersionScheme = iota
	SchemeSemVer                    // Parts >= 2, Major < 1900 (e.g. 3.11.4, 2.48.7, 2.1, v1.109.1, 1.0.20260223)
	SchemeCalVer                    // Parts >= 2, Major in 2000..2099 (e.g. 2026.11.0, 2026.6.4, 2026.11.3)
	SchemeCompactDate               // Parts == 1, valid 8-digit YYYYMMDD date (1970..2099)
	SchemeSingleMajor               // Parts == 1, 0 <= Major < 1000 (e.g. 16, 17, v2)
)

// Version represents a parsed container image tag with scheme, precision, prefix, and flavor metadata.
type Version struct {
	Original    string
	Scheme      VersionScheme
	HasVPrefix  bool
	Parts       int // number of dotted numeric segments (1..4)
	Major       int
	Minor       int
	Patch       int
	Build       int
	IsPre       bool
	Flavor      string // normalized lowercase non-prerelease variant suffix (e.g. "", "alpine", "legacy", "r0-ls")
	RawSuffix   string // full lowercase suffix after version core (for prerelease tie-break)
	HasRevision bool   // true when suffix ends with LinuxServer-style -ls<N> (or ls<N>)
	Revision    int    // numeric revision extracted from ls<N>
}

var (
	versionRegex  = regexp.MustCompile(`^(v)?(\d+)(?:\.(\d+))?(?:\.(\d+))?(?:\.(\d+))?(?:[-+](.+))?$`)
	lsRevRegex    = regexp.MustCompile(`^(?:(.*)-)?ls(\d+)$`)
	preTokenRegex = regexp.MustCompile(`^(alpha|beta|rc|dev|nightly|next|canary|preview|pre|snapshot|edge|test|exp|experimental|unstable)(\d.*)?$`)
)

// ParseVersion parses a container tag into a structured Version.
func ParseVersion(v string) (Version, bool) {
	vClean := strings.TrimSpace(v)
	matches := versionRegex.FindStringSubmatch(vClean)
	if matches == nil {
		return Version{}, false
	}

	hasVPrefix := matches[1] == "v"

	major, err := strconv.Atoi(matches[2])
	if err != nil {
		return Version{}, false
	}
	parts := 1

	var minor, patch, build int
	if matches[3] != "" {
		minor, err = strconv.Atoi(matches[3])
		if err != nil {
			return Version{}, false
		}
		parts = 2
	}
	if matches[4] != "" {
		patch, err = strconv.Atoi(matches[4])
		if err != nil {
			return Version{}, false
		}
		parts = 3
	}
	if matches[5] != "" {
		build, err = strconv.Atoi(matches[5])
		if err != nil {
			return Version{}, false
		}
		parts = 4
	}

	scheme := classifyScheme(parts, major, matches[2])
	if scheme == SchemeUnknown {
		return Version{}, false
	}

	isPre, flavor, rawSuffix, hasRevision, revision := parseSuffix(matches[6])

	return Version{
		Original:    vClean,
		Scheme:      scheme,
		HasVPrefix:  hasVPrefix,
		Parts:       parts,
		Major:       major,
		Minor:       minor,
		Patch:       patch,
		Build:       build,
		IsPre:       isPre,
		Flavor:      flavor,
		RawSuffix:   rawSuffix,
		HasRevision: hasRevision,
		Revision:    revision,
	}, true
}

func classifyScheme(parts, major int, majorRaw string) VersionScheme {
	if parts == 1 {
		if len(majorRaw) == 8 {
			if t, err := time.Parse("20060102", majorRaw); err == nil && t.Year() >= 1970 && t.Year() <= 2099 {
				return SchemeCompactDate
			}
			return SchemeUnknown
		}
		if major >= 0 && major < 1000 {
			return SchemeSingleMajor
		}
		return SchemeUnknown
	}

	if major >= 2000 && major <= 2099 {
		return SchemeCalVer
	}
	if major >= 0 && major < 1900 {
		return SchemeSemVer
	}
	return SchemeUnknown
}

func parseSuffix(suffix string) (isPre bool, flavor, rawSuffix string, hasRevision bool, revision int) {
	if suffix == "" {
		return false, "", "", false, 0
	}

	rawSuffix = strings.ToLower(suffix)
	normalized := rawSuffix

	if m := lsRevRegex.FindStringSubmatch(rawSuffix); m != nil {
		if rev, err := strconv.Atoi(m[2]); err == nil {
			hasRevision = true
			revision = rev
			if m[1] != "" {
				normalized = m[1] + "-ls"
			} else {
				normalized = "ls"
			}
		}
	}

	tokens := strings.FieldsFunc(normalized, func(r rune) bool {
		return r == '-' || r == '.' || r == '+'
	})

	var flavorTokens []string
	prevWasPre := false
	for _, tok := range tokens {
		if tok == "" {
			continue
		}
		if preTokenRegex.MatchString(tok) {
			isPre = true
			prevWasPre = true
			continue
		}
		if prevWasPre && isAllDigits(tok) {
			prevWasPre = false
			continue
		}
		prevWasPre = false
		flavorTokens = append(flavorTokens, tok)
	}

	flavor = strings.Join(flavorTokens, "-")
	return isPre, flavor, rawSuffix, hasRevision, revision
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// IsCompatibleWith reports whether v belongs to the same tag stream as current
// (same scheme, segment count, v-prefix, flavor, revision style, and prerelease eligibility).
func (v Version) IsCompatibleWith(current Version) bool {
	if v.Scheme == SchemeUnknown || current.Scheme == SchemeUnknown {
		return false
	}
	if v.Scheme != current.Scheme ||
		v.Parts != current.Parts ||
		v.HasVPrefix != current.HasVPrefix ||
		v.Flavor != current.Flavor ||
		v.HasRevision != current.HasRevision {
		return false
	}
	if !current.IsPre && v.IsPre {
		return false
	}
	return true
}

// GreaterThan reports whether v sorts higher than other.
func (v Version) GreaterThan(other Version) bool {
	if v.Major != other.Major {
		return v.Major > other.Major
	}
	if v.Minor != other.Minor {
		return v.Minor > other.Minor
	}
	if v.Patch != other.Patch {
		return v.Patch > other.Patch
	}
	if v.Build != other.Build {
		return v.Build > other.Build
	}
	if v.IsPre != other.IsPre {
		return !v.IsPre
	}
	if v.HasRevision && other.HasRevision && v.Revision != other.Revision {
		return v.Revision > other.Revision
	}
	if v.IsPre && other.IsPre && v.RawSuffix != other.RawSuffix {
		return v.RawSuffix > other.RawSuffix
	}
	return false
}

// FindMatchingTagsDescending returns all available tags compatible with and greater than
// currentTag, sorted from highest to lowest version.
func FindMatchingTagsDescending(currentTag string, availableTags []string) ([]Version, bool) {
	currentVer, ok := ParseVersion(currentTag)
	if !ok {
		return nil, false
	}

	var parsedVersions []Version
	for _, tag := range availableTags {
		if tag == currentTag || tag == "latest" {
			continue
		}
		v, valid := ParseVersion(tag)
		if !valid {
			continue
		}
		if !v.IsCompatibleWith(currentVer) {
			continue
		}
		if v.GreaterThan(currentVer) {
			parsedVersions = append(parsedVersions, v)
		}
	}

	if len(parsedVersions) == 0 {
		return nil, false
	}

	sort.Slice(parsedVersions, func(i, j int) bool {
		return parsedVersions[i].GreaterThan(parsedVersions[j])
	})

	return parsedVersions, true
}

// FindLatestMatchingTag searches available tags for the highest compatible tag greater than currentTag.
func FindLatestMatchingTag(currentTag string, availableTags []string) (string, bool) {
	versions, ok := FindMatchingTagsDescending(currentTag, availableTags)
	if !ok || len(versions) == 0 {
		return "", false
	}
	return versions[0].Original, true
}
