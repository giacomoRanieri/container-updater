package db

import (
	"database/sql"
	"errors"
	"time"
)

type RegistryCredential struct {
	ID            string    `json:"id"`
	ServerAddress string    `json:"server_address"`
	Username      string    `json:"username"`
	Password      string    `json:"password"`
	CreatedAt     time.Time `json:"created_at"`
}

func SaveRegistryCredential(c *RegistryCredential) error {
	query := `
		INSERT INTO registry_credentials (id, server_address, username, password, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			server_address=excluded.server_address,
			username=excluded.username,
			password=excluded.password;
	`
	_, err := DB.Exec(query, c.ID, c.ServerAddress, c.Username, c.Password, c.CreatedAt)
	return err
}

func ListRegistryCredentials() ([]*RegistryCredential, error) {
	query := `SELECT id, server_address, username, password, created_at FROM registry_credentials ORDER BY server_address ASC`
	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*RegistryCredential
	for rows.Next() {
		c := &RegistryCredential{}
		var createdAtVal interface{}
		if err := rows.Scan(&c.ID, &c.ServerAddress, &c.Username, &c.Password, &createdAtVal); err != nil {
			return nil, err
		}
		if t, err := ParseTime(createdAtVal); err == nil {
			c.CreatedAt = t
		}
		list = append(list, c)
	}
	return list, nil
}

func DeleteRegistryCredential(id string) error {
	query := `DELETE FROM registry_credentials WHERE id = ?`
	_, err := DB.Exec(query, id)
	return err
}

func GetRegistryCredential(id string) (*RegistryCredential, error) {
	query := `SELECT id, server_address, username, password, created_at FROM registry_credentials WHERE id = ?`
	row := DB.QueryRow(query, id)

	c := &RegistryCredential{}
	var createdAtVal interface{}
	err := row.Scan(&c.ID, &c.ServerAddress, &c.Username, &c.Password, &createdAtVal)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if t, err := ParseTime(createdAtVal); err == nil {
		c.CreatedAt = t
	}
	return c, nil
}

func MaskRegistryPassword(password string) string {
	if password == "" {
		return ""
	}
	return "[set]"
}
