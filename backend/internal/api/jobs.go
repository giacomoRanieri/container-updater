package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"container-updater/backend/internal/db"
	"container-updater/backend/internal/docker"
	"container-updater/backend/internal/git"
	"container-updater/backend/internal/k8s"
	"container-updater/backend/internal/logger"
	"container-updater/backend/internal/notify"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TriggerUpdateResponse struct {
	JobID   string `json:"job_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func HandleTriggerUpdate(w http.ResponseWriter, r *http.Request) {
	workloadID := chi.URLParam(r, "id")
	if workloadID == "" {
		http.Error(w, "Missing workload ID parameter", http.StatusBadRequest)
		return
	}

	// 1. Fetch workload from database
	workload, err := db.GetWorkload(workloadID)
	if err != nil {
		logger.Log.Error("failed to retrieve workload from DB", "id", workloadID, "error", err)
		http.Error(w, "Failed to retrieve workload details", http.StatusInternalServerError)
		return
	}

	if workload == nil {
		http.Error(w, "Workload not found", http.StatusNotFound)
		return
	}

	// Verify if update is actually available
	if workload.UpdateStatus != "update_available" && workload.UpdateStatus != "failed" {
		http.Error(w, "No update available or workload is already updating", http.StatusBadRequest)
		return
	}

	if workload.NewImage == nil || *workload.NewImage == "" || workload.NewDigest == nil || *workload.NewDigest == "" {
		http.Error(w, "Invalid workload update target metadata", http.StatusBadRequest)
		return
	}

	// 2. Generate unique job details
	jobID := "job-" + uuid.New().String()[:8]
	job := &db.UpdateJob{
		ID:         jobID,
		WorkloadID: workloadID,
		Status:     "pending",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := db.SaveUpdateJob(job); err != nil {
		logger.Log.Error("failed to create job log in DB", "job_id", jobID, "error", err)
		http.Error(w, "Failed to initialize update job log", http.StatusInternalServerError)
		return
	}

	// 3. Spawn background execution goroutine
	go executeUpdateAsync(jobID, workload)

	// 4. Return 202 Accepted
	response := TriggerUpdateResponse{
		JobID:   jobID,
		Status:  "pending",
		Message: "Update job scheduled successfully.",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
}

func executeUpdateAsync(jobID string, w *db.Workload) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	logger.Log.Info("starting asynchronous update execution", "job_id", jobID, "workload", w.Name)

	// 1. Transition Job & Workload to active updating states
	db.UpdateJobStatus(jobID, "in_progress", "")
	db.UpdateWorkloadStatus(w.ID, "updating")

	BroadcastEvent("job_status", map[string]any{
		"job_id":           jobID,
		"status":           "in_progress",
		"percent_complete": 10,
		"message":          "Preparing update environments...",
	})
	BroadcastEvent("workload_updated", map[string]any{
		"id":            w.ID,
		"update_status": "updating",
	})

	var updateErr error

	if w.OrchestratorType == "compose" {
		// --- DOCKER COMPOSE FLOW ---
		containerID := strings.TrimPrefix(w.ID, "docker-")
		
		BroadcastEvent("job_status", map[string]any{
			"job_id":           jobID,
			"status":           "in_progress",
			"percent_complete": 40,
			"message":          fmt.Sprintf("Pulling registry image: %s", *w.NewImage),
		})

		// Modify compose file and run docker compose up -d
		updateErr = docker.UpdateComposeService(ctx, containerID, *w.NewImage)

	} else if w.OrchestratorType == "kubernetes" {
		// --- KUBERNETES FLOW ---
		
		// Parse namespace, workloadType, workloadName, containerName
		namespace, wType, wName, cName, parseErr := parseK8sWorkloadID(w.ID)
		if parseErr != nil {
			updateErr = parseErr
		} else {
			BroadcastEvent("job_status", map[string]any{
				"job_id":           jobID,
				"status":           "in_progress",
				"percent_complete": 40,
				"message":          fmt.Sprintf("Updating manifest on disk for container: %s", cName),
			})

			// 1. Update manifest on disk (and optionally apply to cluster directly)
			updateErr = k8s.UpdateK8sWorkload(ctx, namespace, wType, wName, cName, *w.NewImage)

			// 2. If manifest edit succeeded and GitOps Git is enabled, commit and push manifest
			// We will fully implement git package in subsequent phases
			if updateErr == nil {
				gitOpsErr := git.ProcessGitOpsCommit(ctx, wName, *w.NewImage)
				if gitOpsErr != nil {
					logger.Log.Error("GitOps commit failed", "workload", wName, "error", gitOpsErr)
					updateErr = fmt.Errorf("GitOps sync failed: %w", gitOpsErr)
				}
			}
		}
	} else {
		updateErr = fmt.Errorf("unsupported orchestrator type: %s", w.OrchestratorType)
	}

	// 2. Process Execution Results
	if updateErr != nil {
		logger.Log.Error("asynchronous update execution failed", "job_id", jobID, "error", updateErr)
		
		// Set status to failed
		db.UpdateJobStatus(jobID, "failed", updateErr.Error())
		db.UpdateWorkloadStatus(w.ID, "failed")

		BroadcastEvent("job_status", map[string]any{
			"job_id":           jobID,
			"status":           "failed",
			"percent_complete": 100,
			"message":          fmt.Sprintf("Update failed: %s", updateErr.Error()),
		})
		BroadcastEvent("workload_updated", map[string]any{
			"id":            w.ID,
			"update_status": "failed",
		})

		// Dispatch failure notification
		title := fmt.Sprintf("Update Failed: %s", w.Name)
		body := fmt.Sprintf("Failed to update workload **%s** to image `%s`.\n\n**Error**: %s", w.Name, *w.NewImage, updateErr.Error())
		notify.SendNotification(ctx, title, body)
		return
	}

	// --- SUCCESS OUTCOME ---
	// Update workload details
	w.CurrentImage = *w.NewImage
	w.CurrentDigest = *w.NewDigest
	w.NewImage = nil
	w.NewDigest = nil
	w.UpdateStatus = "up_to_date"
	now := time.Now()
	w.LastUpdatedAt = &now

	if err := db.SaveWorkload(w); err != nil {
		logger.Log.Error("failed to save post-update workload details in DB", "id", w.ID, "error", err)
	}

	db.UpdateJobStatus(jobID, "completed", "")

	BroadcastEvent("job_status", map[string]any{
		"job_id":           jobID,
		"status":           "completed",
		"percent_complete": 100,
		"message":          "Workload updated successfully!",
	})
	BroadcastEvent("workload_updated", map[string]any{
		"id":            w.ID,
		"update_status": "up_to_date",
	})

	// Dispatch success notification
	title := fmt.Sprintf("Update Completed: %s", w.Name)
	body := fmt.Sprintf("Workload **%s** has been successfully updated to version `%s`.", w.Name, w.CurrentImage)
	notify.SendNotification(ctx, title, body)
}

func parseK8sWorkloadID(id string) (string, string, string, string, error) {
	if !strings.HasPrefix(id, "k8s-") {
		return "", "", "", "", fmt.Errorf("invalid kubernetes workload ID format: %s", id)
	}

	trimmed := strings.TrimPrefix(id, "k8s-")
	var indicator string
	var wType string

	if strings.Contains(trimmed, "-deploy-") {
		indicator = "-deploy-"
		wType = "Deployment"
	} else if strings.Contains(trimmed, "-sts-") {
		indicator = "-sts-"
		wType = "StatefulSet"
	} else if strings.Contains(trimmed, "-ds-") {
		indicator = "-ds-"
		wType = "DaemonSet"
	}

	if indicator == "" {
		return "", "", "", "", fmt.Errorf("unknown workload orchestrator type in ID: %s", id)
	}

	parts := strings.SplitN(trimmed, indicator, 2)
	namespace := parts[0]
	rest := parts[1]

	// Find the last hyphen separating workloadName and containerName
	lastIdx := strings.LastIndex(rest, "-")
	if lastIdx == -1 {
		return "", "", "", "", fmt.Errorf("failed to parse workload name and container name from: %s", rest)
	}

	wName := rest[:lastIdx]
	cName := rest[lastIdx+1:]

	return namespace, wType, wName, cName, nil
}
