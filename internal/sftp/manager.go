package sftp

import (
	"fmt"
	"sync"

	"vshell/internal/crypto"
	vshellssh "vshell/internal/ssh"
)

type Manager struct {
	mu      sync.RWMutex
	clients map[string]*Client
	sshMgr  *vshellssh.Manager
	crypto  *crypto.CryptoService
	onEvent func(string, any)
}

func NewManager(sshMgr *vshellssh.Manager, cryptoSvc *crypto.CryptoService, onEvent func(string, any)) *Manager {
	return &Manager{
		clients: make(map[string]*Client),
		sshMgr:  sshMgr,
		crypto:  cryptoSvc,
		onEvent: onEvent,
	}
}

func (m *Manager) GetOrCreateClient(connectionID string) (*Client, error) {
	m.mu.RLock()
	if c, ok := m.clients[connectionID]; ok {
		m.mu.RUnlock()
		return c, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if c, ok := m.clients[connectionID]; ok {
		return c, nil
	}

	sshClient, err := m.sshMgr.GetSSHClient(connectionID)
	if err != nil {
		return nil, fmt.Errorf("get ssh client: %w", err)
	}

	client, err := NewClient(sshClient, m.crypto, m.onEvent)
	if err != nil {
		return nil, fmt.Errorf("create sftp client: %w", err)
	}

	m.clients[connectionID] = client
	return client, nil
}

func (m *Manager) ReadDir(connectionID, path string) ([]FileInfo, error) {
	client, err := m.GetOrCreateClient(connectionID)
	if err != nil {
		return nil, err
	}
	return client.ReadDir(path)
}

func (m *Manager) UploadFile(connectionID, localPath, remotePath string) error {
	client, err := m.GetOrCreateClient(connectionID)
	if err != nil {
		return err
	}
	return client.UploadFile(localPath, remotePath)
}

func (m *Manager) DownloadFile(connectionID, remotePath, localPath string) error {
	client, err := m.GetOrCreateClient(connectionID)
	if err != nil {
		return err
	}
	return client.DownloadFile(remotePath, localPath)
}

func (m *Manager) CloseClient(connectionID string) {
	m.mu.Lock()
	c, ok := m.clients[connectionID]
	if ok {
		delete(m.clients, connectionID)
	}
	m.mu.Unlock()

	if ok {
		c.Close()
	}
}

func (m *Manager) CloseAll() {
	m.mu.Lock()
	clients := m.clients
	m.clients = make(map[string]*Client)
	m.mu.Unlock()

	for _, c := range clients {
		c.Close()
	}
}
