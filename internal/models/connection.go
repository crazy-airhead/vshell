package models

import "time"

type AuthType string

const (
	AuthPassword    AuthType = "password"
	AuthPrivateKey  AuthType = "private_key"
	AuthAgent       AuthType = "agent"
	AuthInteractive AuthType = "interactive"
)

type Connection struct {
	ID            string   `json:"id"`
	GroupID       *string  `json:"group_id"`
	Name          string   `json:"name"`
	Host          string   `json:"host"`
	Port          int      `json:"port"`
	Username      string   `json:"username"`
	AuthType      AuthType `json:"auth_type"`
	Password      string   `json:"-"` // AES encrypted, never expose to frontend
	PrivateKey    string   `json:"-"` // AES encrypted
	KeyPassphrase string   `json:"-"` // AES encrypted
	// KeyName records which managed key file (~/.ssh/<name>) the private key
	// was taken from, so the edit form can restore the key source UI.
	KeyName    *string    `json:"key_name,omitempty"`
	ProxyType  *string    `json:"proxy_type"`
	ProxyAddr  *string    `json:"proxy_addr"`
	JumpHostID *string    `json:"jump_host_id"`
	UploadPath string     `json:"upload_path"`
	DefaultCmd *string    `json:"default_cmd"`
	SortOrder  int        `json:"sort_order"`
	Color      *string    `json:"color"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// ConnectionForm is used for creating/updating connections from frontend.
type ConnectionForm struct {
	ID            string   `json:"id"`
	GroupID       *string  `json:"group_id"`
	Name          string   `json:"name"`
	Host          string   `json:"host"`
	Port          int      `json:"port"`
	Username      string   `json:"username"`
	AuthType      AuthType `json:"auth_type"`
	Password      string   `json:"password,omitempty"`
	PrivateKey    string   `json:"private_key,omitempty"`
	KeyPassphrase string   `json:"key_passphrase,omitempty"`
	KeyName       *string  `json:"key_name,omitempty"`
	ProxyType     *string  `json:"proxy_type"`
	ProxyAddr     *string  `json:"proxy_addr"`
	JumpHostID    *string  `json:"jump_host_id"`
	UploadPath    string   `json:"upload_path"`
	DefaultCmd    *string  `json:"default_cmd"`
	SortOrder     int      `json:"sort_order"`
	Color         *string  `json:"color"`
}
