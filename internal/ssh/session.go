package ssh

import (
	"fmt"
	"io"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// flushInterval controls how often buffered stdout data is flushed to the frontend.
const flushInterval = 50 * time.Millisecond

type Session struct {
	id          string
	client      *ssh.Client
	session     *ssh.Session
	stdinWriter io.WriteCloser
	onEvent     func(string, any)
	closeOnce   sync.Once
	done        chan struct{}
}

func newSession(client *ssh.Client, sessionID string, onEvent func(string, any)) (*Session, error) {
	sess, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("new ssh session: %w", err)
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := sess.RequestPty("xterm-256color", 24, 80, modes); err != nil {
		sess.Close()
		return nil, fmt.Errorf("request pty: %w", err)
	}

	// Create all pipes BEFORE starting the shell
	stdin, err := sess.StdinPipe()
	if err != nil {
		sess.Close()
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	stdoutPipe, err := sess.StdoutPipe()
	if err != nil {
		sess.Close()
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	stderrPipe, err := sess.StderrPipe()
	if err != nil {
		sess.Close()
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := sess.Shell(); err != nil {
		sess.Close()
		return nil, fmt.Errorf("start shell: %w", err)
	}

	s := &Session{
		id:          sessionID,
		client:      client,
		session:     sess,
		stdinWriter: stdin,
		onEvent:     onEvent,
		done:        make(chan struct{}),
	}

	stdoutFW := newFlushingWriter(sessionID, "terminal:stdout", onEvent)
	stderrFW := newFlushingWriter(sessionID, "terminal:stderr", onEvent)

	go func() {
		io.Copy(stdoutFW, stdoutPipe)
		stdoutFW.Flush()
	}()

	go func() {
		io.Copy(stderrFW, stderrPipe)
		stderrFW.Flush()
	}()

	go func() {
		sess.Wait()
		close(s.done)
		onEvent("terminal:closed", map[string]any{
			"sessionID": sessionID,
		})
	}()

	return s, nil
}

func (s *Session) ID() string {
	return s.id
}

func (s *Session) Close() {
	s.closeOnce.Do(func() {
		s.stdinWriter.Close()
		s.session.Close()
	})
}

func (s *Session) Done() <-chan struct{} {
	return s.done
}

// WriteStdin writes data to the SSH session's stdin.
func (s *Session) WriteStdin(data []byte) (int, error) {
	return s.stdinWriter.Write(data)
}

// ResizeWindow resizes the PTY.
func (s *Session) ResizeWindow(rows, cols int) error {
	return s.session.WindowChange(rows, cols)
}

// flushingWriter buffers writes and flushes them on a timer.
// This avoids per-byte event emissions while keeping latency low.
type flushingWriter struct {
	sessionID   string
	eventName   string
	onEvent     func(string, any)
	mu          sync.Mutex
	pending     []byte
	flushTicker *time.Ticker
	done        chan struct{}
}

func newFlushingWriter(sessionID, eventName string, onEvent func(string, any)) *flushingWriter {
	fw := &flushingWriter{
		sessionID:   sessionID,
		eventName:   eventName,
		onEvent:     onEvent,
		flushTicker: time.NewTicker(flushInterval),
		done:        make(chan struct{}),
	}

	go fw.tickFlush()

	return fw
}

func (fw *flushingWriter) Write(p []byte) (int, error) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	fw.pending = append(fw.pending, p...)
	return len(p), nil
}

func (fw *flushingWriter) Flush() {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	fw.doFlush()
}

func (fw *flushingWriter) doFlush() {
	if len(fw.pending) == 0 {
		return
	}
	data := make([]byte, len(fw.pending))
	copy(data, fw.pending)
	fw.pending = fw.pending[:0]
	fw.onEvent(fw.eventName, map[string]any{
		"sessionID": fw.sessionID,
		"data":      string(data),
	})
}

func (fw *flushingWriter) tickFlush() {
	for {
		select {
		case <-fw.flushTicker.C:
			fw.mu.Lock()
			fw.doFlush()
			fw.mu.Unlock()
		case <-fw.done:
			fw.flushTicker.Stop()
			return
		}
	}
}
