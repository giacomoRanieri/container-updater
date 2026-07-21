package registry

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Version represents a parsed semantic version or calver (e.g. 1.25.0, 2026.6.4, v2.3.1)
type Version struct {
	Original string
	Major    int
	Minor    int
	Patch    int
	Build    int
	IsPre    bool
}

var versionRegex = regexp.MustCompile(`^v?(\d+)(?:\.(\d+))?(?:\.(\d+))?(?:\.(\d+))?(?:[-+].*)?$`)

func ParseVersion(v string) (Version, bool) {
	vClean := strings.TrimSpace(v)
	matches := versionRegex.FindStringSubmatch(vClean)
	if matches == nil {
		return Version{}, false
	}

	major, _ := strconv.Atoi(matches[1])
	minor := 0
	if matches[2] != "" {
		minor, _ = strconv.Atoi(matches[2])
	}
	patch := 0
	if matches[3] != "" {
		patch, _ = strconv.Atoi(matches[3])
	}
	build := 0
	if matches[4] != "" {
		build, _ = strconv.Atoi(matches[4])
	}

	isPre := strings.Contains(vClean, "-") || strings.Contains(vClean, "alpha") ||
		strings.Contains(vClean, "beta") || strings.Contains(vClean, "rc") || strings.Contains(vClean, "dev")

	return Version{
		Original: vClean,
		Major:    major,
		Minor:    minor,
		Patch:    patch,
		Build:    build,
		IsPre:    isPre,
	}, true
}

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
	return v.Build > other.Build
}

// FindLatestMatchingTag searches available tags for the highest semver tag greater than currentTag.
func FindLatestMatchingTag(currentTag string, availableTags []string) (string, bool) {
	currentVer, ok := ParseVersion(currentTag)
	if !ok {
		return "", false
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
		// Skip prereleases unless the current version itself is a prerelease
		if v.IsPre && !currentVer.IsPre {
			continue
		}
		if v.GreaterThan(currentVer) {
			parsedVersions = append(parsedVersions, v)
		}
	}

	if len(parsedVersions) == 0 {
		return "", false
	}

	sort.Slice(parsedVersions, func(i, j int) bool {
		return parsedVersions[i].GreaterThan(parsedVersions[j])
	})

	return parsedVersions[0].Original, true
}
