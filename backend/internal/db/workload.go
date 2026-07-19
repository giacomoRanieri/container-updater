package db

import (
	"database/sql"
	"errors"
	"time"
)

type Workload struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	NamespaceProject string     `json:"namespace_project"`
	OrchestratorType string     `json:"orchestrator_type"` // "compose" or "kubernetes"
	CurrentImage     string     `json:"current_image"`
	CurrentDigest    string     `json:"current_digest"`
	NewImage         *string    `json:"new_image,omitempty"`
	NewDigest        *string    `json:"new_digest,omitempty"`
	UpdateStatus     string     `json:"update_status"` // 'up_to_date', 'update_available', 'updating', 'failed'
	LastCheckedAt    *time.Time `json:"last_checked_at,omitempty"`
	LastUpdatedAt    *time.Time `json:"last_updated_at,omitempty"`
}

func SaveWorkload(w *Workload) error {
	query := `
		INSERT INTO workloads (
			id, name, namespace_project, orchestrator_type, current_image, current_digest,
			new_image, new_digest, update_status, last_checked_at, last_updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name,
			namespace_project=excluded.namespace_project,
			orchestrator_type=excluded.orchestrator_type,
			current_image=excluded.current_image,
			current_digest=excluded.current_digest,
			new_image=excluded.new_image,
			new_digest=excluded.new_digest,
			update_status=excluded.update_status,
			last_checked_at=excluded.last_checked_at,
			last_updated_at=excluded.last_updated_at;
	`
	_, err := DB.Exec(
		query,
		w.ID, w.Name, w.NamespaceProject, w.OrchestratorType, w.CurrentImage, w.CurrentDigest,
		w.NewImage, w.NewDigest, w.UpdateStatus, w.LastCheckedAt, w.LastUpdatedAt,
	)
	return err
}

func GetWorkload(id string) (*Workload, error) {
	query := `
		SELECT id, name, namespace_project, orchestrator_type, current_image, current_digest,
		       new_image, new_digest, update_status, last_checked_at, last_updated_at
		FROM workloads WHERE id = ?
	`
	row := DB.QueryRow(query, id)

	w := &Workload{}
	var lastChecked, lastUpdated sql.NullTime
	err := row.Scan(
		&w.ID, &w.Name, &w.NamespaceProject, &w.OrchestratorType, &w.CurrentImage, &w.CurrentDigest,
		&w.NewImage, &w.NewDigest, &w.UpdateStatus, &lastChecked, &lastUpdated,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if lastChecked.Valid {
		w.LastCheckedAt = &lastChecked.Time
	}
	if lastUpdated.Valid {
		w.LastUpdatedAt = &lastUpdated.Time
	}

	return w, nil
}

func ListWorkloads() ([]*Workload, error) {
	query := `
		SELECT id, name, namespace_project, orchestrator_type, current_image, current_digest,
		       new_image, new_digest, update_status, last_checked_at, last_updated_at
		FROM workloads
	`
	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Workload
	for rows.Next() {
		w := &Workload{}
		var lastChecked, lastUpdated sql.NullTime
		err := rows.Scan(
			&w.ID, &w.Name, &w.NamespaceProject, &w.OrchestratorType, &w.CurrentImage, &w.CurrentDigest,
			&w.NewImage, &w.NewDigest, &w.UpdateStatus, &lastChecked, &lastUpdated,
		)
		if err != nil {
			return nil, err
		}
		if lastChecked.Valid {
			w.LastCheckedAt = &lastChecked.Time
		}
		if lastUpdated.Valid {
			w.LastUpdatedAt = &lastUpdated.Time
		}
		list = append(list, w)
	}

	return list, nil
}

func UpdateWorkloadStatus(id string, status string) error {
	query := `UPDATE workloads SET update_status = ?, last_updated_at = ? WHERE id = ?`
	_, err := DB.Exec(query, status, time.Now(), id)
	return err
}

func DeleteWorkload(id string) error {
	query := `DELETE FROM workloads WHERE id = ?`
	_, err := DB.Exec(query, id)
	return err
}
