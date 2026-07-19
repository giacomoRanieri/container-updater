package docker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"container-updater/backend/internal/logger"

	"gopkg.in/yaml.v3"
)

// UpdateComposeService updates the image in the docker-compose.yml file and runs docker compose up -d
func UpdateComposeService(ctx context.Context, containerID string, newImage string) error {
	// 1. Initialize Docker Client to inspect container labels
	dClient, err := NewDockerClient()
	if err != nil {
		return fmt.Errorf("failed to create docker client: %w", err)
	}
	defer dClient.Close()

	inspect, err := dClient.API().ContainerInspect(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to inspect container %s: %w", containerID, err)
	}

	// 2. Read compose metadata labels
	composeFileHostPath := inspect.Config.Labels["com.docker.compose.project.config_files"]
	serviceName := inspect.Config.Labels["com.docker.compose.service"]

	if composeFileHostPath == "" || serviceName == "" {
		return errors.New("container does not have docker compose metadata labels (com.docker.compose.project.config_files or com.docker.compose.service)")
	}

	// 3. Map host path to container path if mapping is configured
	// e.g. COMPOSE_PATH_MAP="/home/user/apps:/apps"
	composeFileLocalPath := mapHostPathToLocal(composeFileHostPath)
	logger.Log.Info("modifying compose file", "host_path", composeFileHostPath, "local_path", composeFileLocalPath, "service", serviceName, "new_image", newImage)

	// 4. Update the docker-compose.yml file on disk
	if err := updateComposeYAML(composeFileLocalPath, serviceName, newImage); err != nil {
		return fmt.Errorf("failed to update compose yaml file: %w", err)
	}

	// 5. Execute docker compose up -d
	logger.Log.Info("executing docker compose command...", "file", composeFileLocalPath, "service", serviceName)
	
	// We want to run: docker compose -f <file> up -d <service>
	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", composeFileLocalPath, "up", "-d", serviceName)
	
	// Ensure it uses host socket
	cmd.Env = os.Environ()
	
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err = cmd.Run()
	duration := time.Since(start)

	logger.Log.Debug("docker compose output", "stdout", stdout.String(), "stderr", stderr.String(), "duration", duration.String())

	if err != nil {
		return fmt.Errorf("docker compose command failed: %s; error: %w", stderr.String(), err)
	}

	logger.Log.Info("docker compose service updated successfully", "container", containerID, "service", serviceName)
	return nil
}

func mapHostPathToLocal(hostPath string) string {
	pathMap := os.Getenv("COMPOSE_PATH_MAP") // format: "/host/path:/container/path"
	if pathMap == "" {
		return hostPath
	}

	parts := strings.Split(pathMap, ":")
	if len(parts) != 2 {
		logger.Log.Warn("invalid COMPOSE_PATH_MAP environment variable format, expected host_prefix:container_prefix")
		return hostPath
	}

	hostPrefix := parts[0]
	containerPrefix := parts[1]

	// If the file path matches the host prefix, replace it
	if strings.HasPrefix(hostPath, hostPrefix) {
		return containerPrefix + strings.TrimPrefix(hostPath, hostPrefix)
	}

	return hostPath
}

func updateComposeYAML(filePath string, serviceName string, newImage string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Unmarshal to raw node to preserve structure & comments if possible, 
	// but to make editing clean, unmarshal to generic map.
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return err
	}

	services, ok := doc["services"].(map[string]any)
	if !ok {
		return errors.New("compose file is missing the 'services' section")
	}

	serviceItem, ok := services[serviceName]
	if !ok {
		return fmt.Errorf("service '%s' not found in compose file services", serviceName)
	}

	serviceMap, ok := serviceItem.(map[string]any)
	if !ok {
		// Try parsing if it was read as map[any]any
		return fmt.Errorf("invalid format of service '%s' properties", serviceName)
	}

	// Set the new image
	serviceMap["image"] = newImage
	services[serviceName] = serviceMap
	doc["services"] = services

	// Write back
	output, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, output, 0644)
}
