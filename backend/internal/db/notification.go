package db

import (
	"database/sql"
	"errors"
	"time"
)

type NotificationService struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	AppriseURL string    `json:"apprise_url"`
	IsEnabled  bool      `json:"is_enabled"`
	CreatedAt  time.Time `json:"created_at"`
}

func SaveNotificationService(n *NotificationService) error {
	query := `
		INSERT INTO notification_services (id, name, apprise_url, is_enabled, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name,
			apprise_url=excluded.apprise_url,
			is_enabled=excluded.is_enabled;
	`
	_, err := DB.Exec(query, n.ID, n.Name, n.AppriseURL, n.IsEnabled, n.CreatedAt)
	return err
}

func GetNotificationService(id string) (*NotificationService, error) {
	query := `SELECT id, name, apprise_url, is_enabled, created_at FROM notification_services WHERE id = ?`
	row := DB.QueryRow(query, id)

	n := &NotificationService{}
	var createdAtVal interface{}
	err := row.Scan(&n.ID, &n.Name, &n.AppriseURL, &n.IsEnabled, &createdAtVal)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if t, err := ParseTime(createdAtVal); err == nil {
		n.CreatedAt = t
	}
	return n, nil
}

func ListNotificationServices() ([]*NotificationService, error) {
	query := `SELECT id, name, apprise_url, is_enabled, created_at FROM notification_services`
	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*NotificationService
	for rows.Next() {
		n := &NotificationService{}
		var createdAtVal interface{}
		err := rows.Scan(&n.ID, &n.Name, &n.AppriseURL, &n.IsEnabled, &createdAtVal)
		if err != nil {
			return nil, err
		}
		if t, err := ParseTime(createdAtVal); err == nil {
			n.CreatedAt = t
		}
		list = append(list, n)
	}
	return list, nil
}

func DeleteNotificationService(id string) error {
	query := `DELETE FROM notification_services WHERE id = ?`
	_, err := DB.Exec(query, id)
	return err
}
