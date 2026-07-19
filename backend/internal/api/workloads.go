package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"container-updater/backend/internal/db"
	"container-updater/backend/internal/logger"

	"github.com/google/uuid"
)

type AuditLogItem struct {
	ID           string    `json:"id"`
	WorkloadID   string    `json:"workload_id"`
	WorkloadName string    `json:"workload_name"`
	Status       string    `json:"status"`
	ErrorMessage string    `json:"error_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type AppStats struct {
	TotalWorkloads    int        `json:"total_workloads"`
	UpdatesAvailable  int        `json:"updates_available"`
	SuccessfulUpdates int        `json:"successful_updates"`
	LastCheckAt       *time.Time `json:"last_check_at,omitempty"`
}

func HandleListWorkloads(w http.ResponseWriter, r *http.Request) {
	workloads, err := db.ListWorkloads()
	if err != nil {
		logger.Log.Error("failed to retrieve workloads from database", "error", err)
		http.Error(w, "Failed to retrieve workloads", http.StatusInternalServerError)
		return
	}

	// Default to empty array rather than null in JSON
	if workloads == nil {
		workloads = []*db.Workload{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workloads)
}

func HandleListNotifications(w http.ResponseWriter, r *http.Request) {
	services, err := db.ListNotificationServices()
	if err != nil {
		logger.Log.Error("failed to list notification services", "error", err)
		http.Error(w, "Failed to retrieve notifications configuration", http.StatusInternalServerError)
		return
	}

	if services == nil {
		services = []*db.NotificationService{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(services)
}

type SaveNotificationInput struct {
	ID         string `json:"id,omitempty"`
	Name       string `json:"name"`
	AppriseURL string `json:"apprise_url"`
	IsEnabled  bool   `json:"is_enabled"`
}

func HandleSaveNotification(w http.ResponseWriter, r *http.Request) {
	var input SaveNotificationInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if input.Name == "" || input.AppriseURL == "" {
		http.Error(w, "name and apprise_url are required fields", http.StatusBadRequest)
		return
	}

	id := input.ID
	if id == "" {
		id = "notify-" + uuid.New().String()[:8]
	}

	service := &db.NotificationService{
		ID:         id,
		Name:       input.Name,
		AppriseURL: input.AppriseURL,
		IsEnabled:  input.IsEnabled,
		CreatedAt:  time.Now(),
	}

	if err := db.SaveNotificationService(service); err != nil {
		logger.Log.Error("failed to save notification service", "id", id, "error", err)
		http.Error(w, "Failed to save configuration", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(service)
}

func HandleListAuditLogs(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT j.id, j.workload_id, w.name, j.status, j.error_message, j.created_at, j.updated_at
		FROM update_jobs j
		JOIN workloads w ON j.workload_id = w.id
		ORDER BY j.created_at DESC
	`
	rows, err := db.DB.Query(query)
	if err != nil {
		logger.Log.Error("failed to query update jobs from DB", "error", err)
		http.Error(w, "Failed to retrieve audit logs", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var logs []AuditLogItem
	for rows.Next() {
		var item AuditLogItem
		var errMsg sql.NullString
		err := rows.Scan(
			&item.ID, &item.WorkloadID, &item.WorkloadName, &item.Status, &errMsg, &item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			logger.Log.Error("failed to scan audit log row", "error", err)
			http.Error(w, "Failed to parse audit logs", http.StatusInternalServerError)
			return
		}
		if errMsg.Valid {
			item.ErrorMessage = errMsg.String
		}
		logs = append(logs, item)
	}

	if logs == nil {
		logs = []AuditLogItem{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

func HandleGetStats(w http.ResponseWriter, r *http.Request) {
	var stats AppStats

	// 1. Get total workloads and updates available
	queryWorkloads := `
		SELECT COUNT(*), SUM(CASE WHEN update_status = 'update_available' THEN 1 ELSE 0 END), MAX(last_checked_at)
		FROM workloads
	`
	var lastCheckedVal interface{}
	var total, updates sql.NullInt64
	err := db.DB.QueryRow(queryWorkloads).Scan(&total, &updates, &lastCheckedVal)
	if err != nil {
		logger.Log.Error("failed to calculate workload stats", "error", err)
		http.Error(w, "Failed to fetch stats", http.StatusInternalServerError)
		return
	}
	stats.TotalWorkloads = int(total.Int64)
	stats.UpdatesAvailable = int(updates.Int64)
	if lastCheckedVal != nil {
		switch v := lastCheckedVal.(type) {
		case time.Time:
			stats.LastCheckAt = &v
		case string:
			if t, err := parseSQLiteTime(v); err == nil {
				stats.LastCheckAt = &t
			}
		case []byte:
			if t, err := parseSQLiteTime(string(v)); err == nil {
				stats.LastCheckAt = &t
			}
		}
	}

	// 2. Get total successful updates
	querySuccessfulJobs := `
		SELECT COUNT(*) FROM update_jobs WHERE status = 'completed'
	`
	var successfulJobs int
	err = db.DB.QueryRow(querySuccessfulJobs).Scan(&successfulJobs)
	if err != nil {
		logger.Log.Error("failed to count successful update jobs", "error", err)
		http.Error(w, "Failed to fetch stats", http.StatusInternalServerError)
		return
	}
	stats.SuccessfulUpdates = successfulJobs

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func parseSQLiteTime(val string) (time.Time, error) {
	if val == "" {
		return time.Time{}, fmt.Errorf("empty time string")
	}
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, val); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse sqlite time string: %s", val)
}
