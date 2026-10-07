package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"vshell/internal/crypto"
	"vshell/internal/models"
)

// testSSHServer is a minimal in-process SSH server that accepts session
// channels up to a per-connection limit and refuses further opens with
// "administratively prohibited" — mimicking OpenSSH's MaxSessions behavior.
type testSSHServer struct {
	ln          net.Listener
	config      *ssh.ServerConfig
	maxSessions int
	failFirst   bool // accept the first TCP conn, finish handshake, then drop it

	conns int
	mu    sync.Mutex
	wg    sync.WaitGroup
}

func newTestSSHServer(t *testing.T, maxSessions int, failFirst bool) *testSSHServer {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate host key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}
	config := &ssh.ServerConfig{
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if conn.User() == "tester" && string(password) == "pw" {
				return &ssh.Permissions{}, nil
			}
			return nil, fmt.Errorf("auth rejected for %q", conn.User())
		},
	}
	config.AddHostKey(signer)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	s := &testSSHServer{ln: ln, config: config, maxSessions: maxSessions, failFirst: failFirst}
	s.wg.Add(1)
	go s.serve()
	t.Cleanup(func() {
		ln.Close()
		s.wg.Wait()
	})
	return s
}

func (s *testSSHServer) addr() string { return s.ln.Addr().String() }

func (s *testSSHServer) serve() {
	defer s.wg.Done()
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		s.mu.Lock()
		s.conns++
		n := s.conns
		s.mu.Unlock()

		if s.failFirst && n == 1 {
			// Simulate a server-side idle disconnect on a cached client:
			// complete the SSH handshake, then drop the TCP connection.
			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				_, chans, reqs, err := ssh.NewServerConn(conn, s.config)
				if err != nil {
					return
				}
				conn.Close()
				go ssh.DiscardRequests(reqs)
				for range chans {
					// conn is closed; nothing to serve
				}
			}()
			continue
		}

		s.wg.Add(1)
		go s.handleConn(conn)
	}
}

func (s *testSSHServer) handleConn(conn net.Conn) {
	defer s.wg.Done()
	sconn, chans, reqs, err := ssh.NewServerConn(conn, s.config)
	if err != nil {
		return
	}
	defer sconn.Close()
	go ssh.DiscardRequests(reqs)

	var openMu sync.Mutex
	open := 0
	for newCh := range chans {
		if newCh.ChannelType() != "session" {
			newCh.Reject(ssh.UnknownChannelType, "unsupported")
			continue
		}
		openMu.Lock()
		refuse := open >= s.maxSessions
		if !refuse {
			open++
		}
		openMu.Unlock()
		if refuse {
			newCh.Reject(ssh.Prohibited, "too many sessions")
			continue
		}

		ch, requests, err := newCh.Accept()
		if err != nil {
			openMu.Lock()
			open--
			openMu.Unlock()
			continue
		}
		go func() {
			defer func() {
				openMu.Lock()
				open--
				openMu.Unlock()
				ch.Close()
			}()
			// Drain stdin so client writes never block on flow control.
			go io.Copy(io.Discard, ch)
			for req := range requests {
				switch req.Type {
				case "pty-req", "shell":
					if req.WantReply {
						req.Reply(true, nil)
					}
				default:
					if req.WantReply {
						req.Reply(false, nil)
					}
				}
			}
		}()
	}
}

func testManager(t *testing.T) *Manager {
	t.Helper()
	// crypto.New() is a process-wide singleton reading VSHELL_ENCRYPTION_KEY;
	// set a fixed key before first use so Encrypt/Decrypt round-trip in tests.
	os.Setenv("VSHELL_ENCRYPTION_KEY", base64.StdEncoding.EncodeToString(make([]byte, 32)))
	return NewManager(crypto.New(), func(string, any) {})
}

func testConnection(t *testing.T, mgr *Manager, host string, port int) *models.Connection {
	t.Helper()
	enc, err := mgr.crypto.Encrypt("pw")
	if err != nil {
		t.Fatalf("encrypt password: %v", err)
	}
	t.Cleanup(func() { mgr.Disconnect("conn-1") }) // release conns before server cleanup wg.Wait
	return &models.Connection{
		ID:         "conn-1",
		Host:       host,
		Port:       port,
		Username:   "tester",
		AuthType:   models.AuthPassword,
		Password:   enc,
		UploadPath: "/",
	}
}

func dialTarget(t *testing.T, srv *testSSHServer) (string, int) {
	t.Helper()
	host, portStr, err := net.SplitHostPort(srv.addr())
	if err != nil {
		t.Fatalf("split addr: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}
	return host, port
}

// TestConnectDoesNotEvictSharedClientOnSessionRefusal is the regression test
// for ISSUE-0010: when the server refuses a new session channel (MaxSessions
// limit reached), the shared SSH client must survive — every terminal already
// open on it must stay connected.
func TestConnectDoesNotEvictSharedClientOnSessionRefusal(t *testing.T) {
	const maxSessions = 3
	srv := newTestSSHServer(t, maxSessions, false)
	host, port := dialTarget(t, srv)
	mgr := testManager(t)
	conn := testConnection(t, mgr, host, port)

	sessions := make([]*Session, 0, maxSessions)
	for i := 0; i < maxSessions; i++ {
		s, err := mgr.Connect(conn, fmt.Sprintf("sess-%d", i))
		if err != nil {
			t.Fatalf("open #%d: %v", i+1, err)
		}
		sessions = append(sessions, s)
	}

	// The (maxSessions+1)-th open must fail with a refusal error — and must
	// NOT evict/close the shared client.
	_, err := mgr.Connect(conn, "sess-over")
	if err == nil {
		t.Fatal("expected the over-limit open to fail")
	}
	if !strings.Contains(err.Error(), "MaxSessions") {
		t.Errorf("expected a MaxSessions hint in the error, got: %v", err)
	}

	time.Sleep(100 * time.Millisecond) // let any wrongful eviction settle

	if _, err := mgr.GetSSHClient(conn.ID); err != nil {
		t.Fatalf("shared client was evicted after a session refusal: %v", err)
	}
	for i, s := range sessions {
		select {
		case <-s.Done():
			t.Fatalf("session %d/%d was closed by the failed over-limit open", i+1, maxSessions)
		default:
		}
		if _, err := s.WriteStdin([]byte("echo hi\n")); err != nil {
			t.Fatalf("session %d/%d stdin write after refusal: %v", i+1, maxSessions, err)
		}
	}
}

// TestConnectReconnectsAfterCachedClientDies ensures the evict-and-redial path
// still works for genuine connection-level failures (server dropped the TCP
// connection, e.g. idle timeout): the new tab must connect transparently.
func TestConnectReconnectsAfterCachedClientDies(t *testing.T) {
	srv := newTestSSHServer(t, 10, true)
	host, port := dialTarget(t, srv)
	mgr := testManager(t)
	conn := testConnection(t, mgr, host, port)

	s, err := mgr.Connect(conn, "sess-1")
	if err != nil {
		t.Fatalf("connect after dead cached client: %v", err)
	}
	if _, err := s.WriteStdin([]byte("echo hi\n")); err != nil {
		t.Fatalf("stdin write on reconnected session: %v", err)
	}
}
