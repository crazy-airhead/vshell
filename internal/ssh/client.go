package ssh

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"vshell/internal/crypto"
	"vshell/internal/models"
)

type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	crypto   *crypto.CryptoService
	onEvent  func(event string, data any)
}

func NewManager(cryptoSvc *crypto.CryptoService, onEvent func(string, any)) *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
		crypto:   cryptoSvc,
		onEvent:  onEvent,
	}
}

func (m *Manager) Connect(conn *models.Connection) (*Session, error) {
	config, err := m.buildSSHConfig(conn)
	if err != nil {
		return nil, fmt.Errorf("build ssh config: %w", err)
	}

	var client *ssh.Client

	if conn.JumpHostID != nil && *conn.JumpHostID != "" {
		return nil, fmt.Errorf("jump host connections not yet implemented")
	}

	addr := fmt.Sprintf("%s:%d", conn.Host, conn.Port)
	client, err = ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}

	session, err := newSession(client, conn.ID, m.onEvent)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("create session: %w", err)
	}

	m.mu.Lock()
	m.sessions[conn.ID] = session
	m.mu.Unlock()

	return session, nil
}

func (m *Manager) Disconnect(connectionID string) {
	m.mu.Lock()
	s, ok := m.sessions[connectionID]
	if ok {
		delete(m.sessions, connectionID)
	}
	m.mu.Unlock()

	if ok {
		s.Close()
	}
}

func (m *Manager) GetSession(connectionID string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[connectionID]
	return s, ok
}

func (m *Manager) buildSSHConfig(conn *models.Connection) (*ssh.ClientConfig, error) {
	authMethods := []ssh.AuthMethod{}

	switch conn.AuthType {
	case models.AuthPassword:
		password, err := m.crypto.Decrypt(conn.Password)
		if err != nil {
			return nil, fmt.Errorf("decrypt password: %w", err)
		}
		authMethods = append(authMethods, ssh.Password(password))

	case models.AuthPrivateKey:
		keyData, err := m.crypto.Decrypt(conn.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("decrypt private key: %w", err)
		}
		var signer ssh.Signer
		if conn.KeyPassphrase != "" {
			passphrase, err := m.crypto.Decrypt(conn.KeyPassphrase)
			if err != nil {
				return nil, fmt.Errorf("decrypt key passphrase: %w", err)
			}
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(keyData), []byte(passphrase))
			if err != nil {
				return nil, fmt.Errorf("parse private key with passphrase: %w", err)
			}
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(keyData))
			if err != nil {
				return nil, fmt.Errorf("parse private key: %w", err)
			}
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))

	case models.AuthAgent:
		return nil, fmt.Errorf("ssh agent auth not yet implemented")

	case models.AuthInteractive:
		password, err := m.crypto.Decrypt(conn.Password)
		if err != nil {
			return nil, fmt.Errorf("decrypt password: %w", err)
		}
		authMethods = append(authMethods, ssh.KeyboardInteractive(
			func(user, instruction string, questions []string, echos []bool) ([]string, error) {
				answers := make([]string, len(questions))
				for i := range questions {
					answers[i] = password
				}
				return answers, nil
			},
		))

	default:
		return nil, fmt.Errorf("unsupported auth type: %s", conn.AuthType)
	}

	return &ssh.ClientConfig{
		User:            conn.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: implement host key verification
		Timeout:         10 * time.Second,
	}, nil
}

func (m *Manager) NewMonitorSession(connectionID string) (*ssh.Session, *ssh.Client, error) {
	m.mu.RLock()
	s, ok := m.sessions[connectionID]
	m.mu.RUnlock()

	if !ok {
		return nil, nil, fmt.Errorf("no active session for connection %s", connectionID)
	}

	session, err := s.client.NewSession()
	if err != nil {
		return nil, nil, fmt.Errorf("new monitor session: %w", err)
	}

	return session, s.client, nil
}

// WriteStdin writes data to the stdin pipe of the session identified by connectionID.
func (m *Manager) WriteStdin(connectionID string, data []byte) error {
	m.mu.RLock()
	s, ok := m.sessions[connectionID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("no active session for connection %s", connectionID)
	}

	_, err := s.WriteStdin(data)
	return err
}

// ResizeWindow resizes the PTY for the session identified by connectionID.
func (m *Manager) ResizeWindow(connectionID string, rows, cols int) error {
	m.mu.RLock()
	s, ok := m.sessions[connectionID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("no active session for connection %s", connectionID)
	}

	return s.ResizeWindow(rows, cols)
}

// ExecOnConnection runs a command on a new temporary session using the existing client.
func (m *Manager) ExecOnConnection(connectionID string, cmd string) (string, error) {
	m.mu.RLock()
	s, ok := m.sessions[connectionID]
	m.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("no active session for connection %s", connectionID)
	}

	sess, err := s.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("new session: %w", err)
	}
	defer sess.Close()

	out, err := sess.CombinedOutput(cmd)
	if err != nil {
		return string(out), fmt.Errorf("exec %q: %w", cmd, err)
	}
	return string(out), nil
}

// GetSSHClient returns the underlying SSH client for a connection.
func (m *Manager) GetSSHClient(connectionID string) (*ssh.Client, error) {
	m.mu.RLock()
	s, ok := m.sessions[connectionID]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("no active session for connection %s", connectionID)
	}
	return s.client, nil
}
