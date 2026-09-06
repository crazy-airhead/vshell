package cert

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	xssh "golang.org/x/crypto/ssh"
)

// RunOptions tunes a streamed remote command.
type RunOptions struct {
	// Timeout bounds the command; 0 means no timeout.
	Timeout time.Duration
	// OnLine, when set, receives every output line as it arrives.
	OnLine func(stream, line string)
}

// RunResult is the outcome of a streamed remote command.
type RunResult struct {
	ExitCode int    // -1 when no exit status is available
	Combined string // full output, capped at maxCombinedOutput bytes
}

const maxCombinedOutput = 1 << 20 // 1MB

// Run executes cmd on a new session of the connection's SSH client and
// streams its output line by line. No PTY is requested so stdout/stderr stay
// separated and clean.
func (m *Manager) Run(ctx context.Context, connectionID, cmd string, opts RunOptions) (RunResult, error) {
	client, err := m.ssh.GetSSHClient(connectionID)
	if err != nil {
		return RunResult{ExitCode: -1}, err
	}
	return runOnClient(ctx, client, cmd, opts)
}

// runOnClient streams a command on the given SSH client. Split from Run so
// integration tests can exercise it with a directly dialed client.
func runOnClient(ctx context.Context, client *xssh.Client, cmd string, opts RunOptions) (RunResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	sess, err := client.NewSession()
	if err != nil {
		return RunResult{ExitCode: -1}, fmt.Errorf("new session: %w", err)
	}
	defer sess.Close()

	stdout, err := sess.StdoutPipe()
	if err != nil {
		return RunResult{ExitCode: -1}, fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := sess.StderrPipe()
	if err != nil {
		return RunResult{ExitCode: -1}, fmt.Errorf("stderr pipe: %w", err)
	}

	// Pipes must be in place before the exec request so no early output is
	// lost. Start (not Run) because Run installs its own stdout/stderr.
	if err := sess.Start(cmd); err != nil {
		return RunResult{ExitCode: -1}, fmt.Errorf("start %q: %w", cmd, err)
	}

	res := RunResult{ExitCode: -1}
	var mu sync.Mutex
	var sb strings.Builder

	readLines := func(r io.Reader, stream string, wg *sync.WaitGroup) {
		defer wg.Done()
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 64*1024), 512*1024)
		for sc.Scan() {
			line := sc.Text()
			mu.Lock()
			if sb.Len() < maxCombinedOutput {
				sb.WriteString(line)
				sb.WriteByte('\n')
			}
			mu.Unlock()
			if opts.OnLine != nil {
				opts.OnLine(stream, line)
			}
		}
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go readLines(stdout, "stdout", &wg)
	go readLines(stderr, "stderr", &wg)

	// Cancel path: closing the session tears down the channel and makes
	// Wait return. Signal(ssh.SIGKILL) is not portable across sshd builds.
	waitDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			sess.Close()
		case <-waitDone:
		}
	}()

	waitErr := sess.Wait()
	close(waitDone)
	wg.Wait()

	mu.Lock()
	res.Combined = sb.String()
	mu.Unlock()
	var exitErr *xssh.ExitError
	if errors.As(waitErr, &exitErr) {
		res.ExitCode = exitErr.ExitStatus()
	} else if waitErr == nil {
		res.ExitCode = 0
	}
	if ctx.Err() != nil {
		return res, fmt.Errorf("command canceled or timed out: %w", ctx.Err())
	}
	if res.ExitCode != 0 {
		// Non-zero exit (including transport errors with -1) is an error so
		// orchestration stages never mistake a failed acme.sh run for
		// success. res.Combined still carries the output for diagnostics.
		if res.ExitCode < 0 && waitErr != nil {
			return res, fmt.Errorf("run remote command: %w", waitErr)
		}
		return res, fmt.Errorf("remote command exited with status %d", res.ExitCode)
	}
	return res, nil
}

// outputTail returns the last n lines of output for error messages.
func outputTail(output string, n int) string {
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
