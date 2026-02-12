package models

import "time"

// Preference stores a single user preference as a key-value pair
// with a JSON-encoded value.
type Preference struct {
	ID        int64     `json:"id"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}
