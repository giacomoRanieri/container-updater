package monitor

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"container-updater/backend/internal/db"
	"container-updater/backend/internal/docker"
	"container-updater/backend/internal/logger"
	"container-updater/backend/internal/registry"

	dockertypes "github.com/docker/docker/api/types/registry"
)

type registryAPI interface {
	FetchDigest(ctx context.Context, imageRef, username, password string) (string, error)
	ListTags(ctx context.Context, imageRef, username, password string) ([]string, error)
	FetchImageLabels(ctx context.Context, imageRef, username, password string) (map[string]string, error)
}

const maxFallbackDigestChecks = 15

// CheckWorkloadUpdate compares the running digest/tag of a workload against its remote registry.
func CheckWorkloadUpdate(ctx context.Context, dockerClient *docker.DockerClient, w *db.Workload) (bool, error) {
	registryHost := resolveRegistryHost(w.CurrentImage)

	var username, password string
	if db.DB != nil {
		query := `SELECT username, password FROM registry_credentials WHERE server_address = ?`
		err := db.DB.QueryRow(query, registryHost).Scan(&username, &password)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			logger.Log.Error("database query failed for registry credentials", "registry", registryHost, "error", err)
		}
	}

	return checkWorkloadWithRegistry(ctx, registry.NewClient(), dockerClient, w, registryHost, username, password)
}

func checkWorkloadWithRegistry(
	ctx context.Context,
	regClient registryAPI,
	dockerClient *docker.DockerClient,
	w *db.Workload,
	registryHost, username, password string,
) (bool, error) {
	repo, currentTag := splitImageRepoTag(w.CurrentImage)

	// 1. Explicit opt-in channel pointer tracking (e.g. track-tag: "stable")
	if w.TrackTag != nil {
		trackTag := strings.TrimSpace(*w.TrackTag)
		if trackTag != "" && currentTag != trackTag {
			return checkTrackedChannel(ctx, regClient, w, repo, currentTag, trackTag, username, password)
		}
	}

	// 2. Default schema- & flavor-aware version tag tracking (skip ListTags for non-version tags)
	if _, isVersion := registry.ParseVersion(currentTag); currentTag != "latest" && isVersion {
		tags, err := regClient.ListTags(ctx, w.CurrentImage, username, password)
		if err == nil {
			if latestSemverTag, found := registry.FindLatestMatchingTag(currentTag, tags); found {
				newerImageRef := repo + ":" + latestSemverTag
				newerDigest, err := regClient.FetchDigest(ctx, newerImageRef, username, password)
				if err == nil {
					cleanNewerDigest := strings.TrimPrefix(newerDigest, "sha256:")
					logger.Log.Info("New compatible version release tag found", "workload", w.Name, "currentTag", currentTag, "newerTag", latestSemverTag)
					w.NewImage = &newerImageRef
					w.NewDigest = &cleanNewerDigest
					w.UpdateStatus = "update_available"
					now := time.Now()
					w.LastCheckedAt = &now
					return true, nil
				}
			}
		} else {
			logger.Log.Debug("Tag listing not supported or failed, falling back to digest check", "image", w.CurrentImage, "error", err)
		}
	}

	// 3. Fallback to same-tag digest comparison
	cleanRemote, err := regClient.FetchDigest(ctx, w.CurrentImage, username, password)
	if err != nil {
		logger.Log.Warn("HTTP OCI registry inspect failed, trying Docker client fallback", "image", w.CurrentImage, "error", err)
		if dockerClient != nil {
			return checkViaDockerClient(ctx, dockerClient, w, registryHost, username, password)
		}
		return false, err
	}

	cleanRemote = strings.TrimPrefix(cleanRemote, "sha256:")
	cleanCurrent := strings.TrimPrefix(w.CurrentDigest, "sha256:")
	logger.Log.Info("Compare image digests", "workload", w.Name, "current", cleanCurrent, "remote", cleanRemote)

	now := time.Now()
	w.LastCheckedAt = &now

	if cleanRemote != cleanCurrent && cleanCurrent != "" {
		w.NewImage = &w.CurrentImage
		w.NewDigest = &cleanRemote
		w.UpdateStatus = "update_available"
		return true, nil
	}

	markUpToDate(w, now)
	return false, nil
}

func checkTrackedChannel(
	ctx context.Context,
	regClient registryAPI,
	w *db.Workload,
	repo, currentTag, trackTag, username, password string,
) (bool, error) {
	trackImageRef := repo + ":" + trackTag
	trackDigest, err := regClient.FetchDigest(ctx, trackImageRef, username, password)
	if err != nil {
		return false, fmt.Errorf("failed to fetch tracked tag %q for %s: %w", trackTag, w.CurrentImage, err)
	}
	trackDigest = strings.TrimPrefix(trackDigest, "sha256:")

	now := time.Now()

	// Compare tracked channel digest against current tag's remote digest first, then running digest.
	if currentTagDigest, err := regClient.FetchDigest(ctx, w.CurrentImage, username, password); err == nil {
		if strings.TrimPrefix(currentTagDigest, "sha256:") == trackDigest {
			markUpToDate(w, now)
			return false, nil
		}
	}
	if cleanCurrent := strings.TrimPrefix(w.CurrentDigest, "sha256:"); cleanCurrent != "" && cleanCurrent == trackDigest {
		markUpToDate(w, now)
		return false, nil
	}

	resolvedTag := resolvePinnedTagFromChannel(ctx, regClient, w.CurrentImage, repo, currentTag, trackImageRef, trackDigest, username, password)

	// Guard against proposing a downgrade if running version is already ahead of tracked channel.
	if resolvedTag != "" {
		if curVer, ok1 := registry.ParseVersion(currentTag); ok1 {
			if resVer, ok2 := registry.ParseVersion(resolvedTag); ok2 && curVer.IsCompatibleWith(resVer) && curVer.GreaterThan(resVer) {
				logger.Log.Info("Running version is ahead of tracked channel tag; skipping downgrade",
					"workload", w.Name, "currentTag", currentTag, "trackTag", trackTag, "resolvedTag", resolvedTag)
				markUpToDate(w, now)
				return false, nil
			}
		}
	}

	if resolvedTag == "" {
		logger.Log.Warn("Tracked channel digest changed but no matching pinned version tag found; falling back to tracked channel tag",
			"workload", w.Name, "currentImage", w.CurrentImage, "trackTag", trackTag)
		resolvedTag = trackTag
	}

	newerImageRef := repo + ":" + resolvedTag
	logger.Log.Info("Update available on tracked channel",
		"workload", w.Name, "currentTag", currentTag, "trackTag", trackTag, "resolvedTag", resolvedTag)

	w.NewImage = &newerImageRef
	w.NewDigest = &trackDigest
	w.UpdateStatus = "update_available"
	w.LastCheckedAt = &now
	return true, nil
}

func resolvePinnedTagFromChannel(
	ctx context.Context,
	regClient registryAPI,
	currentImage, repo, currentTag, trackImageRef, trackDigest, username, password string,
) string {
	// Strategy 1: Inspect OCI org.opencontainers.image.version label on tracked channel
	if labels, err := regClient.FetchImageLabels(ctx, trackImageRef, username, password); err == nil {
		if ver := strings.TrimSpace(labels["org.opencontainers.image.version"]); ver != "" {
			for _, candTag := range buildVersionTagCandidates(ver, currentTag) {
				if candTag == currentTag {
					continue
				}
				if candDigest, err := regClient.FetchDigest(ctx, repo+":"+candTag, username, password); err == nil {
					if strings.TrimPrefix(candDigest, "sha256:") == trackDigest {
						return candTag
					}
				}
			}
		}
	} else {
		logger.Log.Debug("OCI label inspection failed for tracked tag", "image", trackImageRef, "error", err)
	}

	// Strategy 2: Fallback digest match against compatible higher version tags
	tags, err := regClient.ListTags(ctx, currentImage, username, password)
	if err != nil {
		return ""
	}
	candidates, ok := registry.FindMatchingTagsDescending(currentTag, tags)
	if !ok {
		return ""
	}
	limit := maxFallbackDigestChecks
	if len(candidates) < limit {
		limit = len(candidates)
	}
	for _, cand := range candidates[:limit] {
		if candDigest, err := regClient.FetchDigest(ctx, repo+":"+cand.Original, username, password); err == nil {
			if strings.TrimPrefix(candDigest, "sha256:") == trackDigest {
				return cand.Original
			}
		}
	}
	return ""
}

func buildVersionTagCandidates(ver, currentTag string) []string {
	var adjusted string
	if strings.HasPrefix(currentTag, "v") && !strings.HasPrefix(ver, "v") {
		adjusted = "v" + ver
	} else if !strings.HasPrefix(currentTag, "v") && strings.HasPrefix(ver, "v") {
		adjusted = strings.TrimPrefix(ver, "v")
	}

	if adjusted != "" && adjusted != ver {
		return []string{adjusted, ver}
	}
	return []string{ver}
}

func markUpToDate(w *db.Workload, checkedAt time.Time) {
	w.NewImage = nil
	w.NewDigest = nil
	w.UpdateStatus = "up_to_date"
	w.LastCheckedAt = &checkedAt
}

func splitImageRepoTag(imageRef string) (repo, tag string) {
	ref := imageRef
	if atIdx := strings.Index(ref, "@"); atIdx != -1 {
		ref = ref[:atIdx]
	}
	tag = "latest"
	if idx := strings.LastIndex(ref, ":"); idx != -1 && !strings.Contains(ref[idx:], "/") {
		tag = ref[idx+1:]
		repo = ref[:idx]
		return repo, tag
	}
	return ref, tag
}

func checkViaDockerClient(ctx context.Context, dockerClient *docker.DockerClient, w *db.Workload, registryHost, username, password string) (bool, error) {
	var encodedAuth string
	if username != "" && password != "" {
		authConfig := dockertypes.AuthConfig{
			Username:      username,
			Password:      password,
			ServerAddress: registryHost,
		}
		authBytes, err := json.Marshal(authConfig)
		if err == nil {
			encodedAuth = base64.URLEncoding.EncodeToString(authBytes)
		}
	}

	fullImageName := w.CurrentImage
	if !strings.Contains(fullImageName, "/") {
		fullImageName = "docker.io/library/" + fullImageName
	} else if !strings.Contains(strings.Split(fullImageName, "/")[0], ".") && !strings.Contains(strings.Split(fullImageName, "/")[0], ":") {
		fullImageName = "docker.io/" + fullImageName
	}

	inspect, err := dockerClient.DistributionInspect(ctx, fullImageName, encodedAuth)
	if err != nil {
		return false, err
	}

	remoteDigest := string(inspect.Descriptor.Digest)
	if remoteDigest == "" {
		return false, errors.New("empty remote digest returned from registry")
	}

	cleanRemote := strings.TrimPrefix(remoteDigest, "sha256:")
	cleanCurrent := strings.TrimPrefix(w.CurrentDigest, "sha256:")

	now := time.Now()
	w.LastCheckedAt = &now

	if cleanRemote != cleanCurrent && cleanCurrent != "" {
		w.NewImage = &w.CurrentImage
		w.NewDigest = &cleanRemote
		w.UpdateStatus = "update_available"
		return true, nil
	}

	markUpToDate(w, now)
	return false, nil
}

// resolveRegistryHost parses the image reference and extracts the registry domain
func resolveRegistryHost(imageRef string) string {
	repo, _ := splitImageRepoTag(imageRef)
	parts := strings.Split(repo, "/")
	if len(parts) <= 1 {
		return "docker.io"
	}

	firstPart := parts[0]
	// If the first part contains a dot (.) or colon (:), it is a registry host name
	if strings.Contains(firstPart, ".") || strings.Contains(firstPart, ":") || firstPart == "localhost" {
		return firstPart
	}

	return "docker.io"
}
