package cert

import (
	"strings"
	"testing"
)

func TestPluginFor(t *testing.T) {
	if p, err := PluginFor("aliyun", ""); err != nil || p != "dns_ali" {
		t.Errorf("aliyun → %q, err=%v", p, err)
	}
	if p, err := PluginFor("cloudflare", ""); err != nil || p != "dns_cf" {
		t.Errorf("cloudflare → %q, err=%v", p, err)
	}
	if p, err := PluginFor("custom", "dns_he"); err != nil || p != "dns_he" {
		t.Errorf("custom dns_he → %q, err=%v", p, err)
	}
	for _, bad := range []string{"", "dns_cf; rm -rf /", "dns-xx", "DNS_CF", "curl"} {
		if _, err := PluginFor("custom", bad); err == nil {
			t.Errorf("custom plugin %q should be rejected", bad)
		}
	}
}

func TestValidateCredentials(t *testing.T) {
	if err := ValidateCredentials("aliyun", map[string]string{"Ali_Key": "k", "Ali_Secret": "s"}); err != nil {
		t.Errorf("valid creds rejected: %v", err)
	}
	if err := ValidateCredentials("aliyun", map[string]string{"Ali_Key": "k"}); err == nil {
		t.Error("missing required Ali_Secret must fail")
	}
	if err := ValidateCredentials("nope", nil); err == nil {
		t.Error("unknown provider must fail")
	}
	if err := ValidateCredentials("cloudflare", map[string]string{"A=B\nC": "x", "CF_Token": "t"}); err == nil {
		t.Error("illegal env key must fail")
	}
	// Optional fields may stay empty.
	if err := ValidateCredentials("cloudflare", map[string]string{"CF_Token": "t"}); err != nil {
		t.Errorf("optional fields empty should pass: %v", err)
	}
}

func TestBuildEnvFileContent(t *testing.T) {
	content, err := BuildEnvFileContent(map[string]string{
		"Ali_Secret": "s3cr't",
		"Ali_Key":    "key",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 export lines, got %d: %q", len(lines), content)
	}
	if lines[0] != "export Ali_Key='key'" {
		t.Errorf("sorted first line = %q", lines[0])
	}
	if lines[1] != `export Ali_Secret='s3cr'\''t'` {
		t.Errorf("quoted second line = %q", lines[1])
	}

	if _, err := BuildEnvFileContent(nil); err == nil {
		t.Error("empty creds must fail")
	}
	if _, err := BuildEnvFileContent(map[string]string{"bad key": "v"}); err == nil {
		t.Error("illegal key must fail")
	}
}

func TestTempEnvPath(t *testing.T) {
	p1, p2 := TempEnvPath(), TempEnvPath()
	if !strings.HasPrefix(p1, "/tmp/.vshell_acme_") || !strings.HasSuffix(p1, ".env") {
		t.Errorf("unexpected path %q", p1)
	}
	if p1 == p2 {
		t.Error("paths must be random")
	}
}

func TestGetProvider(t *testing.T) {
	if p, ok := GetProvider("dnspod"); !ok || p.Plugin != "dns_dp" || len(p.Fields) != 2 {
		t.Errorf("dnspod provider: %+v ok=%v", p, ok)
	}
	if _, ok := GetProvider("nope"); ok {
		t.Error("unknown provider should not be found")
	}
}
