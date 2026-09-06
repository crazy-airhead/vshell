package models

import "time"

type CertStatus string

const (
	CertStatusIdle    CertStatus = "idle"
	CertStatusRunning CertStatus = "running"
	CertStatusIssued  CertStatus = "issued"
	CertStatusFailed  CertStatus = "failed"
)

// CertTask is a local record of a certificate managed by acme.sh on a remote
// server. Credentials are stored AES-encrypted and never leave the backend.
type CertTask struct {
	ID             string     `json:"id"`
	ConnectionID   string     `json:"connection_id"`
	ConnectionName string     `json:"connection_name"`
	ConnectionHost string     `json:"connection_host"`
	Name           string     `json:"name"`
	PrimaryDomain  string     `json:"primary_domain"`
	SANDomains     []string   `json:"san_domains"`
	DNSProvider    string     `json:"dns_provider"`
	DNSPlugin      string     `json:"dns_plugin,omitempty"`
	DNSCredentials string     `json:"-"` // AES encrypted JSON map, never expose to frontend
	KeyLength      string     `json:"key_length"`
	DNSSleep       int        `json:"dns_sleep"`
	TestMode       bool       `json:"test_mode"`
	AutoInstall    bool       `json:"auto_install"`
	CertDir        string     `json:"cert_dir,omitempty"`
	KeyFile        string     `json:"key_file,omitempty"`
	FullchainFile  string     `json:"fullchain_file,omitempty"`
	ReloadCmd      string     `json:"reload_cmd,omitempty"`
	LastStatus     CertStatus `json:"last_status"`
	LastError      string     `json:"last_error,omitempty"`
	LastRunAt      *time.Time `json:"last_run_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// CertTaskForm is used for creating/updating cert tasks from frontend.
// Empty DNSCredentials means "keep stored credentials" on update.
type CertTaskForm struct {
	ID             string            `json:"id"`
	ConnectionID   string            `json:"connection_id"`
	Name           string            `json:"name"`
	PrimaryDomain  string            `json:"primary_domain"`
	SANDomains     []string          `json:"san_domains"`
	DNSProvider    string            `json:"dns_provider"`
	DNSPlugin      string            `json:"dns_plugin,omitempty"`
	DNSCredentials map[string]string `json:"dns_credentials,omitempty"`
	KeyLength      string            `json:"key_length"`
	DNSSleep       int               `json:"dns_sleep"`
	TestMode       bool              `json:"test_mode"`
	AutoInstall    bool              `json:"auto_install"`
	CertDir        string            `json:"cert_dir,omitempty"`
	KeyFile        string            `json:"key_file,omitempty"`
	FullchainFile  string            `json:"fullchain_file,omitempty"`
	ReloadCmd      string            `json:"reload_cmd,omitempty"`
}

// RemoteCert is one row of `acme.sh --list` output, optionally enriched with
// `--info` data (NextRenewTime/DaysLeft).
type RemoteCert struct {
	MainDomain    string   `json:"main_domain"`
	KeyLength     string   `json:"key_length"`
	SANDomains    []string `json:"san_domains"`
	CA            string   `json:"ca"`
	Created       string   `json:"created"`
	Renew         string   `json:"renew"`
	NextRenewTime int64    `json:"next_renew_time"` // epoch seconds, 0 = unknown
	DaysLeft      *int     `json:"days_left"`       // nil = unknown
	ECC           bool     `json:"ecc"`
}

// RemoteCertInfo holds `acme.sh --info -d <domain>` key=value output.
type RemoteCertInfo struct {
	Domain           string            `json:"domain"`
	Fields           map[string]string `json:"fields"`
	NextRenewTime    int64             `json:"next_renew_time"`
	NextRenewTimeStr string            `json:"next_renew_time_str"`
	CertCreateTime   string            `json:"cert_create_time"`
	CertPath         string            `json:"cert_path"`
}

// CertEnvironment is the result of the remote detection script.
type CertEnvironment struct {
	ConnectionID string `json:"connection_id"`
	Home         string `json:"home"`
	AcmeShPath   string `json:"acme_sh_path"`
	Installed    bool   `json:"installed"`
	CronPresent  bool   `json:"cron_present"`
	CurlPresent  bool   `json:"curl_present"`
}

// DNSFieldSpec describes one credential input of a DNS provider.
type DNSFieldSpec struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Secret      bool   `json:"secret"`
	Required    bool   `json:"required"`
	Placeholder string `json:"placeholder"`
}

// DNSProvider is a registry entry used by the frontend to render the
// credential form dynamically.
type DNSProvider struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Plugin string         `json:"plugin"`
	Fields []DNSFieldSpec `json:"fields"`
}
