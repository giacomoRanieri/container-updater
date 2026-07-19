package db

import (
	"fmt"
	"time"
)

// ParseTime parses timestamps from SQLite driver safely across time.Time, string, and []byte types.
func ParseTime(val interface{}) (time.Time, error) {
	if val == nil {
		return time.Time{}, nil
	}
	switch v := val.(type) {
	case time.Time:
		return v, nil
	case string:
		return parseTimeString(v)
	case []byte:
		return parseTimeString(string(v))
	default:
		return time.Time{}, fmt.Errorf("unsupported time type: %T", val)
	}
}

func parseTimeString(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
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
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse time string: %s", s)
}
