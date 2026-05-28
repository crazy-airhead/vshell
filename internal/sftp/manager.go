package sftp

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"

	"vshell/internal/crypto"
	vshellssh "vshell/internal/ssh"
)

type Manager struct {
	mu      sync.RWMutex
	clients map[string]*Client
	sshMgr  *vshellssh.Manager
	crypto  *crypto.CryptoService
	onEvent func(string, any)
	sem     chan struct{}

	cancelMu sync.Mutex
	cancels  map[string]context.CancelFunc
}

func NewManager(sshMgr *vshellssh.Manager, cryptoSvc *crypto.CryptoService, onEvent func(string, any)) *Manager {
	return &Manager{
		clients: make(map[string]*Client),
		sshMgr:  sshMgr,
		crypto:  cryptoSvc,
		onEvent: onEvent,
		sem:     make(chan struct{}, 3),
		cancels: make(map[string]context.CancelFunc),
	}
}

func (m *Manager) registerCancel(id string, cancel context.CancelFunc) {
	m.cancelMu.Lock()
	m.cancels[id] = cancel
	m.cancelMu.Unlock()
}

func (m *Manager) unregisterCancel(id string) {
	m.cancelMu.Lock()
	delete(m.cancels, id)
	m.cancelMu.Unlock()
}

func (m *Manager) CancelAllTransfers() {
	m.cancelMu.Lock()
	for id, cancel := range m.cancels {
		cancel()
		delete(m.cancels, id)
	}
	m.cancelMu.Unlock()
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
	transferID := newTransferID()
	fileName := filepath.Base(localPath)
	ctx, cancel := context.WithCancel(context.Background())
	m.registerCancel(transferID, cancel)
	defer m.unregisterCancel(transferID)

	m.onEvent("sftp:progress", TransferProgress{
		ID: transferID, FileName: fileName, Direction: DirUpload,
	})

	m.sem <- struct{}{}
	defer func() { <-m.sem }()

	client, err := m.GetOrCreateClient(connectionID)
	if err != nil {
		m.emitDone(transferID, fileName, DirUpload, err)
		return err
	}
	return client.Upload(ctx, transferID, localPath, remotePath)
}

func (m *Manager) DownloadFile(connectionID, remotePath, localPath string) error {
	transferID := newTransferID()
	fileName := filepath.Base(remotePath)
	ctx, cancel := context.WithCancel(context.Background())
	m.registerCancel(transferID, cancel)
	defer m.unregisterCancel(transferID)

	m.onEvent("sftp:progress", TransferProgress{
		ID: transferID, FileName: fileName, Direction: DirDownload,
	})

	m.sem <- struct{}{}
	defer func() { <-m.sem }()

	client, err := m.GetOrCreateClient(connectionID)
	if err != nil {
		m.emitDone(transferID, fileName, DirDownload, err)
		return err
	}
	return client.Download(ctx, transferID, remotePath, localPath)
}

func (m *Manager) emitDone(id, fileName string, dir Direction, err error) {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	m.onEvent("sftp:progress", TransferProgress{
		ID: id, FileName: fileName, Done: true, Error: errStr, Direction: dir,
	})
}

func newTransferID() string {
	return fmt.Sprintf("tx-%d", nextTransferSeq())
}

var transferSeq int64

func nextTransferSeq() int64 {
	return atomic.AddInt64(&transferSeq, 1)
}

func (m *Manager) RemoveFile(connectionID, remotePath string) error {
	m.sem <- struct{}{}
	defer func() { <-m.sem }()
	client, err := m.GetOrCreateClient(connectionID)
	if err != nil {
		return err
	}
	return client.Remove(remotePath)
}

func (m *Manager) ReadFileContent(connectionID, remotePath string) (string, error) {
	client, err := m.GetOrCreateClient(connectionID)
	if err != nil {
		return "", err
	}
	return client.ReadFileContent(remotePath)
}

func (m *Manager) WriteFileContent(connectionID, remotePath, content string) error {
	client, err := m.GetOrCreateClient(connectionID)
	if err != nil {
		return err
	}
	return client.WriteFileContent(remotePath, content)
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
