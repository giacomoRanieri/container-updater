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

	"github.com/docker/docker/api/types/registry"
)

// CheckWorkloadUpdate compares the running digest of a workload with its remote registry digest
func CheckWorkloadUpdate(ctx context.Context, dockerClient *docker.DockerClient, w *db.Workload) (bool, error) {
	// 1. Resolve registry host from image name
	registryHost := resolveRegistryHost(w.CurrentImage)

	// 2. Fetch registry credentials from DB if they exist
	var username, password string
	query := `SELECT username, password FROM registry_credentials WHERE server_address = ?`
	err := db.DB.QueryRow(query, registryHost).Scan(&username, &password)
	
	var encodedAuth string
	if err == nil {
		authConfig := registry.AuthConfig{
			Username:      username,
			Password:      password,
			ServerAddress: registryHost,
		}
		authBytes, err := json.Marshal(authConfig)
		if err == nil {
			encodedAuth = base64.URLEncoding.EncodeToString(authBytes)
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		logger.Log.Error("database query failed for registry credentials", "registry", registryHost, "error", err)
	}

	// 3. Resolve the remote image reference (prepend docker.io if needed)
	fullImageName := w.CurrentImage
	if !strings.Contains(fullImageName, "/") {
		fullImageName = "docker.io/library/" + fullImageName
	} else if !strings.Contains(strings.Split(fullImageName, "/")[0], ".") && !strings.Contains(strings.Split(fullImageName, "/")[0], ":") {
		// e.g. "username/repo:tag" -> "docker.io/username/repo:tag"
		fullImageName = "docker.io/" + fullImageName
	}

	// 4. Query distribution inspect to get remote digest
	logger.Log.Debug("inspecting distribution info from registry", "image", fullImageName, "host", registryHost)
	inspect, err := dockerClient.DistributionInspect(ctx, fullImageName, encodedAuth)
	if err != nil {
		return false, err
	}

	remoteDigest := string(inspect.Descriptor.Digest)
	if remoteDigest == "" {
		return false, errors.New("empty remote digest returned from registry")
	}

	// Normalize digests (remove prefix like sha256: if present for strict comparison)
	cleanRemote := strings.TrimPrefix(remoteDigest, "sha256:")
	cleanCurrent := strings.TrimPrefix(w.CurrentDigest, "sha256:")

	logger.Log.Info("Compare image digests", "workload", w.Name, "current", cleanCurrent, "remote", cleanRemote)

	now := time.Now()
	w.LastCheckedAt = &now

	if cleanRemote != cleanCurrent {
		w.NewImage = &w.CurrentImage
		w.NewDigest = &remoteDigest
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
