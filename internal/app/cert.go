package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"vshell/internal/cert"
	"vshell/internal/models"
)

// ---- helpers ----

// ensureCertConnection returns an active SSH client's connection, dialing it
// on demand when no terminal session is open (same pattern as port
// forwarding).
func (a *AppService) ensureCertConnection(connectionID string) (*models.Connection, error) {
	if _, err := a.sshManager.GetSSHClient(connectionID); err == nil {
		return a.getConnectionByID(connectionID)
	}
	conn, err := a.getConnectionByID(connectionID)
	if err != nil {
		return nil, fmt.Errorf("connection %s not found", connectionID)
	}
	if err := a.sshManager.EnsureClient(conn); err != nil {
		return nil, fmt.Errorf("connect to %s: %w", conn.Host, err)
	}
	return conn, nil
}

const certTaskColumns = `ct.id, ct.connection_id, COALESCE(c.name, ''), COALESCE(c.host, ''),
	ct.name, ct.primary_domain, ct.san_domains, ct.dns_provider, COALESCE(ct.dns_plugin, ''),
	ct.dns_credentials, ct.key_length, ct.dns_sleep, ct.test_mode, ct.auto_install,
	COALESCE(ct.cert_dir, ''), COALESCE(ct.key_file, ''), COALESCE(ct.fullchain_file, ''), COALESCE(ct.reload_cmd, ''),
	ct.last_status, COALESCE(ct.last_error, ''), ct.last_run_at, ct.created_at, ct.updated_at`

func scanCertTask(scan func(dest ...any) error) (models.CertTask, error) {
	var t models.CertTask
	var sanDomains string
	var testMode, autoInstall int
	if err := scan(&t.ID, &t.ConnectionID, &t.ConnectionName, &t.ConnectionHost,
		&t.Name, &t.PrimaryDomain, &sanDomains, &t.DNSProvider, &t.DNSPlugin,
		&t.DNSCredentials, &t.KeyLength, &t.DNSSleep, &testMode, &autoInstall,
		&t.CertDir, &t.KeyFile, &t.FullchainFile, &t.ReloadCmd,
		&t.LastStatus, &t.LastError, &t.LastRunAt, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return models.CertTask{}, err
	}
	t.SANDomains = splitNonEmpty(sanDomains, ",")
	t.TestMode = testMode == 1
	t.AutoInstall = autoInstall == 1
	return t, nil
}

func splitNonEmpty(s, sep string) []string {
	var out []string
	for _, part := range strings.Split(s, sep) {
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func (a *AppService) getCertTaskByID(id string) (*models.CertTask, error) {
	row := a.db.QueryRow(`
		SELECT `+certTaskColumns+`
		FROM cert_tasks ct
		LEFT JOIN connections c ON c.id = ct.connection_id
		WHERE ct.id = ?`, id)
	t, err := scanCertTask(row.Scan)
	if err != nil {
		return nil, fmt.Errorf("cert task %s not found", id)
	}
	return &t, nil
}

func (a *AppService) emitCertTaskUpdated(taskID string) {
	task, err := a.getCertTaskByID(taskID)
	if err != nil {
		return
	}
	a.wailsApp.Event.Emit("cert:task-updated", task)
}

func (a *AppService) setCertTaskStatus(id string, status models.CertStatus, errMsg string) error {
	_, err := a.db.Exec(
		"UPDATE cert_tasks SET last_status = ?, last_error = ?, last_run_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		string(status), errMsg, id)
	return err
}

func encryptCertCredentials(creds map[string]string, encrypt func(string) (string, error)) (string, error) {
	if len(creds) == 0 {
		return "", nil
	}
	data, err := json.Marshal(creds)
	if err != nil {
		return "", fmt.Errorf("marshal credentials: %w", err)
	}
	return encrypt(string(data))
}

func decryptCertCredentials(encoded string, decrypt func(string) (string, error)) (map[string]string, error) {
	if encoded == "" {
		return map[string]string{}, nil
	}
	plain, err := decrypt(encoded)
	if err != nil {
		return nil, fmt.Errorf("decrypt credentials: %w", err)
	}
	var creds map[string]string
	if err := json.Unmarshal([]byte(plain), &creds); err != nil {
		return nil, fmt.Errorf("unmarshal credentials: %w", err)
	}
	return creds, nil
}

// ---- sync bindings: local task CRUD ----

// ListCertTasks returns all local certificate tasks with connection info.
func (a *AppService) ListCertTasks() ([]models.CertTask, error) {
	rows, err := a.db.Query(`
		SELECT ` + certTaskColumns + `
		FROM cert_tasks ct
		LEFT JOIN connections c ON c.id = ct.connection_id
		ORDER BY c.name, c.host, ct.primary_domain`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []models.CertTask
	for rows.Next() {
		t, err := scanCertTask(rows.Scan)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// CreateCertTask stores a new certificate task. DNS credentials are
// encrypted before storage.
func (a *AppService) CreateCertTask(form models.CertTaskForm) (models.CertTask, error) {
	if form.ConnectionID == "" {
		return models.CertTask{}, fmt.Errorf("connection is required")
	}
	if _, err := a.getConnectionByID(form.ConnectionID); err != nil {
		return models.CertTask{}, fmt.Errorf("connection %s not found", form.ConnectionID)
	}
	if form.PrimaryDomain == "" {
		return models.CertTask{}, fmt.Errorf("primary domain is required")
	}
	if _, ok := cert.GetProvider(form.DNSProvider); !ok {
		return models.CertTask{}, fmt.Errorf("unknown DNS provider %q", form.DNSProvider)
	}
	if _, err := cert.PluginFor(form.DNSProvider, form.DNSPlugin); err != nil {
		return models.CertTask{}, err
	}
	if err := cert.ValidateCredentials(form.DNSProvider, form.DNSCredentials); err != nil {
		return models.CertTask{}, err
	}
	if form.KeyLength == "" {
		form.KeyLength = "ec-256"
	}
	if form.DNSSleep <= 0 {
		form.DNSSleep = 120
	}
	if form.Name == "" {
		form.Name = form.PrimaryDomain
	}
	encCreds, err := encryptCertCredentials(form.DNSCredentials, a.db.Crypto().Encrypt)
	if err != nil {
		return models.CertTask{}, err
	}

	taskID := uuid.New().String()
	testMode, autoInstall := 0, 0
	if form.TestMode {
		testMode = 1
	}
	if form.AutoInstall {
		autoInstall = 1
	}
	_, err = a.db.Exec(`
		INSERT INTO cert_tasks (id, connection_id, name, primary_domain, san_domains, dns_provider, dns_plugin,
			dns_credentials, key_length, dns_sleep, test_mode, auto_install,
			cert_dir, key_file, fullchain_file, reload_cmd, last_status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		taskID, form.ConnectionID, form.Name, form.PrimaryDomain, strings.Join(form.SANDomains, ","),
		form.DNSProvider, nullIfEmpty(form.DNSPlugin), encCreds, form.KeyLength, form.DNSSleep, testMode, autoInstall,
		nullIfEmpty(form.CertDir), nullIfEmpty(form.KeyFile), nullIfEmpty(form.FullchainFile), nullIfEmpty(form.ReloadCmd),
		string(models.CertStatusIdle))
	if err != nil {
		return models.CertTask{}, fmt.Errorf("create cert task: %w", err)
	}
	created, err := a.getCertTaskByID(taskID)
	if err != nil {
		return models.CertTask{}, err
	}
	return *created, nil
}

// UpdateCertTask updates an existing task. Empty DNS credentials keep the
// stored ones (same convention as connection secrets).
func (a *AppService) UpdateCertTask(form models.CertTaskForm) (models.CertTask, error) {
	existing, err := a.getCertTaskByID(form.ID)
	if err != nil {
		return models.CertTask{}, err
	}
	if form.PrimaryDomain == "" {
		return models.CertTask{}, fmt.Errorf("primary domain is required")
	}
	if _, ok := cert.GetProvider(form.DNSProvider); !ok {
		return models.CertTask{}, fmt.Errorf("unknown DNS provider %q", form.DNSProvider)
	}
	if _, err := cert.PluginFor(form.DNSProvider, form.DNSPlugin); err != nil {
		return models.CertTask{}, err
	}

	encCreds := existing.DNSCredentials
	if len(form.DNSCredentials) > 0 {
		if err := cert.ValidateCredentials(form.DNSProvider, form.DNSCredentials); err != nil {
			return models.CertTask{}, err
		}
		encCreds, err = encryptCertCredentials(form.DNSCredentials, a.db.Crypto().Encrypt)
		if err != nil {
			return models.CertTask{}, err
		}
	}

	keyLength := form.KeyLength
	if keyLength == "" {
		keyLength = existing.KeyLength
	}
	dnsSleep := form.DNSSleep
	if dnsSleep <= 0 {
		dnsSleep = existing.DNSSleep
	}
	name := form.Name
	if name == "" {
		name = form.PrimaryDomain
	}
	testMode, autoInstall := 0, 0
	if form.TestMode {
		testMode = 1
	}
	if form.AutoInstall {
		autoInstall = 1
	}

	_, err = a.db.Exec(`
		UPDATE cert_tasks SET name = ?, primary_domain = ?, san_domains = ?, dns_provider = ?, dns_plugin = ?,
			dns_credentials = ?, key_length = ?, dns_sleep = ?, test_mode = ?, auto_install = ?,
			cert_dir = ?, key_file = ?, fullchain_file = ?, reload_cmd = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		name, form.PrimaryDomain, strings.Join(form.SANDomains, ","), form.DNSProvider, nullIfEmpty(form.DNSPlugin),
		encCreds, keyLength, dnsSleep, testMode, autoInstall,
		nullIfEmpty(form.CertDir), nullIfEmpty(form.KeyFile), nullIfEmpty(form.FullchainFile), nullIfEmpty(form.ReloadCmd),
		form.ID)
	if err != nil {
		return models.CertTask{}, fmt.Errorf("update cert task: %w", err)
	}
	updated, err := a.getCertTaskByID(form.ID)
	if err != nil {
		return models.CertTask{}, err
	}
	a.emitCertTaskUpdated(form.ID)
	return *updated, nil
}

// DeleteCertTask removes the local task only; the server keeps its acme.sh
// config and cron renewals.
func (a *AppService) DeleteCertTask(id string) error {
	if a.certManager.IsRunning(id) {
		a.certManager.Cancel(id)
	}
	if _, err := a.db.Exec("DELETE FROM cert_tasks WHERE id = ?", id); err != nil {
		return err
	}
	a.wailsApp.Event.Emit("cert:task-deleted", map[string]any{"taskID": id})
	return nil
}

// GetCertTaskCredentials decrypts and returns the stored DNS credentials for
// the edit form's reveal button (same pattern as GetPassword).
func (a *AppService) GetCertTaskCredentials(id string) (map[string]string, error) {
	task, err := a.getCertTaskByID(id)
	if err != nil {
		return nil, err
	}
	return decryptCertCredentials(task.DNSCredentials, a.db.Crypto().Decrypt)
}

// ListDNSProviders returns the DNS provider registry for dynamic form
// rendering.
func (a *AppService) ListDNSProviders() ([]models.DNSProvider, error) {
	return cert.Providers, nil
}

// ---- sync bindings: remote inspection ----

// DetectCertEnvironment checks acme.sh installation, cron entry and curl on
// the target server.
func (a *AppService) DetectCertEnvironment(connectionID string) (models.CertEnvironment, error) {
	if _, err := a.ensureCertConnection(connectionID); err != nil {
		return models.CertEnvironment{}, err
	}
	return a.certManager.Detect(context.Background(), connectionID)
}

// ListRemoteCerts lists the certs managed by acme.sh on the server,
// enriched with renewal dates.
func (a *AppService) ListRemoteCerts(connectionID string) ([]models.RemoteCert, error) {
	if _, err := a.ensureCertConnection(connectionID); err != nil {
		return nil, err
	}
	return a.certManager.ListCerts(context.Background(), connectionID)
}

// GetRemoteCertInfo returns detailed key=value info for one remote cert.
func (a *AppService) GetRemoteCertInfo(connectionID, domain string, ecc bool) (*models.RemoteCertInfo, error) {
	if _, err := a.ensureCertConnection(connectionID); err != nil {
		return nil, err
	}
	return a.certManager.CertInfo(context.Background(), connectionID, domain, ecc)
}

// GetCertServerLog tails the persistent acme.sh log on the server.
func (a *AppService) GetCertServerLog(connectionID string) (string, error) {
	if _, err := a.ensureCertConnection(connectionID); err != nil {
		return "", err
	}
	return a.certManager.ReadServerLog(context.Background(), connectionID)
}

// CancelCertOp aborts a running issue/renew/remove/install operation.
func (a *AppService) CancelCertOp(id string) error {
	a.certManager.Cancel(id)
	return nil
}

// ---- async bindings: long-running operations ----

// StartAcmeShInstall installs acme.sh on the server; progress arrives via
// cert:log / cert:stage / cert:op-done events.
func (a *AppService) StartAcmeShInstall(connectionID, email string) (string, error) {
	if email == "" {
		return "", fmt.Errorf("account email is required to install acme.sh")
	}
	conn, err := a.ensureCertConnection(connectionID)
	if err != nil {
		return "", err
	}
	opID := uuid.New().String()
	if a.certManager.IsRunning(opID) {
		return "", fmt.Errorf("acme.sh install already running")
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.certManager.Register(opID, cancel)
	go func() {
		defer a.certManager.Unregister(opID)
		err := a.certManager.InstallAcmeSh(ctx, conn, opID, email)
		a.wailsApp.Event.Emit("cert:op-done", map[string]any{
			"opID": opID, "connectionID": connectionID, "error": errString(err),
		})
	}()
	return opID, nil
}

// StartCertIssue runs the full issuance flow for a stored task. email is
// only used when acme.sh has to be installed first.
func (a *AppService) StartCertIssue(taskID, email string) (string, error) {
	return a.startCertOperation(taskID, "issue", email)
}

// StartCertRenew forces an immediate renewal of a stored task.
func (a *AppService) StartCertRenew(taskID string) (string, error) {
	return a.startCertOperation(taskID, "renew", "")
}

func (a *AppService) startCertOperation(taskID, op, email string) (string, error) {
	if a.certManager.IsRunning(taskID) {
		return "", fmt.Errorf("cert task %s is already running", taskID)
	}
	task, err := a.getCertTaskByID(taskID)
	if err != nil {
		return "", err
	}
	conn, err := a.ensureCertConnection(task.ConnectionID)
	if err != nil {
		return "", err
	}
	creds, err := decryptCertCredentials(task.DNSCredentials, a.db.Crypto().Decrypt)
	if err != nil {
		return "", err
	}
	if err := a.setCertTaskStatus(taskID, models.CertStatusRunning, ""); err != nil {
		return "", err
	}
	a.emitCertTaskUpdated(taskID)

	ctx, cancel := context.WithCancel(context.Background())
	a.certManager.Register(taskID, cancel)
	go func() {
		defer a.certManager.Unregister(taskID)
		var runErr error
		switch op {
		case "issue":
			runErr = a.certManager.Issue(ctx, conn, task, creds, email)
		case "renew":
			runErr = a.certManager.Renew(ctx, conn, task)
		}
		status := models.CertStatusIssued
		errMsg := ""
		if runErr != nil {
			status = models.CertStatusFailed
			errMsg = runErr.Error()
		}
		if err := a.setCertTaskStatus(taskID, status, errMsg); err != nil {
			return
		}
		a.emitCertTaskUpdated(taskID)
	}()
	return taskID, nil
}

// StartCertRemove removes the cert from acme.sh on the server and,
// optionally, deletes the local task as well.
func (a *AppService) StartCertRemove(taskID string, deleteTask bool) (string, error) {
	if a.certManager.IsRunning(taskID) {
		return "", fmt.Errorf("cert task %s is already running", taskID)
	}
	task, err := a.getCertTaskByID(taskID)
	if err != nil {
		return "", err
	}
	conn, err := a.ensureCertConnection(task.ConnectionID)
	if err != nil {
		return "", err
	}
	ecc := strings.HasPrefix(task.KeyLength, "ec-")
	if err := a.setCertTaskStatus(taskID, models.CertStatusRunning, ""); err != nil {
		return "", err
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.certManager.Register(taskID, cancel)
	go func() {
		defer a.certManager.Unregister(taskID)
		err := a.certManager.Remove(ctx, conn, task.PrimaryDomain, ecc, taskID)
		status := models.CertStatusIdle
		errMsg := ""
		if err != nil {
			status = models.CertStatusFailed
			errMsg = err.Error()
		}
		if deleteTask && err == nil {
			if _, dbErr := a.db.Exec("DELETE FROM cert_tasks WHERE id = ?", taskID); dbErr == nil {
				a.wailsApp.Event.Emit("cert:task-deleted", map[string]any{"taskID": taskID})
				return
			}
		}
		if err := a.setCertTaskStatus(taskID, status, errMsg); err != nil {
			return
		}
		a.emitCertTaskUpdated(taskID)
	}()
	return taskID, nil
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// resetInterruptedCertTasks flips tasks stuck in "running" (after an app
// crash/restart) to "failed"; called during startup.
func (a *AppService) resetInterruptedCertTasks() {
	a.db.Exec("UPDATE cert_tasks SET last_status = ?, last_error = ?, updated_at = CURRENT_TIMESTAMP WHERE last_status = ?",
		string(models.CertStatusFailed), "interrupted by app restart", string(models.CertStatusRunning))
}
