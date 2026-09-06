package cert

import (
	"context"
	"fmt"
	"strings"
	"time"

	"vshell/internal/models"
)

// Command timeouts in seconds, generous enough for slow networks.
const (
	timeoutQuick       = 30
	timeoutInfo        = 15
	timeoutInstall     = 300
	timeoutIssue       = 1200 // DNS propagation waits included
	timeoutInstallCert = 120
	timeoutRemove      = 60
)

func seconds(n int) time.Duration {
	return time.Duration(n) * time.Second
}

func nowMillis() int64 {
	return time.Now().UnixMilli()
}

// wrapStageError decorates a failed stage error with exit code and output
// tail so the UI can show something actionable.
func wrapStageError(stage string, res RunResult, err error) error {
	if res.ExitCode >= 0 {
		tail := outputTail(res.Combined, 15)
		if tail != "" {
			return fmt.Errorf("%s failed (exit %d):\n%s", stage, res.ExitCode, tail)
		}
		return fmt.Errorf("%s failed (exit %d)", stage, res.ExitCode)
	}
	return fmt.Errorf("%s: %w", stage, err)
}

// Detect gathers the acme.sh environment facts for a connection.
func (m *Manager) Detect(ctx context.Context, connectionID string) (models.CertEnvironment, error) {
	res, err := m.Run(ctx, connectionID, BuildDetectScript(), RunOptions{Timeout: seconds(timeoutQuick)})
	if err != nil {
		return models.CertEnvironment{}, wrapStageError(StageDetect, res, err)
	}
	env := ParseDetectOutput(res.Combined)
	env.ConnectionID = connectionID
	return env, nil
}

// ListCerts lists the certs managed by acme.sh on the server. It parses
// `--list`, and enriches every row with `--info` renewal data. The
// directory-enumeration fallback only runs when the `--list` table itself is
// unrecognised (format drift or acme.sh missing) — an empty table means
// genuinely no certs.
func (m *Manager) ListCerts(ctx context.Context, connectionID string) ([]models.RemoteCert, error) {
	res, err := m.Run(ctx, connectionID, BuildListCmd(), RunOptions{Timeout: seconds(timeoutQuick)})
	certs, recognized := ParseListOutput(res.Combined)
	if !recognized {
		if err != nil && res.ExitCode < 0 {
			// Transport-level failure (no exit status): surface it.
			return nil, wrapStageError("list", res, err)
		}
		// Format drift or acme.sh missing: enumerate ~/.acme.sh/<domain>[_ecc]
		// dirs; always exits 0 and yields an empty list on a fresh install.
		dirRes, dirErr := m.Run(ctx, connectionID, BuildListDomainsCmd(), RunOptions{Timeout: seconds(timeoutQuick)})
		if dirErr != nil {
			return nil, wrapStageError("list", dirRes, dirErr)
		}
		certs = ParseAcmeDirsOutput(dirRes.Combined)
	}
	m.enrichWithInfo(ctx, connectionID, certs)
	return certs, nil
}

func (m *Manager) enrichWithInfo(ctx context.Context, connectionID string, certs []models.RemoteCert) {
	now := time.Now()
	for i := range certs {
		info, err := m.CertInfo(ctx, connectionID, certs[i].MainDomain, certs[i].ECC)
		if err != nil || info == nil {
			continue
		}
		certs[i].NextRenewTime = info.NextRenewTime
		certs[i].DaysLeft = DaysLeft(info.NextRenewTime, now)
		if certs[i].CA == "" && info.Fields["Le_API"] != "" {
			certs[i].CA = caNameFromAPI(info.Fields["Le_API"])
		}
	}
}

func caNameFromAPI(api string) string {
	switch {
	case strings.Contains(api, "letsencrypt"):
		return "letsencrypt"
	case strings.Contains(api, "zerossl"):
		return "zerossl"
	default:
		return api
	}
}

// CertInfo returns the `--info` key=value data for one cert.
func (m *Manager) CertInfo(ctx context.Context, connectionID, domain string, ecc bool) (*models.RemoteCertInfo, error) {
	res, err := m.Run(ctx, connectionID, BuildInfoCmd(domain, ecc), RunOptions{Timeout: seconds(timeoutInfo)})
	if err != nil {
		return nil, wrapStageError("info", res, err)
	}
	return ParseInfoOutput(res.Combined), nil
}

// ReadServerLog tails the persistent acme.sh log on the server.
func (m *Manager) ReadServerLog(ctx context.Context, connectionID string) (string, error) {
	res, err := m.Run(ctx, connectionID, BuildTailLogCmd(), RunOptions{Timeout: seconds(timeoutQuick)})
	if err != nil {
		return "", wrapStageError("read log", res, err)
	}
	return res.Combined, nil
}

// InstallAcmeSh installs acme.sh on the server and switches the default CA
// to Let's Encrypt. opID correlates the emitted events.
func (m *Manager) InstallAcmeSh(ctx context.Context, conn *models.Connection, opID, email string) error {
	connectionID := conn.ID
	m.stage("", opID, connectionID, StageInstall, "start", "")
	_, err := m.stream(ctx, connectionID, "", opID, StageInstall, BuildInstallCmd(email), timeoutInstall)
	if err == nil {
		_, err = m.stream(ctx, connectionID, "", opID, StageInstall, BuildSetDefaultCACmd(), timeoutQuick)
	}
	if err == nil {
		env, detectErr := m.Detect(ctx, connectionID)
		if detectErr == nil && !env.Installed {
			// The get.acme.sh bootstrap can exit 0 while failing (e.g. no
			// cron on the server: "Pre-check failed"), so verify here.
			if !env.CronPresent {
				err = fmt.Errorf("acme.sh still not found at %s after install; the server appears to lack a working crontab, which the acme.sh installer requires", env.AcmeShPath)
			} else {
				err = fmt.Errorf("acme.sh still not found at %s after install", env.AcmeShPath)
			}
		}
	}
	if err != nil {
		m.stage("", opID, connectionID, StageInstall, "fail", err.Error())
		return err
	}
	m.stage("", opID, connectionID, StageInstall, "ok", "")
	return nil
}

// Issue runs the full issuance flow for a task: environment detection,
// optional acme.sh install, credential upload (via SFTP, deleted right
// after sourcing), `--issue`, optional `--install-cert`, cron check.
func (m *Manager) Issue(ctx context.Context, conn *models.Connection, task *models.CertTask, creds map[string]string, email string) error {
	connectionID := conn.ID
	taskID := task.ID
	fail := func(err error) error {
		m.stage(taskID, "", connectionID, StageIssue, "fail", err.Error())
		return err
	}

	plugin, err := PluginFor(task.DNSProvider, task.DNSPlugin)
	if err != nil {
		return fail(err)
	}

	m.stage(taskID, "", connectionID, StageDetect, "start", "")
	env, err := m.Detect(ctx, connectionID)
	if err != nil {
		return fail(err)
	}
	if !env.Installed {
		m.stage(taskID, "", connectionID, StageDetect, "ok", "")
		if email == "" {
			return fail(fmt.Errorf("acme.sh is not installed on %s and no account email was provided", conn.Name))
		}
		if err := m.InstallAcmeSh(ctx, conn, taskID, email); err != nil {
			return fail(err)
		}
	} else {
		m.stage(taskID, "", connectionID, StageDetect, "ok", "")
	}

	if len(creds) > 0 {
		content, err := BuildEnvFileContent(creds)
		if err != nil {
			return fail(err)
		}
		envFile := TempEnvPath()
		if err := m.sftp.WriteFileContent(connectionID, envFile, content); err != nil {
			return fail(fmt.Errorf("upload DNS credentials: %w", err))
		}
		// The file is chmod'ed to 600 and removed inside the issue command
		// itself (sourced then deleted; a trailing rm guards the failure
		// paths).
		m.stage(taskID, "", connectionID, StageIssue, "start", "")
		_, err = m.stream(ctx, connectionID, taskID, "", StageIssue, BuildIssueCmd(envFile, task, plugin), timeoutIssue)
		if err != nil {
			return fail(err)
		}
	} else {
		// Renewal-style re-issue: credentials already persisted by acme.sh
		// in account.conf, no temporary file needed.
		m.stage(taskID, "", connectionID, StageIssue, "start", "")
		if _, err := m.stream(ctx, connectionID, taskID, "", StageIssue, BuildIssueCmd("", task, plugin), timeoutIssue); err != nil {
			return fail(err)
		}
	}
	m.stage(taskID, "", connectionID, StageIssue, "ok", "")

	if task.AutoInstall {
		if task.CertDir == "" || task.KeyFile == "" || task.FullchainFile == "" {
			return fail(fmt.Errorf("auto install is enabled but cert dir/key file/fullchain file are not configured"))
		}
		m.stage(taskID, "", connectionID, StageInstallCert, "start", "")
		if _, err := m.stream(ctx, connectionID, taskID, "", StageInstallCert, BuildInstallCertCmd(task), timeoutInstallCert); err != nil {
			return fail(err)
		}
		m.stage(taskID, "", connectionID, StageInstallCert, "ok", "")
	}

	if err := m.ensureCronStage(ctx, connectionID, taskID); err != nil {
		return fail(err)
	}

	m.stage(taskID, "", connectionID, StageDone, "ok", "")
	return nil
}

// Renew forces an immediate renewal. acme.sh re-runs the saved install-cert
// and reload command automatically. Task credentials, when present, are
// pushed via a temporary env file so the renewal heals stale server-side
// credentials (a failed earlier issue may have persisted wrong ones).
func (m *Manager) Renew(ctx context.Context, conn *models.Connection, task *models.CertTask, creds map[string]string) error {
	connectionID := conn.ID
	taskID := task.ID
	fail := func(err error) error {
		m.stage(taskID, "", connectionID, StageRenew, "fail", err.Error())
		return err
	}

	env, err := m.Detect(ctx, connectionID)
	if err != nil {
		return fail(err)
	}
	if !env.Installed {
		return fail(fmt.Errorf("acme.sh is not installed on %s", conn.Name))
	}

	envFile := ""
	if len(creds) > 0 {
		content, buildErr := BuildEnvFileContent(creds)
		if buildErr != nil {
			return fail(buildErr)
		}
		envFile = TempEnvPath()
		if err := m.sftp.WriteFileContent(connectionID, envFile, content); err != nil {
			return fail(fmt.Errorf("upload DNS credentials: %w", err))
		}
	}

	m.stage(taskID, "", connectionID, StageRenew, "start", "")
	if _, err := m.stream(ctx, connectionID, taskID, "", StageRenew, BuildRenewCmd(envFile, task), timeoutIssue); err != nil {
		return fail(err)
	}
	m.stage(taskID, "", connectionID, StageRenew, "ok", "")

	if err := m.ensureCronStage(ctx, connectionID, taskID); err != nil {
		return fail(err)
	}
	m.stage(taskID, "", connectionID, StageDone, "ok", "")
	return nil
}

// Remove drops a domain from acme.sh's renewal list; files on disk are kept.
func (m *Manager) Remove(ctx context.Context, conn *models.Connection, domain string, ecc bool, taskID string) error {
	connectionID := conn.ID
	m.stage(taskID, "", connectionID, StageRemove, "start", "")
	if _, err := m.stream(ctx, connectionID, taskID, "", StageRemove, BuildRemoveCmd(domain, ecc), timeoutRemove); err != nil {
		m.stage(taskID, "", connectionID, StageRemove, "fail", err.Error())
		return err
	}
	m.stage(taskID, "", connectionID, StageRemove, "ok", "")
	m.stage(taskID, "", connectionID, StageDone, "ok", "")
	return nil
}

func (m *Manager) ensureCronStage(ctx context.Context, connectionID, taskID string) error {
	m.stage(taskID, "", connectionID, StageCron, "start", "")
	res, err := m.Run(ctx, connectionID, BuildEnsureCronCmd(), RunOptions{Timeout: seconds(timeoutQuick)})
	if err != nil {
		m.stage(taskID, "", connectionID, StageCron, "fail", err.Error())
		return wrapStageError(StageCron, res, err)
	}
	m.stage(taskID, "", connectionID, StageCron, "ok", "")
	return nil
}
