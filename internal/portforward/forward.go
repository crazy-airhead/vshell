package portforward

import (
	"fmt"
	"io"
	"net"
	"sync"

	"golang.org/x/crypto/ssh"

	"vshell/internal/models"
)

type Manager struct {
	mu        sync.Mutex
	listeners map[string][]net.Listener
	forwards  map[string]*Forward
}

type Forward struct {
	ID           string
	ConnectionID string
	Type         models.ForwardType
	LocalHost    string
	LocalPort    int
	RemoteHost   string
	RemotePort   int
	Listener     net.Listener
	sshClient    *ssh.Client
}

func NewManager() *Manager {
	return &Manager{
		listeners: make(map[string][]net.Listener),
		forwards:  make(map[string]*Forward),
	}
}

func (m *Manager) StartLocal(sshClient *ssh.Client, fwd *models.PortForward) error {
	addr := fmt.Sprintf("%s:%d", fwd.LocalHost, fwd.LocalPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}

	f := &Forward{
		ID:           fwd.ID,
		ConnectionID: fwd.ConnectionID,
		Type:         fwd.Type,
		LocalHost:    fwd.LocalHost,
		LocalPort:    fwd.LocalPort,
		RemoteHost:   fwd.RemoteHost,
		RemotePort:   fwd.RemotePort,
		Listener:     listener,
		sshClient:    sshClient,
	}

	m.mu.Lock()
	m.listeners[fwd.ConnectionID] = append(m.listeners[fwd.ConnectionID], listener)
	m.forwards[fwd.ID] = f
	m.mu.Unlock()

	go m.acceptLoop(f)

	return nil
}

func (m *Manager) acceptLoop(f *Forward) {
	for {
		localConn, err := f.Listener.Accept()
		if err != nil {
			return
		}

		remoteAddr := fmt.Sprintf("%s:%d", f.RemoteHost, f.RemotePort)
		remoteConn, err := f.sshClient.Dial("tcp", remoteAddr)
		if err != nil {
			localConn.Close()
			continue
		}

		go func() {
			io.Copy(localConn, remoteConn)
			localConn.Close()
		}()
		go func() {
			io.Copy(remoteConn, localConn)
			remoteConn.Close()
		}()
	}
}

func (m *Manager) Stop(forwardID string) error {
	m.mu.Lock()
	f, ok := m.forwards[forwardID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("forward %s not found", forwardID)
	}
	delete(m.forwards, forwardID)

	// Remove from connection's listener list
	listeners := m.listeners[f.ConnectionID]
	for i, l := range listeners {
		if l == f.Listener {
			m.listeners[f.ConnectionID] = append(listeners[:i], listeners[i+1:]...)
			break
		}
	}
	m.mu.Unlock()

	return f.Listener.Close()
}

// Forward returns the forward by ID, or nil if not found.
func (m *Manager) Forward(id string) (*Forward, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	f, ok := m.forwards[id]
	return f, ok
}

// RunningIDs returns the IDs of all currently running forwards.
func (m *Manager) RunningIDs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	ids := make([]string, 0, len(m.forwards))
	for id := range m.forwards {
		ids = append(ids, id)
	}
	return ids
}

// ActiveCount returns the number of running forwards for a connection.
func (m *Manager) ActiveCount(connectionID string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.listeners[connectionID])
}

func (m *Manager) StopAllForConnection(connectionID string) {
	m.mu.Lock()
	listeners := m.listeners[connectionID]
	delete(m.listeners, connectionID)
	m.mu.Unlock()

	for _, l := range listeners {
		l.Close()
	}
}
