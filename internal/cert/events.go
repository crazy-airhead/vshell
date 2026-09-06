package cert

// LogEvent carries one output line of a streamed remote command (event
// "cert:log"). TaskID identifies local cert-task operations, OpID one-off
// operations like the acme.sh install.
type LogEvent struct {
	TaskID       string `json:"taskID,omitempty"`
	OpID         string `json:"opID,omitempty"`
	ConnectionID string `json:"connectionID"`
	Stream       string `json:"stream"` // "stdout" | "stderr"
	Line         string `json:"line"`
	Ts           int64  `json:"ts"` // unix milliseconds
}

// StageEvent marks progress through the orchestration stages (event
// "cert:stage").
type StageEvent struct {
	TaskID       string `json:"taskID,omitempty"`
	OpID         string `json:"opID,omitempty"`
	ConnectionID string `json:"connectionID"`
	Stage        string `json:"stage"`
	Status       string `json:"status"` // "start" | "ok" | "fail"
	Error        string `json:"error,omitempty"`
}

// Orchestration stages.
const (
	StageDetect      = "detect"
	StageInstall     = "install"
	StageIssue       = "issue"
	StageRenew       = "renew"
	StageInstallCert = "install-cert"
	StageCron        = "cron"
	StageRemove      = "remove"
	StageDone        = "done"
)

// Stage order for UI progress display.
var StageOrder = []string{
	StageDetect, StageInstall, StageIssue, StageRenew, StageInstallCert, StageCron, StageDone,
}
