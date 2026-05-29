package models

import "time"

type QuickCommand struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Command      string    `json:"command"`
	ConnectionID *string   `json:"connection_id"`
	SortOrder    int       `json:"sort_order"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
