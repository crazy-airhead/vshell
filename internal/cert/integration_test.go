//go:build integration

// Integration smoke test against a real SSH server (e.g. a docker
// openssh-server container). Not run by plain `go test`; enable with:
//
//	go test -tags integration ./internal/cert/ -run TestIntegration -v
//
// Required env:
//	VSHELL_CERT_TEST_ADDR  host:port of the SSH server (e.g. localhost:2222)
//	VSHELL_CERT_TEST_KEY   path to a PEM private key authorized on the server
//	VSHELL_CERT_TEST_USER  username (default: root)
//	VSHELL_CERT_TEST_INSTALL  set to "1" to also install acme.sh (needs outbound network)
package cert

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	xssh "golang.org/x/crypto/ssh"

	"vshell/internal/models"
)

func dialTestServer(t *testing.T) *xssh.Client {
	t.Helper()
	addr := os.Getenv("VSHELL_CERT_TEST_ADDR")
	keyPath := os.Getenv("VSHELL_CERT_TEST_KEY")
	if addr == "" || keyPath == "" {
		t.Skip("VSHELL_CERT_TEST_ADDR / VSHELL_CERT_TEST_KEY not set")
	}
	user := os.Getenv("VSHELL_CERT_TEST_USER")
	if user == "" {
		user = "root"
	}
	key, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("read key: %v", err)
	}
	signer, err := xssh.ParsePrivateKey(key)
	if err != nil {
		t.Fatalf("parse key: %v", err)
	}
	client, err := xssh.Dial("tcp", addr, &xssh.ClientConfig{
		User:            user,
		Auth:            []xssh.AuthMethod{xssh.PublicKeys(signer)},
		HostKeyCallback: xssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	})
	if err != nil {
		t.Fatalf("dial %s: %v", addr, err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

// run is a tiny Manager-free wrapper mirroring Manager.stream for tests.
func run(t *testing.T, client *xssh.Client, cmd string, timeout time.Duration) (RunResult, error) {
	t.Helper()
	return runOnClient(context.Background(), client, cmd, RunOptions{Timeout: timeout})
}

func newTestTask() *models.CertTask {
	return &models.CertTask{
		PrimaryDomain: "example.com",
		KeyLength:     "ec-256",
		DNSSleep:      120,
	}
}

func TestIntegrationDetectAndCommands(t *testing.T) {
	client := dialTestServer(t)
	ctx := context.Background()

	// 1. Detect script: must parse into a usable environment.
	res, err := run(t, client, BuildDetectScript(), 30*time.Second)
	if err != nil {
		t.Fatalf("detect: %v\noutput:\n%s", err, res.Combined)
	}
	env := ParseDetectOutput(res.Combined)
	env.ConnectionID = "test"
	if env.Home == "" {
		t.Fatalf("detect returned empty $HOME; output:\n%s", res.Combined)
	}
	if !strings.HasPrefix(env.AcmeShPath, env.Home) {
		t.Errorf("AcmeShPath %q not under $HOME %q", env.AcmeShPath, env.Home)
	}
	t.Logf("detect: home=%s curl=%v acmesh=%v cron=%v", env.Home, env.CurlPresent, env.Installed, env.CronPresent)

	// 2. List on a box without acme.sh: exit 127, but the dir-enumeration
	// fallback (same as Manager.ListCerts) must yield an empty list.
	listRes, listErr := run(t, client, BuildListCmd(), 30*time.Second)
	var certs []models.RemoteCert
	if listErr != nil && listRes.ExitCode < 0 {
		t.Fatalf("list transport error: %v", listErr)
	}
	certs = ParseListOutput(listRes.Combined)
	if len(certs) == 0 {
		dirRes, dirErr := run(t, client, BuildListDomainsCmd(), 30*time.Second)
		if dirErr != nil {
			t.Fatalf("list fallback: %v\noutput:\n%s", dirErr, dirRes.Combined)
		}
		certs = ParseAcmeDirsOutput(dirRes.Combined)
	}
	t.Logf("list: %d certs (list exit=%d err=%v)", len(certs), listRes.ExitCode, listErr)

	// 3. Ensure-cron: as non-root on many images crontab is unavailable —
	// accept either success+marker or a clear failure (the orchestration
	// surfaces it to the user as a failed cron stage).
	cronRes, cronErr := run(t, client, BuildEnsureCronCmd(), 30*time.Second)
	if cronErr != nil {
		t.Logf("ensure cron failed (may be expected for non-root): exit=%d err=%v", cronRes.ExitCode, cronErr)
	} else {
		res, err = run(t, client, BuildDetectScript(), 30*time.Second)
		if err != nil {
			t.Fatalf("re-detect: %v", err)
		}
		if !ParseDetectOutput(res.Combined).CronPresent {
			t.Errorf("cron marker missing after successful ensure; output:\n%s", res.Combined)
		}
	}

	// 4. Tail log on a fresh box must succeed (marker line, not an error).
	res, err = run(t, client, BuildTailLogCmd(), 30*time.Second)
	if err != nil {
		t.Fatalf("tail log: %v", err)
	}
	if !strings.Contains(res.Combined, "vshell:log=none") && !strings.Contains(res.Combined, "acme") {
		t.Logf("tail log output: %s", res.Combined)
	}

	// 5. Exit codes propagate: a failing command returns exit status AND an
	// error (orchestration must never treat failure as success).
	res, err = run(t, client, "exit 7", 30*time.Second)
	if err == nil || res.ExitCode != 7 {
		t.Errorf("expected exit 7 with error, got exit=%d err=%v", res.ExitCode, err)
	}

	// 6. stderr lines arrive on the stderr stream.
	var stderrLines []string
	_, err = runOnClient(ctx, client, "echo out; echo err 1>&2", RunOptions{
		Timeout: 30 * time.Second,
		OnLine: func(stream, line string) {
			if stream == "stderr" {
				stderrLines = append(stderrLines, line)
			}
		},
	})
	if err != nil {
		t.Fatalf("stream split: %v", err)
	}
	if len(stderrLines) != 1 || stderrLines[0] != "err" {
		t.Errorf("stderr lines = %v, want [err]", stderrLines)
	}

	// 7. Optional: full acme.sh install (network required).
	if os.Getenv("VSHELL_CERT_TEST_INSTALL") == "1" {
		res, err := run(t, client, BuildInstallCmd("test@example.com"), 5*time.Minute)
		if err != nil {
			t.Fatalf("install acme.sh: %v\noutput tail:\n%s", err, outputTail(res.Combined, 20))
		}
		res, err = run(t, client, BuildSetDefaultCACmd(), 30*time.Second)
		if err != nil {
			t.Fatalf("set-default-ca: %v\noutput:\n%s", err, res.Combined)
		}
		res, err = run(t, client, BuildDetectScript(), 30*time.Second)
		if err != nil {
			t.Fatalf("re-detect after install: %v", err)
		}
		if !ParseDetectOutput(res.Combined).Installed {
			t.Errorf("acme.sh still not installed after install; output:\n%s", res.Combined)
		}
		t.Log("acme.sh installed OK")
	}
}

func TestIntegrationIssueCmdShape(t *testing.T) {
	client := dialTestServer(t)

	// The issue command chain must exit with acme.sh's code even though it
	// references a missing env file (chmod fails) — proves the rc=$?/exit
	// chain works under a real shell.
	task := newTestTask()
	res, err := run(t, client, BuildIssueCmd("/tmp/.vshell_acme_missing.env", task, "dns_cf"), 60*time.Second)
	if err == nil || res.ExitCode == 0 {
		t.Errorf("issue with missing env file should fail, got exit=%d err=%v", res.ExitCode, err)
	}
	if _, err := run(t, client, "test ! -e /tmp/.vshell_acme_missing.env && echo gone", 10*time.Second); err != nil {
		t.Errorf("probe: %v", err)
	}
}
