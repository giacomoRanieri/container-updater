package monitor

import (
	"context"
	"fmt"
	"strings"

	"container-updater/backend/internal/config"
	"container-updater/backend/internal/db"
	"container-updater/backend/internal/docker"
	"container-updater/backend/internal/k8s"
	"container-updater/backend/internal/logger"
	"container-updater/backend/internal/notify"

	"github.com/robfig/cron/v3"
)

type MonitorScheduler struct {
	cron         *cron.Cron
	dockerClient *docker.DockerClient
	k8sClient    *k8s.K8sClient
}

func NewMonitorScheduler() (*MonitorScheduler, error) {
	dClient, err := docker.NewDockerClient()
	if err != nil {
		logger.Log.Warn("Docker client initialization skipped or failed (is Docker running?)", "error", err)
	}

	kClient, err := k8s.NewK8sClient()
	if err != nil {
		logger.Log.Warn("Kubernetes client initialization skipped or failed", "error", err)
	}

	return &MonitorScheduler{
		cron:         cron.New(),
		dockerClient: dClient,
		k8sClient:    kClient,
	}, nil
}

func (s *MonitorScheduler) Start(ctx context.Context) error {
	schedule := config.GlobalConfig.CronSchedule
	logger.Log.Info("Starting monitor scheduler", "schedule", schedule)

	_, err := s.cron.AddFunc(schedule, func() {
		s.RunCheck(ctx)
	})
	if err != nil {
		return fmt.Errorf("failed to schedule cron function: %w", err)
	}

	s.cron.Start()
	return nil
}

func (s *MonitorScheduler) Stop() {
	if s.cron != nil {
		s.cron.Stop()
	}
	if s.dockerClient != nil {
		s.dockerClient.Close()
	}
}

// RunCheck executes a full scan of Docker and Kubernetes workloads
func (s *MonitorScheduler) RunCheck(ctx context.Context) {
	logger.Log.Info("Scheduled workload check sequence started...")

	var detectedWorkloads []*db.Workload

	// 1. Scan Docker containers
	if s.dockerClient != nil {
		dockerWorkloads, err := s.dockerClient.ListMonitoredContainers(ctx)
		if err == nil {
			detectedWorkloads = append(detectedWorkloads, dockerWorkloads...)
		} else {
			logger.Log.Error("Docker scan failed", "error", err)
		}
	}

	// 2. Scan Kubernetes workloads
	if s.k8sClient != nil {
		k8sWorkloads, err := s.k8sClient.ListMonitoredWorkloads(ctx)
		if err == nil {
			detectedWorkloads = append(detectedWorkloads, k8sWorkloads...)
		} else {
			logger.Log.Error("Kubernetes scan failed", "error", err)
		}
	}

	// 3. Process each workload
	for _, w := range detectedWorkloads {
		// Read existing state from database if present
		existing, err := db.GetWorkload(w.ID)
		if err != nil {
			logger.Log.Error("failed to fetch workload from DB", "id", w.ID, "error", err)
			continue
		}

		if existing != nil {
			// Retain fields like last updated
			w.LastUpdatedAt = existing.LastUpdatedAt
		}

		// Run comparison check against registry
		if s.dockerClient != nil {
			isNewUpdate, err := CheckWorkloadUpdate(ctx, s.dockerClient, w)
			if err != nil {
				logger.Log.Error("registry check failed for workload", "name", w.Name, "error", err)
				w.UpdateStatus = "failed"
			} else if isNewUpdate {
				// Only notify if update status changed to 'update_available' or if we hadn't notified yet
				if existing == nil || existing.UpdateStatus != "update_available" {
					logger.Log.Info("new update detected, preparing notification...", "workload", w.Name)
					s.notifyUpdate(ctx, w)
				}
			}
		}

		// Save/persist to DB
		if err := db.SaveWorkload(w); err != nil {
			logger.Log.Error("failed to save workload details to DB", "id", w.ID, "error", err)
		}
	}

	logger.Log.Info("Scheduled workload check sequence completed.")
}

func (s *MonitorScheduler) notifyUpdate(ctx context.Context, w *db.Workload) {
	title := fmt.Sprintf("Update Available for %s", w.Name)
	
	body := fmt.Sprintf(`### Container Update Detected 🚀

* **Workload Name**: %s
* **Orchestrator**: %s
* **Namespace/Project**: %s
* **Current Image**: %s
* **Current Digest**: %s
* **New Digest**: %s
`, w.Name, w.OrchestratorType, w.NamespaceProject, w.CurrentImage, w.CurrentDigest, *w.NewDigest)

	// Fetch changelog if image has a linked GitHub source in container image inspect labels
	// We check for Docker containers
	if w.OrchestratorType == "compose" && s.dockerClient != nil {
		// Retrieve container ID (extracted from db.Workload.ID "docker-{container_id}")
		containerID := strings.TrimPrefix(w.ID, "docker-")
		inspect, err := s.dockerClient.API().ContainerInspect(ctx, containerID)
		if err == nil {
			imgInspect, _, err := s.dockerClient.API().ImageInspectWithRaw(ctx, inspect.Image)
			if err == nil {
				githubURL := notify.ExtractGithubSource(imgInspect.Config.Labels)
				if githubURL != "" {
					logger.Log.Info("fetching release notes...", "url", githubURL)
					changelog, err := notify.FetchLatestReleaseNotes(ctx, githubURL)
					if err == nil {
						body = body + "\n---\n\n" + changelog
					} else {
						logger.Log.Warn("failed to fetch release notes from GitHub", "url", githubURL, "error", err)
					}
				}
			}
		}
	}

	// Dispatch notification via Apprise
	if err := notify.SendNotification(ctx, title, body); err != nil {
		logger.Log.Error("notification delivery failed", "error", err)
	}
}
