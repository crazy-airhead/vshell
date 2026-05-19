package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"

	"vshell/internal/db"
	"vshell/internal/models"
	"vshell/internal/portforward"
	"vshell/internal/sftp"
	"vshell/internal/ssh"
)

type AppService struct {
	wailsApp    *application.App
	db          *db.DB
	sshManager  *ssh.Manager
	sftpManager *sftp.Manager
	fwdManager  *portforward.Manager
	monitors    map[string]*ssh.Monitor
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

	a.sshManager = ssh.NewManager(a.db.Crypto(), emit)
	a.sftpManager = sftp.NewManager(a.sshManager, a.db.Crypto(), emit)
	a.fwdManager = portforward.NewManager()
	a.monitors = make(map[string]*ssh.Monitor)

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

// --- SSH Session Management ---

func (a *AppService) ConnectSSH(connectionID string) error {
	conn, err := a.getConnectionByID(connectionID)
	if err != nil {
		return err
	}
	_, err = a.sshManager.Connect(conn)
	return err
}

func (a *AppService) DisconnectSSH(connectionID string) {
	a.StopMonitor(connectionID)
	a.sftpManager.CloseClient(connectionID)
	a.sshManager.Disconnect(connectionID)
	a.fwdManager.StopAllForConnection(connectionID)
}

// --- Monitor ---

func (a *AppService) StartMonitor(connectionID string) error {
	if _, ok := a.monitors[connectionID]; ok {
		return nil
	}
	mon := ssh.NewMonitor(a.sshManager, connectionID)
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

func (a *AppService) DeleteGroup(id string) error {
	_, err := a.db.Exec("DELETE FROM groups WHERE id = ?", id)
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
	rows, err := a.db.Query("SELECT id, connection_id, type, local_host, local_port, remote_host, remote_port, auto_start FROM port_forwards WHERE connection_id = ?", connectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var forwards []models.PortForward
	for rows.Next() {
		var f models.PortForward
		var autoStart int
		if err := rows.Scan(&f.ID, &f.ConnectionID, &f.Type, &f.LocalHost, &f.LocalPort, &f.RemoteHost, &f.RemotePort, &autoStart); err != nil {
			return nil, err
		}
		f.AutoStart = autoStart == 1
		forwards = append(forwards, f)
	}
	return forwards, nil
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

func (a *AppService) SFTPDelete(connectionID, remotePath string) error {
	go func() {
		if err := a.sftpManager.RemoveFile(connectionID, remotePath); err != nil {
			a.wailsApp.Event.Emit("sftp:delete:error", err.Error())
		}
		a.wailsApp.Event.Emit("sftp:transfer-done", map[string]string{"direction": "delete", "connectionID": connectionID})
	}()
	return nil
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

func (a *AppService) ListLocalDir(dirPath string) ([]LocalFileInfo, error) {
	return a.listLocalDir(dirPath)
}

func (a *AppService) DeleteLocalFile(localPath string) error {
	return os.RemoveAll(localPath)
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
