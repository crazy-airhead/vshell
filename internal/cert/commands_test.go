package cert

import (
	"strings"
	"testing"

	"vshell/internal/models"
)

func TestShQuote(t *testing.T) {
	cases := map[string]string{
		"simple":            "'simple'",
		"":                  "''",
		"it's":              `'it'\''s'`,
		"$(rm -rf /)":       `'$(rm -rf /)'`,
		"a\nb":              "'a\nb'",
		"*.example.com":     "'*.example.com'",
		"a; rm -f x && y|z": `'a; rm -f x && y|z'`,
	}
	for in, want := range cases {
		if got := shQuote(in); got != want {
			t.Errorf("shQuote(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestJoinRemotePath(t *testing.T) {
	cases := []struct{ dir, file, want string }{
		{"/etc/nginx/ssl", "example.com.key", "/etc/nginx/ssl/example.com.key"},
		{"/etc/nginx/ssl/", "example.com.key", "/etc/nginx/ssl/example.com.key"},
		{"/etc/nginx/ssl", "/opt/k.pem", "/opt/k.pem"},
		{"/etc/nginx/ssl", "", "/etc/nginx/ssl"},
		{"", "k.pem", "k.pem"},
	}
	for _, c := range cases {
		if got := joinRemotePath(c.dir, c.file); got != c.want {
			t.Errorf("joinRemotePath(%q,%q) = %q, want %q", c.dir, c.file, got, c.want)
		}
	}
}

func TestBuildDetectScript(t *testing.T) {
	script := BuildDetectScript()
	for _, marker := range []string{"vshell:home=$HOME", "vshell:curl=", "vshell:acmesh=", "vshell:cron="} {
		if !strings.Contains(script, marker) {
			t.Errorf("detect script missing marker %s", marker)
		}
	}
	if strings.Contains(script, "/root") {
		t.Error("detect script must not hardcode /root")
	}
}

func TestBuildIssueCmd(t *testing.T) {
	task := &models.CertTask{
		PrimaryDomain: "example.com",
		SANDomains:    []string{"*.example.com", "www.example.com"},
		KeyLength:     "ec-256",
		DNSSleep:      120,
	}
	cmd := BuildIssueCmd("/tmp/.vshell_acme_abc.env", task, "dns_cf")

	for _, want := range []string{
		`f='/tmp/.vshell_acme_abc.env'`,
		`chmod 600 "$f" && . "$f" && rm -f "$f"`,
		`--dns 'dns_cf'`,
		`-d 'example.com'`,
		`-d '*.example.com'`,
		`-d 'www.example.com'`,
		`--keylength 'ec-256'`,
		`--dnssleep 120`,
		`--server letsencrypt`,
		`--log`,
		`; rc=$?; rm -f "$f" 2>/dev/null; exit $rc`,
	} {
		if !strings.Contains(cmd, want) {
			t.Errorf("issue cmd missing %q:\n%s", want, cmd)
		}
	}
	if strings.Contains(cmd, "--save") {
		t.Error("issue cmd must not use --save (not an acme.sh flag)")
	}
	if strings.Contains(cmd, "/root") {
		t.Error("issue cmd must not hardcode /root")
	}
	if strings.Contains(cmd, "--test") {
		t.Error("issue cmd must not contain --test when TestMode is false")
	}

	task.TestMode = true
	if !strings.Contains(BuildIssueCmd("/tmp/x.env", task, "dns_ali"), "--test") {
		t.Error("issue cmd must contain --test when TestMode is true")
	}
}

func TestBuildInstallCertCmd(t *testing.T) {
	task := &models.CertTask{
		PrimaryDomain: "example.com",
		KeyLength:     "ec-256",
		CertDir:       "/etc/nginx/ssl",
		KeyFile:       "example.com.key",
		FullchainFile: "example.com.crt",
		ReloadCmd:     "systemctl reload nginx",
	}
	cmd := BuildInstallCertCmd(task)
	for _, want := range []string{
		`mkdir -p '/etc/nginx/ssl'`,
		`--install-cert`,
		`-d 'example.com'`,
		` --ecc`,
		`--key-file '/etc/nginx/ssl/example.com.key'`,
		`--fullchain-file '/etc/nginx/ssl/example.com.crt'`,
		`--reloadcmd 'systemctl reload nginx'`,
	} {
		if !strings.Contains(cmd, want) {
			t.Errorf("install-cert cmd missing %q:\n%s", want, cmd)
		}
	}

	task.KeyLength = "2048"
	task.ReloadCmd = ""
	cmd = BuildInstallCertCmd(task)
	if strings.Contains(cmd, "--ecc") {
		t.Error("RSA cert must not pass --ecc")
	}
	if strings.Contains(cmd, "--reloadcmd") {
		t.Error("empty reload cmd must omit --reloadcmd")
	}
}

func TestBuildRenewAndRemoveCmd(t *testing.T) {
	task := &models.CertTask{PrimaryDomain: "example.com", KeyLength: "ec-384"}
	renew := BuildRenewCmd("", task)
	for _, want := range []string{`--renew`, `-d 'example.com'`, `--force`, ` --ecc`, `--log`} {
		if !strings.Contains(renew, want) {
			t.Errorf("renew cmd missing %q: %s", want, renew)
		}
	}

	// With an env file, credentials are sourced and cleaned up like issue.
	renewEnv := BuildRenewCmd("/tmp/.vshell_acme_env", task)
	for _, want := range []string{
		`f='/tmp/.vshell_acme_env'`,
		`chmod 600 "$f" && . "$f" && rm -f "$f"`,
		`--renew -d 'example.com' --force --ecc --log`,
		`; rc=$?; rm -f "$f" 2>/dev/null; exit $rc`,
	} {
		if !strings.Contains(renewEnv, want) {
			t.Errorf("renew-with-env cmd missing %q:\n%s", want, renewEnv)
		}
	}

	remove := BuildRemoveCmd("example.com", true)
	if !strings.Contains(remove, `--remove -d 'example.com' --ecc`) {
		t.Errorf("unexpected remove cmd: %s", remove)
	}
	if strings.Contains(BuildRemoveCmd("example.com", false), "--ecc") {
		t.Error("RSA remove must not pass --ecc")
	}
}

func TestBuildInstallCmd(t *testing.T) {
	cmd := BuildInstallCmd("me@example.com")
	for _, want := range []string{
		`d=$(mktemp)`,
		`command -v curl`,
		`curl -fsSL https://get.acme.sh -o "$d"`,
		`command -v wget`,
		`wget -qO "$d" https://get.acme.sh`,
		`sh "$d" email='me@example.com'`,
		"neither curl nor wget found on server",
		`; rc=$?; rm -f "$d"; exit $rc`,
	} {
		if !strings.Contains(cmd, want) {
			t.Errorf("install cmd missing %q:\n%s", want, cmd)
		}
	}
	if strings.Contains(cmd, "| sh") {
		t.Error("install cmd must not pipe the download into sh (masks download failures)")
	}
}

func TestBuildEnsureCronCmd(t *testing.T) {
	cmd := BuildEnsureCronCmd()
	if !strings.Contains(cmd, "grep -q acme.sh") {
		t.Error("cron cmd must check before adding")
	}
	if !strings.Contains(cmd, `"$HOME/.acme.sh"/acme.sh --cron --home "$HOME/.acme.sh"`) {
		t.Error("cron cmd must reference $HOME paths")
	}
	if strings.Contains(cmd, "/root") {
		t.Error("cron cmd must not hardcode /root")
	}
}

func TestBuildTruncateAndTailLogCmd(t *testing.T) {
	if cmd := BuildTruncateLogCmd(); cmd != `: > "$HOME/.acme.sh/acme.sh.log"` {
		t.Errorf("unexpected truncate cmd: %s", cmd)
	}
	if tail := BuildTailLogCmd(); !strings.Contains(tail, `tail -n 200 "$HOME/.acme.sh/acme.sh.log"`) {
		t.Errorf("unexpected tail cmd: %s", tail)
	}
}
