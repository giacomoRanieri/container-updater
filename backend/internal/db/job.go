package db

import (
	"database/sql"
	"errors"
	"time"
)

type UpdateJob struct {
	ID           string    `json:"id"`
	WorkloadID   string    `json:"workload_id"`
	Status       string    `json:"status"` // 'pending', 'in_progress', 'completed', 'failed', 'rolled_back'
	ErrorMessage string    `json:"error_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func SaveUpdateJob(j *UpdateJob) error {
	query := `
		INSERT INTO update_jobs (id, workload_id, status, error_message, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status=excluded.status,
			error_message=excluded.error_message,
			updated_at=excluded.updated_at;
	`
	var errMsg sql.NullString
	if j.ErrorMessage != "" {
		errMsg = sql.NullString{String: j.ErrorMessage, Valid: true}
	}

	_, err := DB.Exec(query, j.ID, j.WorkloadID, j.Status, errMsg, j.CreatedAt, j.UpdatedAt)
	return err
}

func GetUpdateJob(id string) (*UpdateJob, error) {
	query := `SELECT id, workload_id, status, error_message, created_at, updated_at FROM update_jobs WHERE id = ?`
	row := DB.QueryRow(query, id)

	j := &UpdateJob{}
	var errMsg sql.NullString
	var createdAtVal, updatedAtVal interface{}
	err := row.Scan(&j.ID, &j.WorkloadID, &j.Status, &errMsg, &createdAtVal, &updatedAtVal)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if t, err := ParseTime(createdAtVal); err == nil {
		j.CreatedAt = t
	}
	if t, err := ParseTime(updatedAtVal); err == nil {
		j.UpdatedAt = t
	}

	if errMsg.Valid {
		j.ErrorMessage = errMsg.String
	}
	return j, nil
}

func UpdateJobStatus(id string, status string, errorMessage string) error {
	var errMsg sql.NullString
	if errorMessage != "" {
		errMsg = sql.NullString{String: errorMessage, Valid: true}
	}

	query := `UPDATE update_jobs SET status = ?, error_message = ?, updated_at = ? WHERE id = ?`
	_, err := DB.Exec(query, status, errMsg, time.Now(), id)
	return err
}
