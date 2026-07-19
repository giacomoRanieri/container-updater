package monitor

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"container-updater/backend/internal/db"
	"container-updater/backend/internal/docker"
	"container-updater/backend/internal/logger"
	"container-updater/backend/internal/registry"

	dockertypes "github.com/docker/docker/api/types/registry"
)

// CheckWorkloadUpdate compares the running digest of a workload with its remote registry digest
func CheckWorkloadUpdate(ctx context.Context, dockerClient *docker.DockerClient, w *db.Workload) (bool, error) {
	// 1. Resolve registry host from image name
	registryHost := resolveRegistryHost(w.CurrentImage)

	// 2. Fetch registry credentials from DB if they exist
	var username, password string
	query := `SELECT username, password FROM registry_credentials WHERE server_address = ?`
	err := db.DB.QueryRow(query, registryHost).Scan(&username, &password)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		logger.Log.Error("database query failed for registry credentials", "registry", registryHost, "error", err)
	}

	// 3. Query remote OCI registry directly via HTTP
	regClient := registry.NewClient()

	// 3a. Attempt SemVer tag tracking if current image tag is a version
	currentTag := "latest"
	if idx := strings.LastIndex(w.CurrentImage, ":"); idx != -1 && !strings.Contains(w.CurrentImage[idx:], "/") {
		currentTag = w.CurrentImage[idx+1:]
	}

	if currentTag != "latest" {
		tags, err := regClient.ListTags(ctx, w.CurrentImage, username, password)
		if err == nil {
			if latestSemverTag, found := registry.FindLatestMatchingTag(currentTag, tags); found {
				repo := w.CurrentImage
				if idx := strings.LastIndex(repo, ":"); idx != -1 && !strings.Contains(repo[idx:], "/") {
					repo = repo[:idx]
				}
				newerImageRef := repo + ":" + latestSemverTag
				newerDigest, err := regClient.FetchDigest(ctx, newerImageRef, username, password)
				if err == nil {
					cleanNewerDigest := strings.TrimPrefix(newerDigest, "sha256:")
					logger.Log.Info("New SemVer release tag found", "workload", w.Name, "currentTag", currentTag, "newerTag", latestSemverTag)
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

	// 3b. Fallback to same-tag digest comparison
	cleanRemote, err := regClient.FetchDigest(ctx, w.CurrentImage, username, password)
	if err != nil {
		logger.Log.Warn("HTTP OCI registry inspect failed, trying Docker client fallback", "image", w.CurrentImage, "error", err)
		if dockerClient != nil {
			return checkViaDockerClient(ctx, dockerClient, w, registryHost, username, password)
		}
		return false, err
	}

	cleanCurrent := strings.TrimPrefix(w.CurrentDigest, "sha256:")
	logger.Log.Info("Compare image digests", "workload", w.Name, "current", cleanCurrent, "remote", cleanRemote)

	w.NewDigest = &cleanRemote
	now := time.Now()
	w.LastCheckedAt = &now

	if cleanRemote != cleanCurrent && cleanCurrent != "" {
		w.NewImage = &w.CurrentImage
		w.UpdateStatus = "update_available"
		return true, nil
	}

	w.NewImage = nil
	w.NewDigest = nil
	w.UpdateStatus = "up_to_date"
	return false, nil
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

	w.NewDigest = &cleanRemote
	now := time.Now()
	w.LastCheckedAt = &now

	if cleanRemote != cleanCurrent && cleanCurrent != "" {
		w.NewImage = &w.CurrentImage
		w.UpdateStatus = "update_available"
		return true, nil
	}

	w.NewImage = nil
	w.NewDigest = nil
	w.UpdateStatus = "up_to_date"
	return false, nil
}

// resolveRegistryHost parses the image reference and extracts the registry domain
func resolveRegistryHost(imageRef string) string {
	parts := strings.Split(imageRef, "/")
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
