package app

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v3/pkg/application"
	"golang.org/x/crypto/ssh"

	"vshell/internal/db"
	"vshell/internal/models"
	"vshell/internal/portforward"
	"vshell/internal/sftp"
	vshellssh "vshell/internal/ssh"
)

type AppService struct {
	wailsApp    *application.App
	db          *db.DB
	sshManager  *vshellssh.Manager
	sftpManager *sftp.Manager
	fwdManager  *portforward.Manager
	monitors    map[string]*vshellssh.Monitor
}

func New() *AppService {
	return &AppService{}
}

// SetApp sets the Wails application reference. Called after application creation.
func (a *AppService) SetApp(app *application.App) {
	a.wailsApp = app
}

// ServiceStartup is called by Wails when the application starts.
func (a *AppService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	database, err := db.New()
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}
	a.db = database

	emit := func(event string, data any) {
		a.wailsApp.Event.Emit(event, data)
	}

	a.sshManager = vshellssh.NewManager(a.db.Crypto(), emit)
	a.sftpManager = sftp.NewManager(a.sshManager, a.db.Crypto(), emit)
	a.fwdManager = portforward.NewManager()
	a.monitors = make(map[string]*vshellssh.Monitor)

	// Listen for terminal stdin events from frontend
	a.wailsApp.Event.On("terminal:stdin", func(e *application.CustomEvent) {
		payload, _ := json.Marshal(e.Data)
		var msg struct {
			SessionID string `json:"sessionID"`
			Data      string `json:"data"`
		}
		if err := json.Unmarshal(payload, &msg); err != nil {
			return
		}
		if msg.SessionID != "" && msg.Data != "" {
			a.sshManager.WriteStdin(msg.SessionID, []byte(msg.Data))
		}
	})

	// Listen for local filesystem events
	a.wailsApp.Event.On("localfs:homedir", func(e *application.CustomEvent) {
		home, err := os.UserHomeDir()
		if err != nil {
			emit("localfs:homedir:error", err.Error())
			return
		}
		emit("localfs:homedir:result", home)
	})

	a.wailsApp.Event.On("localfs:listdir", func(e *application.CustomEvent) {
		payload, _ := json.Marshal(e.Data)
		var msg struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(payload, &msg); err != nil {
			emit("localfs:listdir:error", err.Error())
			return
		}
		result, err := a.listLocalDir(msg.Path)
		if err != nil {
			emit("localfs:listdir:error", err.Error())
			return
		}
		emit("localfs:listdir:result", result)
	})

	a.wailsApp.Event.On("sftp:upload", func(e *application.CustomEvent) {
		payload, _ := json.Marshal(e.Data)
		var msg struct {
			ConnectionID string `json:"connectionID"`
			LocalPath    string `json:"localPath"`
			RemotePath   string `json:"remotePath"`
		}
		if err := json.Unmarshal(payload, &msg); err != nil {
			emit("sftp:upload:error", err.Error())
			return
		}
		go func() {
			if err := a.sftpManager.UploadFile(msg.ConnectionID, msg.LocalPath, msg.RemotePath); err != nil {
				emit("sftp:upload:error", err.Error())
			}
		}()
	})

	a.wailsApp.Event.On("sftp:download", func(e *application.CustomEvent) {
		payload, _ := json.Marshal(e.Data)
		var msg struct {
			ConnectionID string `json:"connectionID"`
			RemotePath   string `json:"remotePath"`
			LocalPath    string `json:"localPath"`
		}
		if err := json.Unmarshal(payload, &msg); err != nil {
			emit("sftp:download:error", err.Error())
			return
		}
		go func() {
			if err := a.sftpManager.DownloadFile(msg.ConnectionID, msg.RemotePath, msg.LocalPath); err != nil {
				emit("sftp:download:error", err.Error())
			}
		}()
	})

	// Listen for terminal resize events from frontend
	a.wailsApp.Event.On("terminal:resize", func(e *application.CustomEvent) {
		payload, _ := json.Marshal(e.Data)
		var msg struct {
			SessionID string `json:"sessionID"`
			Rows      int    `json:"rows"`
			Cols      int    `json:"cols"`
		}
		if err := json.Unmarshal(payload, &msg); err != nil {
			return
		}
		if msg.SessionID != "" && msg.Rows > 0 && msg.Cols > 0 {
			a.sshManager.ResizeWindow(msg.SessionID, msg.Rows, msg.Cols)
		}
	})

	return nil
}

// ServiceShutdown is called by Wails when the application shuts down.
func (a *AppService) ServiceShutdown() error {
	if a.db != nil {
		a.db.Close()
	}
	return nil
}

// --- Connection CRUD ---

func (a *AppService) ListConnections() ([]models.Connection, error) {
	rows, err := a.db.Query("SELECT id, group_id, name, host, port, username, auth_type, proxy_type, proxy_addr, jump_host_id, upload_path, default_cmd, sort_order, color, last_used_at, created_at, updated_at FROM connections ORDER BY sort_order, name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	conns := make([]models.Connection, 0)
	for rows.Next() {
		var c models.Connection
		if err := rows.Scan(&c.ID, &c.GroupID, &c.Name, &c.Host, &c.Port, &c.Username, &c.AuthType, &c.ProxyType, &c.ProxyAddr, &c.JumpHostID, &c.UploadPath, &c.DefaultCmd, &c.SortOrder, &c.Color, &c.LastUsedAt, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		conns = append(conns, c)
	}
	return conns, nil
}

func (a *AppService) CreateConnection(form models.ConnectionForm) error {
	// Normalize empty strings to nil for nullable fields
	if form.GroupID != nil && *form.GroupID == "" {
		form.GroupID = nil
	}

	encryptedPW, err := a.db.Crypto().Encrypt(form.Password)
	if err != nil {
		return fmt.Errorf("encrypt password: %w", err)
	}
	encryptedKey, err := a.db.Crypto().Encrypt(form.PrivateKey)
	if err != nil {
		return fmt.Errorf("encrypt private key: %w", err)
	}
	encryptedPassphrase, err := a.db.Crypto().Encrypt(form.KeyPassphrase)
	if err != nil {
		return fmt.Errorf("encrypt passphrase: %w", err)
	}

	_, err = a.db.Exec(
		`INSERT INTO connections (id, group_id, name, host, port, username, auth_type, password, private_key, key_passphrase, proxy_type, proxy_addr, jump_host_id, upload_path, default_cmd, sort_order, color)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		form.ID, form.GroupID, form.Name, form.Host, form.Port, form.Username, form.AuthType,
		encryptedPW, encryptedKey, encryptedPassphrase,
		form.ProxyType, form.ProxyAddr, form.JumpHostID,
		form.UploadPath, form.DefaultCmd, form.SortOrder, form.Color,
	)
	return err
}

func (a *AppService) UpdateConnection(form models.ConnectionForm) error {
	// Normalize empty strings to nil for nullable fields
	if form.GroupID != nil && *form.GroupID == "" {
		form.GroupID = nil
	}

	// If sensitive fields are empty, keep existing values from DB
	var encryptedPW, encryptedKey, encryptedPassphrase string
	if form.Password != "" {
		var err error
		encryptedPW, err = a.db.Crypto().Encrypt(form.Password)
		if err != nil {
			return fmt.Errorf("encrypt password: %w", err)
		}
	} else {
		var val string
		a.db.QueryRow("SELECT password FROM connections WHERE id = ?", form.ID).Scan(&val)
		encryptedPW = val
	}
	if form.PrivateKey != "" {
		var err error
		encryptedKey, err = a.db.Crypto().Encrypt(form.PrivateKey)
		if err != nil {
			return fmt.Errorf("encrypt private key: %w", err)
		}
	} else {
		var val string
		a.db.QueryRow("SELECT private_key FROM connections WHERE id = ?", form.ID).Scan(&val)
		encryptedKey = val
	}
	if form.KeyPassphrase != "" {
		var err error
		encryptedPassphrase, err = a.db.Crypto().Encrypt(form.KeyPassphrase)
		if err != nil {
			return fmt.Errorf("encrypt passphrase: %w", err)
		}
	} else {
		var val string
		a.db.QueryRow("SELECT key_passphrase FROM connections WHERE id = ?", form.ID).Scan(&val)
		encryptedPassphrase = val
	}

	_, err := a.db.Exec(
		`UPDATE connections SET group_id=?, name=?, host=?, port=?, username=?, auth_type=?, password=?, private_key=?, key_passphrase=?, proxy_type=?, proxy_addr=?, jump_host_id=?, upload_path=?, default_cmd=?, sort_order=?, color=?, updated_at=CURRENT_TIMESTAMP
		 WHERE id=?`,
		form.GroupID, form.Name, form.Host, form.Port, form.Username, form.AuthType,
		encryptedPW, encryptedKey, encryptedPassphrase,
		form.ProxyType, form.ProxyAddr, form.JumpHostID,
		form.UploadPath, form.DefaultCmd, form.SortOrder, form.Color,
		form.ID,
	)
	return err
}

func (a *AppService) DeleteConnection(id string) error {
	a.sshManager.Disconnect(id)
	_, err := a.db.Exec("DELETE FROM connections WHERE id = ?", id)
	return err
}

// MoveConnection updates just the group_id of a connection (for drag-and-drop reordering).
func (a *AppService) MoveConnection(connectionID string, groupID *string) error {
	if groupID != nil && *groupID == "" {
		groupID = nil
	}
	_, err := a.db.Exec("UPDATE connections SET group_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", groupID, connectionID)
	return err
}

// --- SSH Session Management ---

func (a *AppService) ConnectSSH(connectionID string) (string, error) {
	conn, err := a.getConnectionByID(connectionID)
	if err != nil {
		return "", err
	}
	sessionID := uuid.New().String()
	_, err = a.sshManager.Connect(conn, sessionID)
	if err != nil {
		return "", err
	}
	return sessionID, nil
}

func (a *AppService) DisconnectSSH(connectionID string) {
	a.StopMonitor(connectionID)
	a.sftpManager.CloseClient(connectionID)
	a.sshManager.Disconnect(connectionID)
	a.fwdManager.StopAllForConnection(connectionID)
}

// DisconnectSession disconnects a single terminal session. If it's the last
// session for its connection, the full connection cleanup is also performed.
func (a *AppService) DisconnectSession(sessionID string, connectionID string) {
	a.sshManager.DisconnectSession(sessionID)
	if !a.sshManager.HasActiveSession(connectionID) {
		a.StopMonitor(connectionID)
		a.sftpManager.CloseClient(connectionID)
		a.fwdManager.StopAllForConnection(connectionID)
	}
}

// --- Monitor ---

func (a *AppService) StartMonitor(connectionID string) error {
	if _, ok := a.monitors[connectionID]; ok {
		return nil
	}
	mon := vshellssh.NewMonitor(a.sshManager, connectionID)
	mon.Start()
	a.monitors[connectionID] = mon
	return nil
}

func (a *AppService) StopMonitor(connectionID string) {
	if mon, ok := a.monitors[connectionID]; ok {
		mon.Stop()
		delete(a.monitors, connectionID)
	}
}

func (a *AppService) getConnectionByID(id string) (*models.Connection, error) {
	var c models.Connection
	err := a.db.QueryRow(
		"SELECT id, group_id, name, host, port, username, auth_type, password, private_key, key_passphrase, proxy_type, proxy_addr, jump_host_id, upload_path, default_cmd, sort_order, color, last_used_at FROM connections WHERE id = ?",
		id,
	).Scan(&c.ID, &c.GroupID, &c.Name, &c.Host, &c.Port, &c.Username, &c.AuthType, &c.Password, &c.PrivateKey, &c.KeyPassphrase, &c.ProxyType, &c.ProxyAddr, &c.JumpHostID, &c.UploadPath, &c.DefaultCmd, &c.SortOrder, &c.Color, &c.LastUsedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (a *AppService) GetPassword(id string) (string, error) {
	var encrypted string
	if err := a.db.QueryRow("SELECT password FROM connections WHERE id = ?", id).Scan(&encrypted); err != nil {
		return "", err
	}
	return a.db.Crypto().Decrypt(encrypted)
}

// --- Group CRUD ---

func (a *AppService) ListGroups() ([]models.Group, error) {
	rows, err := a.db.Query("SELECT id, name, parent_id, sort_order FROM groups ORDER BY sort_order, name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []models.Group
	for rows.Next() {
		var g models.Group
		if err := rows.Scan(&g.ID, &g.Name, &g.ParentID, &g.SortOrder); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, nil
}

func (a *AppService) CreateGroup(id, name string, parentID *string, sortOrder int) error {
	_, err := a.db.Exec("INSERT INTO groups (id, name, parent_id, sort_order) VALUES (?, ?, ?, ?)", id, name, parentID, sortOrder)
	return err
}

func (a *AppService) UpdateGroup(id, name string) error {
	_, err := a.db.Exec("UPDATE groups SET name = ? WHERE id = ?", name, id)
	return err
}

func (a *AppService) DeleteGroup(id string) error {
	var count int
	if err := a.db.QueryRow("SELECT COUNT(*) FROM connections WHERE group_id = ?", id).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("cannot delete group: %d connection(s) still belong to this group", count)
	}

	// Reassign child groups to this group's parent
	var parentID *string
	if err := a.db.QueryRow("SELECT parent_id FROM groups WHERE id = ?", id).Scan(&parentID); err != nil {
		return err
	}
	_, err := a.db.Exec("UPDATE groups SET parent_id = ? WHERE parent_id = ?", parentID, id)
	if err != nil {
		return err
	}

	_, err = a.db.Exec("DELETE FROM groups WHERE id = ?", id)
	return err
}

// --- Quick Commands ---

func (a *AppService) ListQuickCommands(connectionID *string) ([]models.QuickCommand, error) {
	var query string
	var args []any

	if connectionID != nil {
		query = "SELECT id, name, command, connection_id, sort_order FROM quick_commands WHERE connection_id = ? OR connection_id IS NULL ORDER BY sort_order"
		args = []any{*connectionID}
	} else {
		query = "SELECT id, name, command, connection_id, sort_order FROM quick_commands WHERE connection_id IS NULL ORDER BY sort_order"
	}

	rows, err := a.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cmds []models.QuickCommand
	for rows.Next() {
		var cmd models.QuickCommand
		if err := rows.Scan(&cmd.ID, &cmd.Name, &cmd.Command, &cmd.ConnectionID, &cmd.SortOrder); err != nil {
			return nil, err
		}
		cmds = append(cmds, cmd)
	}
	return cmds, nil
}

func (a *AppService) CreateQuickCommand(cmd models.QuickCommand) error {
	_, err := a.db.Exec("INSERT INTO quick_commands (id, name, command, connection_id, sort_order) VALUES (?, ?, ?, ?, ?)",
		cmd.ID, cmd.Name, cmd.Command, cmd.ConnectionID, cmd.SortOrder)
	return err
}

func (a *AppService) DeleteQuickCommand(id string) error {
	_, err := a.db.Exec("DELETE FROM quick_commands WHERE id = ?", id)
	return err
}

// --- Port Forwarding ---

func (a *AppService) ListPortForwards(connectionID string) ([]models.PortForward, error) {
	rows, err := a.db.Query("SELECT id, name, connection_id, type, local_host, local_port, remote_host, remote_port, auto_start FROM port_forwards WHERE connection_id = ?", connectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var forwards []models.PortForward
	for rows.Next() {
		var f models.PortForward
		var autoStart int
		if err := rows.Scan(&f.ID, &f.Name, &f.ConnectionID, &f.Type, &f.LocalHost, &f.LocalPort, &f.RemoteHost, &f.RemotePort, &autoStart); err != nil {
			return nil, err
		}
		f.AutoStart = autoStart == 1
		forwards = append(forwards, f)
	}
	return forwards, nil
}

func (a *AppService) ListAllPortForwards() ([]models.PortForward, error) {
	rows, err := a.db.Query(`
		SELECT pf.id, pf.name, pf.connection_id, COALESCE(c.name, ''), COALESCE(c.host, ''),
		       pf.type, pf.local_host, pf.local_port, pf.remote_host, pf.remote_port, pf.auto_start
		FROM port_forwards pf
		LEFT JOIN connections c ON c.id = pf.connection_id
		ORDER BY c.name, c.host, pf.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var forwards []models.PortForward
	for rows.Next() {
		var f models.PortForward
		var autoStart int
		if err := rows.Scan(&f.ID, &f.Name, &f.ConnectionID, &f.ConnectionName, &f.ConnectionHost,
			&f.Type, &f.LocalHost, &f.LocalPort, &f.RemoteHost, &f.RemotePort, &autoStart); err != nil {
			return nil, err
		}
		f.AutoStart = autoStart == 1
		forwards = append(forwards, f)
	}
	return forwards, nil
}

func (a *AppService) CreatePortForward(name, connectionID, fwdType string, localHost string, localPort int, remoteHost string, remotePort int, autoStart bool) (models.PortForward, error) {
	fwd := models.PortForward{
		ID:           uuid.New().String(),
		Name:         name,
		ConnectionID: connectionID,
		Type:         models.ForwardType(fwdType),
		LocalHost:    localHost,
		LocalPort:    localPort,
		RemoteHost:   remoteHost,
		RemotePort:   remotePort,
		AutoStart:    autoStart,
	}
	autoStartInt := 0
	if autoStart {
		autoStartInt = 1
	}
	_, err := a.db.Exec(
		"INSERT INTO port_forwards (id, name, connection_id, type, local_host, local_port, remote_host, remote_port, auto_start) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		fwd.ID, fwd.Name, fwd.ConnectionID, string(fwd.Type), fwd.LocalHost, fwd.LocalPort, fwd.RemoteHost, fwd.RemotePort, autoStartInt,
	)
	if err != nil {
		return models.PortForward{}, fmt.Errorf("create port forward: %w", err)
	}
	return fwd, nil
}

func (a *AppService) DeletePortForward(id string) error {
	a.fwdManager.Stop(id)
	_, err := a.db.Exec("DELETE FROM port_forwards WHERE id = ?", id)
	return err
}

func (a *AppService) UpdatePortForward(id, name, connectionID, fwdType string, localHost string, localPort int, remoteHost string, remotePort int, autoStart bool) error {
	autoStartInt := 0
	if autoStart {
		autoStartInt = 1
	}
	_, err := a.db.Exec(
		"UPDATE port_forwards SET name = ?, connection_id = ?, type = ?, local_host = ?, local_port = ?, remote_host = ?, remote_port = ?, auto_start = ? WHERE id = ?",
		name, connectionID, fwdType, localHost, localPort, remoteHost, remotePort, autoStartInt, id,
	)
	return err
}

func (a *AppService) ListRunningPortForwards() []string {
	return a.fwdManager.RunningIDs()
}

func (a *AppService) StartPortForward(id string) error {
	rows, err := a.db.Query("SELECT id, name, connection_id, type, local_host, local_port, remote_host, remote_port, auto_start FROM port_forwards WHERE id = ?", id)
	if err != nil {
		return err
	}
	if !rows.Next() {
		rows.Close()
		return fmt.Errorf("port forward %s not found", id)
	}
	var f models.PortForward
	var autoStart int
	if err := rows.Scan(&f.ID, &f.Name, &f.ConnectionID, &f.Type, &f.LocalHost, &f.LocalPort, &f.RemoteHost, &f.RemotePort, &autoStart); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	client, err := a.sshManager.GetSSHClient(f.ConnectionID)
	if err != nil {
		// No active session — ensure connection for port forwarding
		conn, dbErr := a.getConnectionByID(f.ConnectionID)
		if dbErr != nil {
			return fmt.Errorf("connection %s not found", f.ConnectionID)
		}
		if ensureErr := a.sshManager.EnsureClient(conn); ensureErr != nil {
			return fmt.Errorf("connect to %s: %w", conn.Host, ensureErr)
		}
		client, err = a.sshManager.GetSSHClient(f.ConnectionID)
		if err != nil {
			return err
		}
	}

	switch f.Type {
	case models.ForwardLocal:
		return a.fwdManager.StartLocal(client, &f)
	default:
		return fmt.Errorf("forward type %s not yet supported", f.Type)
	}
}

func (a *AppService) StopPortForward(id string) error {
	// Resolve connectionID before stopping, so we can check cleanup
	var connectionID string
	if f, ok := a.fwdManager.Forward(id); ok {
		connectionID = f.ConnectionID
	}

	if err := a.fwdManager.Stop(id); err != nil {
		return err
	}

	// If no more active forwards and no terminal sessions, disconnect
	if connectionID != "" && a.fwdManager.ActiveCount(connectionID) == 0 {
		a.sshManager.RemoveClient(connectionID)
	}
	return nil
}

// --- SFTP ---

func (a *AppService) SFTPReadDir(connectionID string, path string) ([]sftp.FileInfo, error) {
	return a.sftpManager.ReadDir(connectionID, path)
}

func (a *AppService) SFTPUpload(connectionID, localPath, remotePath string) error {
	go func() {
		if err := a.sftpManager.UploadFile(connectionID, localPath, remotePath); err != nil {
			a.wailsApp.Event.Emit("sftp:upload:error", err.Error())
		}
		a.wailsApp.Event.Emit("sftp:transfer-done", map[string]string{"direction": "upload", "connectionID": connectionID})
	}()
	return nil
}

func (a *AppService) SFTPDownload(connectionID, remotePath, localPath string) error {
	go func() {
		if err := a.sftpManager.DownloadFile(connectionID, remotePath, localPath); err != nil {
			a.wailsApp.Event.Emit("sftp:download:error", err.Error())
		}
		a.wailsApp.Event.Emit("sftp:transfer-done", map[string]string{"direction": "download"})
	}()
	return nil
}

func (a *AppService) SFTPReadFileContent(connectionID, remotePath string) (string, error) {
	return a.sftpManager.ReadFileContent(connectionID, remotePath)
}

func (a *AppService) SFTPWriteFileContent(connectionID, remotePath, content string) error {
	return a.sftpManager.WriteFileContent(connectionID, remotePath, content)
}

func (a *AppService) SFTPDelete(connectionID, remotePath string) error {
	go func() {
		if err := a.sftpManager.RemoveFile(connectionID, remotePath); err != nil {
			a.wailsApp.Event.Emit("sftp:delete:error", err.Error())
		}
		a.wailsApp.Event.Emit("sftp:transfer-done", map[string]string{"direction": "delete", "connectionID": connectionID})
	}()
	return nil
}

func (a *AppService) SFTPCancelTransfers() {
	a.sftpManager.CancelAllTransfers()
}

// --- Local File System ---

type LocalFileInfo struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Mode    uint32 `json:"mode"`
	ModTime int64  `json:"mod_time"`
	IsDir   bool   `json:"is_dir"`
}

func (a *AppService) GetHomeDir() (string, error) {
	return os.UserHomeDir()
}

func (a *AppService) OpenInFileManager(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("explorer", path)
	default: // linux
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}

func (a *AppService) ListLocalDir(dirPath string) ([]LocalFileInfo, error) {
	return a.listLocalDir(dirPath)
}

func (a *AppService) DeleteLocalFile(localPath string) error {
	return os.RemoveAll(localPath)
}

const maxLocalEditSize = 5 * 1024 * 1024 // 5MB

func (a *AppService) ReadLocalFileContent(localPath string) (string, error) {
	stat, err := os.Stat(localPath)
	if err != nil {
		return "", err
	}
	if stat.Size() > maxLocalEditSize {
		return "", fmt.Errorf("file too large to edit (%d bytes, max %d)", stat.Size(), maxLocalEditSize)
	}
	data, err := os.ReadFile(localPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (a *AppService) WriteLocalFileContent(localPath, content string) error {
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(localPath, []byte(content), 0644)
}

func (a *AppService) listLocalDir(dirPath string) ([]LocalFileInfo, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	result := make([]LocalFileInfo, 0, len(entries))
	var dirs, files []LocalFileInfo

	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		fi := LocalFileInfo{
			Name:    e.Name(),
			Path:    filepath.Join(dirPath, e.Name()),
			Size:    info.Size(),
			Mode:    uint32(info.Mode()),
			ModTime: info.ModTime().Unix(),
			IsDir:   e.IsDir(),
		}
		if e.IsDir() {
			dirs = append(dirs, fi)
		} else {
			files = append(files, fi)
		}
	}

	result = append(dirs, files...)
	return result, nil
}

// --- SSH Keys (direct ~/.ssh management) ---

type SSHKeyInfo struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	Fingerprint   string `json:"fingerprint"`
	PublicKey     string `json:"public_key"`
	Comment       string `json:"comment"`
	HasPassphrase bool   `json:"has_passphrase"`
}

func sshDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".ssh"), nil
}

// parsePubComment extracts the comment from a public key line like "ssh-ed25519 AAAA... user@host"
func parsePubComment(pubLine string) string {
	parts := strings.SplitN(strings.TrimSpace(pubLine), " ", 3)
	if len(parts) == 3 {
		return parts[2]
	}
	return ""
}

func parseKeyFile(keyPath string) (SSHKeyInfo, error) {
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return SSHKeyInfo{}, err
	}

	info := SSHKeyInfo{Name: filepath.Base(keyPath)}

	// Read .pub companion for public key, type, and fingerprint
	if pubData, err := os.ReadFile(keyPath + ".pub"); err == nil {
		pubStr := strings.TrimSpace(string(pubData))
		info.PublicKey = pubStr
		info.Comment = parsePubComment(pubStr)
		if pubKey, _, _, _, err := ssh.ParseAuthorizedKey(pubData); err == nil {
			info.Type = pubKey.Type()
			info.Fingerprint = ssh.FingerprintSHA256(pubKey)
		}
	}

	content := strings.TrimSpace(string(data))
	signer, err := ssh.ParsePrivateKey(data)
	if err != nil {
		if strings.Contains(err.Error(), "password") || strings.Contains(err.Error(), "passphrase") || strings.Contains(err.Error(), "decrypt") {
			info.HasPassphrase = true
			if info.Type == "" {
				info.Type = guessKeyTypeFromContent(content)
			}
			return info, nil
		}
		return SSHKeyInfo{}, err
	}

	// Prefer private key for type/fingerprint when available
	pubKey := signer.PublicKey()
	info.Type = pubKey.Type()
	info.Fingerprint = ssh.FingerprintSHA256(pubKey)
	return info, nil
}

func guessKeyTypeFromContent(content string) string {
	if strings.Contains(content, "BEGIN RSA PRIVATE KEY") {
		return "ssh-rsa"
	}
	if strings.Contains(content, "BEGIN EC PRIVATE KEY") {
		return "ecdsa-sha2-nistp256"
	}
	if strings.Contains(content, "BEGIN DSA PRIVATE KEY") {
		return "ssh-dss"
	}
	return "ssh-ed25519"
}

// ListSSHKeys scans ~/.ssh and its subdirectories for private key files and returns their metadata.
func (a *AppService) ListSSHKeys() ([]SSHKeyInfo, error) {
	dir, err := sshDir()
	if err != nil {
		return nil, err
	}

	skipSuffixes := []string{".pub", "-cert.pub", ".known_hosts", ".known_hosts_old", ".authorized_keys", ".config", ".rfc"}

	var results []SSHKeyInfo
	var walk func(string)
	walk = func(current string) {
		entries, err := os.ReadDir(current)
		if err != nil {
			return
		}
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}
			path := filepath.Join(current, name)
			if e.IsDir() {
				walk(path)
				continue
			}
			skip := false
			for _, s := range skipSuffixes {
				if strings.HasSuffix(name, s) {
					skip = true
					break
				}
			}
			if skip {
				continue
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			if !strings.Contains(string(raw), "PRIVATE KEY") {
				continue
			}
			info, err := parseKeyFile(path)
			if err != nil {
				continue
			}
			// Use relative path from ~/.ssh as the name for keys in subdirectories
			rel, err := filepath.Rel(dir, path)
			if err == nil {
				info.Name = rel
			}
			results = append(results, info)
		}
	}
	walk(dir)

	return results, nil
}

// SaveSSHKey writes a private key (and optional .pub) to ~/.ssh/<name>.
func (a *AppService) SaveSSHKey(name string, privateKey string, publicKey string) error {
	dir, err := sshDir()
	if err != nil {
		return err
	}

	keyPath := filepath.Join(dir, name)
	os.MkdirAll(filepath.Dir(keyPath), 0700)

	if err := os.WriteFile(keyPath, []byte(privateKey), 0600); err != nil {
		return fmt.Errorf("write key: %w", err)
	}

	if publicKey != "" {
		pubPath := keyPath + ".pub"
		if err := os.WriteFile(pubPath, []byte(strings.TrimSpace(publicKey)+"\n"), 0644); err != nil {
			return fmt.Errorf("write pub: %w", err)
		}
	}

	return nil
}

// RenameSSHKey renames a key file and its .pub companion in ~/.ssh.
func (a *AppService) RenameSSHKey(oldName string, newName string) error {
	dir, err := sshDir()
	if err != nil {
		return err
	}

	oldPath := filepath.Join(dir, oldName)
	newPath := filepath.Join(dir, newName)
	os.MkdirAll(filepath.Dir(newPath), 0700)

	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("rename key: %w", err)
	}

	oldPub := oldPath + ".pub"
	if _, err := os.Stat(oldPub); err == nil {
		newPub := newPath + ".pub"
		if err := os.Rename(oldPub, newPub); err != nil {
			return fmt.Errorf("rename pub: %w", err)
		}
	}

	return nil
}

// DeleteSSHKey removes a key file and its .pub companion from ~/.ssh.
// If the key is in a subdirectory, the subdirectory is also removed when empty.
func (a *AppService) DeleteSSHKey(name string) error {
	dir, err := sshDir()
	if err != nil {
		return err
	}

	keyPath := filepath.Join(dir, name)
	if err := os.Remove(keyPath); err != nil {
		return fmt.Errorf("delete key: %w", err)
	}

	os.Remove(keyPath + ".pub")
	// Remove parent subdirectory if empty (not ~/.ssh itself)
	if parent := filepath.Dir(keyPath); parent != dir {
		os.Remove(parent)
	}
	return nil
}

// ReadSSHKeyContent returns the public key or private key content.
// kind = "pub" returns the .pub file, kind = "private" returns the private key.
func (a *AppService) ReadSSHKeyContent(name string, kind string) (string, error) {
	dir, err := sshDir()
	if err != nil {
		return "", err
	}

	var path string
	if kind == "pub" {
		path = filepath.Join(dir, name+".pub")
	} else {
		path = filepath.Join(dir, name)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read key: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

// GenerateSSHKey generates a new SSH key pair and writes it to ~/.ssh/<name>.
func (a *AppService) GenerateSSHKey(name string, keyType string, bits int, comment string, passphrase string) error {
	dir, err := sshDir()
	if err != nil {
		return err
	}
	os.MkdirAll(dir, 0700)

	var pubKey ssh.PublicKey
	var rawPriv any

	switch keyType {
	case "ed25519":
		pub, priv, err := ed25519.GenerateKey(nil)
		if err != nil {
			return fmt.Errorf("generate ed25519: %w", err)
		}
		rawPriv = priv
		pubKey, err = ssh.NewPublicKey(pub)
		if err != nil {
			return fmt.Errorf("create public key: %w", err)
		}

	case "rsa":
		if bits < 2048 {
			bits = 4096
		}
		priv, err := rsa.GenerateKey(rand.Reader, bits)
		if err != nil {
			return fmt.Errorf("generate rsa: %w", err)
		}
		rawPriv = priv
		pubKey, err = ssh.NewPublicKey(&priv.PublicKey)
		if err != nil {
			return fmt.Errorf("create public key: %w", err)
		}

	case "ecdsa":
		var curve elliptic.Curve
		switch bits {
		case 384:
			curve = elliptic.P384()
		case 521:
			curve = elliptic.P521()
		default:
			curve = elliptic.P256()
		}
		priv, err := ecdsa.GenerateKey(curve, rand.Reader)
		if err != nil {
			return fmt.Errorf("generate ecdsa: %w", err)
		}
		rawPriv = priv
		pubKey, err = ssh.NewPublicKey(&priv.PublicKey)
		if err != nil {
			return fmt.Errorf("create public key: %w", err)
		}

	default:
		return fmt.Errorf("unsupported key type: %s", keyType)
	}

	var block *pem.Block
	if passphrase != "" {
		block, err = ssh.MarshalPrivateKeyWithPassphrase(rawPriv, comment, []byte(passphrase))
	} else {
		block, err = ssh.MarshalPrivateKey(rawPriv, comment)
	}
	if err != nil {
		return fmt.Errorf("marshal private key: %w", err)
	}
	privateKeyPEM := pem.EncodeToMemory(block)

	// Write private key
	keyPath := filepath.Join(dir, name)
	os.MkdirAll(filepath.Dir(keyPath), 0700)
	if err := os.WriteFile(keyPath, privateKeyPEM, 0600); err != nil {
		return fmt.Errorf("write private key: %w", err)
	}

	// Write .pub
	pubLine := string(ssh.MarshalAuthorizedKey(pubKey))
	pubLine = strings.TrimSpace(pubLine)
	if comment != "" {
		pubLine += " " + comment
	}
	pubPath := keyPath + ".pub"
	if err := os.WriteFile(pubPath, []byte(pubLine+"\n"), 0644); err != nil {
		return fmt.Errorf("write public key: %w", err)
	}

	return nil
}
