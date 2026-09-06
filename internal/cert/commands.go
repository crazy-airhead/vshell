package cert

import (
	"fmt"
	"strings"

	"vshell/internal/models"
)

// acmeSh is the absolute acme.sh path resolved via the login user's $HOME.
// Non-interactive shells have no acme.sh alias, and the install dir differs
// between root and regular users, so never hardcode /root.
const acmeSh = `"$HOME"/.acme.sh/acme.sh`

// shQuote wraps s in single quotes using the POSIX '\'' escape so arbitrary
// user input (domains, paths, reload commands) is never interpreted by the
// remote shell.
func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// joinRemotePath joins a directory and a file name; an absolute file name
// wins so users may enter full paths in the filename fields.
func joinRemotePath(dir, file string) string {
	if file == "" {
		return dir
	}
	if strings.HasPrefix(file, "/") || dir == "" {
		return file
	}
	return strings.TrimSuffix(dir, "/") + "/" + file
}

// BuildDetectScript gathers environment facts in one round trip. It prints
// machine-readable "vshell:key=value" lines parsed by ParseDetectOutput.
func BuildDetectScript() string {
	return `echo "vshell:home=$HOME"
command -v curl >/dev/null 2>&1 && echo "vshell:curl=present" || echo "vshell:curl=missing"
[ -x "$HOME/.acme.sh/acme.sh" ] && echo "vshell:acmesh=installed" || echo "vshell:acmesh=missing"
crontab -l 2>/dev/null | grep -q acme.sh && echo "vshell:cron=present" || echo "vshell:cron=missing"`
}

// BuildInstallCmd installs acme.sh via the official installer. The download
// goes to a temp file instead of a `curl | sh` pipe: when curl/wget fail
// (missing binary, network error) the pipe still exits 0 because sh reads an
// empty stream, faking a successful install. The get.acme.sh bootstrap
// itself also exits 0 on some failures, which the install-then-re-detect
// step in InstallAcmeSh guards against.
func BuildInstallCmd(email string) string {
	return fmt.Sprintf(`d=$(mktemp) || exit 1; `+
		`if command -v curl >/dev/null 2>&1; then curl -fsSL https://get.acme.sh -o "$d"; `+
		`elif command -v wget >/dev/null 2>&1; then wget -qO "$d" https://get.acme.sh; `+
		`else echo "vshell:install-error=neither curl nor wget found on server" >&2; rm -f "$d"; exit 127; fi && `+
		`sh "$d" email=%s; rc=$?; rm -f "$d"; exit $rc`, shQuote(email))
}

// BuildSetDefaultCACmd switches the default CA to Let's Encrypt (acme.sh
// defaults to ZeroSSL nowadays).
func BuildSetDefaultCACmd() string {
	return acmeSh + " --set-default-ca --server letsencrypt"
}

// BuildIssueCmd sources the temporary credential file, removes it right
// away, then runs the issuance. Credentials live only in shell environment
// variables for the duration of the command and never appear on the command
// line. acme.sh persists them to account.conf on first DNS API use (there is
// no --save flag), so later cron renewals work unattended. The trailing
// rc=$?/rm/exit chain guarantees cleanup and exit-code fidelity even when
// sourcing or chmod fails. An empty envFile skips the credential part
// entirely (relying on credentials already stored in account.conf). With
// force the command carries --force: manual runs must re-apply even when a
// cert is not yet due — and, critically, --issue (unlike --renew) re-applies
// the given --dns plugin, overwriting the provider recorded in the domain
// conf at first-issue time.
func BuildIssueCmd(envFile string, t *models.CertTask, plugin string, force bool) string {
	args := buildIssueArgs(t, plugin, force)
	if envFile == "" {
		return acmeSh + " " + args
	}
	return fmt.Sprintf("f=%s; chmod 600 \"$f\" && . \"$f\" && rm -f \"$f\" && %s %s; "+
		"rc=$?; rm -f \"$f\" 2>/dev/null; exit $rc", shQuote(envFile), acmeSh, args)
}

func buildIssueArgs(t *models.CertTask, plugin string, force bool) string {
	var b strings.Builder
	b.WriteString("--issue")
	fmt.Fprintf(&b, " --dns %s", shQuote(plugin))
	fmt.Fprintf(&b, " -d %s", shQuote(t.PrimaryDomain))
	for _, d := range t.SANDomains {
		fmt.Fprintf(&b, " -d %s", shQuote(d))
	}
	fmt.Fprintf(&b, " --keylength %s", shQuote(t.KeyLength))
	fmt.Fprintf(&b, " --dnssleep %d", t.DNSSleep)
	b.WriteString(" --server letsencrypt --log")
	if force {
		b.WriteString(" --force")
	}
	if t.TestMode {
		b.WriteString(" --test")
	}
	return b.String()
}

// BuildInstallCertCmd copies key/fullchain into the target directory and
// registers the reload command; acme.sh re-runs it automatically on every
// renewal. ECC certs need --ecc to select the <domain>_ecc directory.
func BuildInstallCertCmd(t *models.CertTask) string {
	var b strings.Builder
	fmt.Fprintf(&b, "mkdir -p %s && ", shQuote(t.CertDir))
	b.WriteString(acmeSh)
	b.WriteString(" --install-cert")
	fmt.Fprintf(&b, " -d %s", shQuote(t.PrimaryDomain))
	if isECC(t.KeyLength) {
		b.WriteString(" --ecc")
	}
	fmt.Fprintf(&b, " --key-file %s", shQuote(joinRemotePath(t.CertDir, t.KeyFile)))
	fmt.Fprintf(&b, " --fullchain-file %s", shQuote(joinRemotePath(t.CertDir, t.FullchainFile)))
	if t.ReloadCmd != "" {
		fmt.Fprintf(&b, " --reloadcmd %s", shQuote(t.ReloadCmd))
	}
	return b.String()
}

// BuildRenewCmd was removed: `acme.sh --renew` replays the DNS plugin saved
// in the domain conf at first-issue time (Le_Webroot), so changing the
// provider in vShell never took effect on already-registered domains.
// Renewals now go through BuildIssueCmd with force=true, which re-applies
// the task's current plugin and credentials and rewrites the conf.

// BuildRemoveCmd drops a domain from acme.sh's renewal list. Certificate
// files on disk (both ~/.acme.sh and installed copies) are kept.
func BuildRemoveCmd(domain string, ecc bool) string {
	cmd := acmeSh + " --remove -d " + shQuote(domain)
	if ecc {
		cmd += " --ecc"
	}
	return cmd
}

// BuildListCmd lists all certs managed by acme.sh on the server.
func BuildListCmd() string {
	return acmeSh + " --list"
}

// BuildInfoCmd prints the domain.conf key=value pairs for one cert.
func BuildInfoCmd(domain string, ecc bool) string {
	cmd := acmeSh + " --info -d " + shQuote(domain)
	if ecc {
		cmd += " --ecc"
	}
	return cmd
}

// BuildListDomainsCmd enumerates acme.sh cert directories as a fallback for
// --list format drift: "<domain>_ecc" dirs are ECC certs, plain ones RSA.
func BuildListDomainsCmd() string {
	return `ls -1d "$HOME"/.acme.sh/*/ 2>/dev/null || true`
}

// BuildEnsureCronCmd adds the daily acme.sh cron entry only when missing.
// The installer normally registers it already; this is a safety net.
func BuildEnsureCronCmd() string {
	return `(crontab -l 2>/dev/null | grep -q acme.sh) || ` +
		`(crontab -l 2>/dev/null; echo '0 0 * * * "$HOME/.acme.sh"/acme.sh --cron --home "$HOME/.acme.sh" > /dev/null') | crontab -`
}

// BuildTailLogCmd reads the tail of the persistent acme.sh log (written
// because issue/renew commands run with --log).
func BuildTailLogCmd() string {
	return `tail -n 200 "$HOME/.acme.sh/acme.sh.log" 2>/dev/null || echo "vshell:log=none"`
}

// BuildTruncateLogCmd clears the persistent acme.sh log so each vShell
// operation starts from a fresh log (the file accumulates across runs,
// which makes the log view show mostly stale history). Truncating (not
// deleting) keeps the file in place for acme.sh to append to.
func BuildTruncateLogCmd() string {
	return `: > "$HOME/.acme.sh/acme.sh.log"`
}

func isECC(keyLength string) bool {
	return strings.HasPrefix(keyLength, "ec-")
}
