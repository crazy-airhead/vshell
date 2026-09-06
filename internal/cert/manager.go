package cert

import (
	"context"
	"sync"

	vshellssh "vshell/internal/ssh"
	"vshell/internal/sftp"
)

// Manager orchestrates acme.sh operations on remote servers over existing
// SSH connections. It only drives commands and parses output; persistence
// and credential encryption live in the app layer.
type Manager struct {
	ssh     *vshellssh.Manager
	sftp    *sftp.Manager
	onEvent func(event string, data any)

	mu      sync.Mutex
	running map[string]context.CancelFunc // taskID/opID -> cancel
}

// NewManager wires the cert manager to the shared SSH/SFTP managers.
func NewManager(sshMgr *vshellssh.Manager, sftpMgr *sftp.Manager, onEvent func(string, any)) *Manager {
	return &Manager{
		ssh:     sshMgr,
		sftp:    sftpMgr,
		onEvent: onEvent,
		running: make(map[string]context.CancelFunc),
	}
}

// Register marks an operation as running so it can be canceled and is not
// started twice.
func (m *Manager) Register(id string, cancel context.CancelFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.running[id] = cancel
}

// Unregister removes a finished operation.
func (m *Manager) Unregister(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.running, id)
}

// IsRunning reports whether an operation with this id is active.
func (m *Manager) IsRunning(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.running[id]
	return ok
}

// Cancel aborts a running operation, if any.
func (m *Manager) Cancel(id string) {
	m.mu.Lock()
	cancel, ok := m.running[id]
	m.mu.Unlock()
	if ok {
		cancel()
	}
}

// emitLog forwards one command output line to the frontend.
func (m *Manager) emitLog(e LogEvent) {
	if m.onEvent != nil {
		m.onEvent("cert:log", e)
	}
}

// emitStage reports a stage transition to the frontend.
func (m *Manager) emitStage(e StageEvent) {
	if m.onEvent != nil {
		m.onEvent("cert:stage", e)
	}
}

// stage emits a stage event with the given status/error.
func (m *Manager) stage(taskID, opID, connectionID, stage, status, errMsg string) {
	m.emitStage(StageEvent{
		TaskID:       taskID,
		OpID:         opID,
		ConnectionID: connectionID,
		Stage:        stage,
		Status:       status,
		Error:        errMsg,
	})
}

// stream runs a long command, forwarding every output line as a cert:log
// event. On failure the error message carries the command's exit code and
// the tail of its output for diagnostics.
func (m *Manager) stream(ctx context.Context, connectionID, taskID, opID, stage, cmd string, timeout int) (RunResult, error) {
	res, err := m.Run(ctx, connectionID, cmd, RunOptions{
		Timeout: seconds(timeout),
		OnLine: func(streamName, line string) {
			m.emitLog(LogEvent{
				TaskID:       taskID,
				OpID:         opID,
				ConnectionID: connectionID,
				Stream:       streamName,
				Line:         line,
				Ts:           nowMillis(),
			})
		},
	})
	if err != nil {
		return res, wrapStageError(stage, res, err)
	}
	return res, nil
}
