package models

type ForwardType string

const (
	ForwardLocal   ForwardType = "local"
	ForwardRemote  ForwardType = "remote"
	ForwardDynamic ForwardType = "dynamic"
)

type PortForward struct {
	ID           string      `json:"id"`
	ConnectionID string      `json:"connection_id"`
	Type         ForwardType `json:"type"`
	LocalHost    string      `json:"local_host"`
	LocalPort    int         `json:"local_port"`
	RemoteHost   string      `json:"remote_host"`
	RemotePort   int         `json:"remote_port"`
	AutoStart    bool        `json:"auto_start"`
}
