package docker

import (
	"context"
	"strings"
	"time"

	"container-updater/backend/internal/db"
	"container-updater/backend/internal/logger"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
)

type DockerClient struct {
	cli *client.Client
}

func NewDockerClient() (*DockerClient, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &DockerClient{cli: cli}, nil
}

func (d *DockerClient) Close() {
	if d.cli != nil {
		d.cli.Close()
	}
}

func (d *DockerClient) API() *client.Client {
	return d.cli
}

func (d *DockerClient) DistributionInspect(ctx context.Context, imageRef, encodedAuth string) (registry.DistributionInspect, error) {
	return d.cli.DistributionInspect(ctx, imageRef, encodedAuth)
}

// ListMonitoredContainers returns a list of workloads representing Docker containers
// which have the label "container-updater.enable=true".
func (d *DockerClient) ListMonitoredContainers(ctx context.Context) ([]*db.Workload, error) {
	containers, err := d.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}

	var workloads []*db.Workload
	now := time.Now()

	for _, c := range containers {
		// Filter by label
		if c.Labels["container-updater.enable"] != "true" {
			continue
		}

		// Inspect container for detailed config & image ID
		inspect, err := d.cli.ContainerInspect(ctx, c.ID)
		if err != nil {
			logger.Log.Error("failed to inspect container", "id", c.ID, "error", err)
			continue
		}

		// Clean up container name (remove leading slash)
		name := strings.TrimPrefix(inspect.Name, "/")

		// Retrieve compose project or default to "default"
		project := inspect.Config.Labels["com.docker.compose.project"]
		if project == "" {
			project = "default"
		}

		// Determine current running image and its digest
		currentImageName := inspect.Config.Image // e.g. "nginx:1.25.0"
		currentDigest := inspect.Image           // Fallback to local Image ID (sha256:...)

		// Try to inspect the image to resolve repo digests
		imgInspect, _, err := d.cli.ImageInspectWithRaw(ctx, inspect.Image)
		if err == nil && len(imgInspect.RepoDigests) > 0 {
			// Find the digest matching the repository name if possible
			for _, rd := range imgInspect.RepoDigests {
				if strings.Contains(rd, "@sha256:") {
					parts := strings.Split(rd, "@")
					if len(parts) == 2 {
						currentDigest = parts[1]
						break
					}
				}
			}
		}

		// Set initial update status or keep existing
		status := "up_to_date"

		workloads = append(workloads, &db.Workload{
			ID:               "docker-" + c.ID[:12],
			Name:             name,
			NamespaceProject: project,
			OrchestratorType: "compose",
			CurrentImage:     currentImageName,
			CurrentDigest:    currentDigest,
			UpdateStatus:     status,
			LastCheckedAt:    &now,
		})
	}

	return workloads, nil
}
