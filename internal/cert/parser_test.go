package cert

import (
	"testing"
	"time"
)

const listPiped = `Main_Domain    | KeyLength | SAN_Domains    | CA              | Created               | Renew
example.com    | ec-256    | *.example.com | letsencrypt.org | 2026-09-06T00:00:00Z | 2026-12-05T00:00:00Z
b.example.com  | "2048"    |                | letsencrypt.org | 2026-08-01T00:00:00Z | 2026-10-31T00:00:00Z`

const listPipedNoCA = `Main_Domain   | KeyLength | SAN_Domains   | Created  | Renew
example.com   | ec-256    | no           | 2026-09-06 | 2026-12-05`

const listSpaced = `Main_Domain  KeyLength  SAN_Domains  CA  Created  Renew
example.com  "ec-256"  www.example.com,*.example.com  letsencrypt  2026-09-06T00:00:00Z  2026-12-05T00:00:00Z
b.example.com  2048  no  letsencrypt  2026-08-01T00:00:00Z  2026-10-31T00:00:00Z`

func TestParseListOutputPiped(t *testing.T) {
	certs, ok := ParseListOutput(listPiped)
	if !ok {
		t.Fatal("piped table should be recognized")
	}
	if len(certs) != 2 {
		t.Fatalf("want 2 certs, got %d: %+v", len(certs), certs)
	}
	c := certs[0]
	if c.MainDomain != "example.com" || c.KeyLength != "ec-256" || !c.ECC {
		t.Errorf("unexpected first cert: %+v", c)
	}
	if len(c.SANDomains) != 1 || c.SANDomains[0] != "*.example.com" {
		t.Errorf("unexpected SAN: %+v", c.SANDomains)
	}
	if c.CA != "letsencrypt.org" {
		t.Errorf("CA = %q", c.CA)
	}
	if c.Created != "2026-09-06T00:00:00Z" || c.Renew != "2026-12-05T00:00:00Z" {
		t.Errorf("dates: %q / %q", c.Created, c.Renew)
	}
	if certs[1].ECC {
		t.Error("RSA cert must not be ECC")
	}
}

func TestParseListOutputPipedNoCAColumn(t *testing.T) {
	certs, ok := ParseListOutput(listPipedNoCA)
	if !ok {
		t.Fatal("table should be recognized")
	}
	if len(certs) != 1 {
		t.Fatalf("want 1 cert, got %d", len(certs))
	}
	c := certs[0]
	if c.MainDomain != "example.com" || c.CA != "" {
		t.Errorf("unexpected cert: %+v", c)
	}
	if c.SANDomains != nil {
		t.Errorf("SAN 'no' must parse to nil, got %+v", c.SANDomains)
	}
}

func TestParseListOutputSpaced(t *testing.T) {
	certs, ok := ParseListOutput(listSpaced)
	if !ok {
		t.Fatal("table should be recognized")
	}
	if len(certs) != 2 {
		t.Fatalf("want 2 certs, got %d: %+v", len(certs), certs)
	}
	c := certs[0]
	if c.MainDomain != "example.com" || c.KeyLength != "ec-256" {
		t.Errorf("unexpected first cert: %+v", c)
	}
	if len(c.SANDomains) != 2 || c.SANDomains[0] != "www.example.com" || c.SANDomains[1] != "*.example.com" {
		t.Errorf("unexpected SAN: %+v", c.SANDomains)
	}
	if c.Renew != "2026-12-05T00:00:00Z" {
		t.Errorf("Renew = %q", c.Renew)
	}
}

// Tab-separated layout of acme.sh 3.x (verified against a real install,
// 2026-09): note the extra Profile column between SAN_Domains and CA.
const listTabbedReal = "Main_Domain\tKeyLength\tSAN_Domains\tProfile\tCA\tCreated\tRenew\n" +
	"example.com\tec-256\t*.example.com\tprod\tletsencrypt\t2026-09-06\t2026-12-05\n" +
	"b.example.com\t2048\tno\tprod\tletsencrypt\t2026-08-01\t2026-10-31\n"

func TestParseListOutputTabbedRealFormat(t *testing.T) {
	certs, ok := ParseListOutput(listTabbedReal)
	if !ok {
		t.Fatal("table should be recognized")
	}
	if len(certs) != 2 {
		t.Fatalf("want 2 certs, got %d: %+v", len(certs), certs)
	}
	c := certs[0]
	if c.MainDomain != "example.com" || c.KeyLength != "ec-256" || !c.ECC {
		t.Errorf("unexpected first cert: %+v", c)
	}
	if len(c.SANDomains) != 1 || c.SANDomains[0] != "*.example.com" {
		t.Errorf("unexpected SAN: %+v", c.SANDomains)
	}
	if c.Created != "2026-09-06" || c.Renew != "2026-12-05" {
		t.Errorf("dates: %q / %q", c.Created, c.Renew)
	}
	// SAN_Domains "no" must not become a domain entry.
	if certs[1].SANDomains != nil {
		t.Errorf("SAN 'no' must parse to nil, got %+v", certs[1].SANDomains)
	}
}

func TestParseListOutputEmptyOrUnknown(t *testing.T) {
	for _, out := range []string{"", "\n", "No cert found.\n", "random shell noise\n"} {
		if certs, ok := ParseListOutput(out); len(certs) != 0 || ok {
			t.Errorf("ParseListOutput(%q) = %+v ok=%v, want empty and unrecognized", out, certs, ok)
		}
	}
}

// A recognized header with zero rows means "no certs" (e.g. after a failed
// issuance the domain conf exists but the cert is not in the renewal list).
func TestParseListOutputHeaderOnlyIsRecognized(t *testing.T) {
	certs, ok := ParseListOutput(listTabbedReal[:len(listTabbedReal)-len("example.com\tec-256\t*.example.com\tprod\tletsencrypt\t2026-09-06\t2026-12-05\nb.example.com\t2048\tno\tprod\tletsencrypt\t2026-08-01\t2026-10-31\n")])
	if !ok {
		t.Fatal("header-only table should be recognized")
	}
	if len(certs) != 0 {
		t.Errorf("want zero certs, got %+v", certs)
	}
}

const infoOut = `DOMAIN_CONF=/home/ubuntu/.acme.sh/example.com_ecc/example.com.conf
Le_Domain=example.com
Le_API=https://acme-v02.api.letsencrypt.org/directory
Le_NextRenewTime='1793644800'
Le_NextRenewTimeStr='Sat Dec  5 00:00:00 UTC 2026'
Le_CertCreateTimeStr='Sat Sep 6 00:00:00 UTC 2026'
Le_RealFullChainPath='/etc/nginx/ssl/example.com.crt'`

func TestParseInfoOutput(t *testing.T) {
	info := ParseInfoOutput(infoOut)
	if info == nil {
		t.Fatal("want info, got nil")
	}
	if info.Domain != "example.com" {
		t.Errorf("Domain = %q", info.Domain)
	}
	if info.NextRenewTime != 1793644800 {
		t.Errorf("NextRenewTime = %d", info.NextRenewTime)
	}
	if info.NextRenewTimeStr != "Sat Dec  5 00:00:00 UTC 2026" {
		t.Errorf("NextRenewTimeStr = %q", info.NextRenewTimeStr)
	}
	if info.CertPath != "/etc/nginx/ssl/example.com.crt" {
		t.Errorf("CertPath = %q", info.CertPath)
	}
	if info.Fields["Le_API"] != "https://acme-v02.api.letsencrypt.org/directory" {
		t.Errorf("Le_API = %q", info.Fields["Le_API"])
	}
}

func TestParseInfoOutputEmpty(t *testing.T) {
	if info := ParseInfoOutput("nothing here\nno equals"); info != nil {
		t.Errorf("want nil for non key=value output, got %+v", info)
	}
}

const detectOut = `Last login: ...
vshell:home=/home/ubuntu
vshell:curl=present
some noise
vshell:acmesh=installed
vshell:cron=missing`

func TestParseDetectOutput(t *testing.T) {
	env := ParseDetectOutput(detectOut)
	if env.Home != "/home/ubuntu" {
		t.Errorf("Home = %q", env.Home)
	}
	if !env.CurlPresent || !env.Installed {
		t.Errorf("curl/acmesh should be present: %+v", env)
	}
	if env.CronPresent {
		t.Error("cron should be missing")
	}
	if env.AcmeShPath != "/home/ubuntu/.acme.sh/acme.sh" {
		t.Errorf("AcmeShPath = %q", env.AcmeShPath)
	}
}

const acmeDirs = `/root/.acme.sh/ca/
/root/.acme.sh/deploy/
/root/.acme.sh/dnsapi/
/root/.acme.sh/notify/
/root/.acme.sh/example.com/
/root/.acme.sh/example.com_ecc/
/root/.acme.sh/b.example.com/
`

func TestParseAcmeDirsOutput(t *testing.T) {
	certs := ParseAcmeDirsOutput(acmeDirs)
	if len(certs) != 2 {
		t.Fatalf("want 2 certs (internal dirs filtered, ECC wins over RSA), got %+v", certs)
	}
	if certs[0].MainDomain != "example.com" || !certs[0].ECC {
		t.Errorf("first cert should be ECC example.com: %+v", certs[0])
	}
	if certs[1].MainDomain != "b.example.com" || certs[1].ECC {
		t.Errorf("second cert should be RSA b.example.com: %+v", certs[1])
	}
}

func TestDaysLeft(t *testing.T) {
	now := time.Unix(1700000000, 0)
	if d := DaysLeft(0, now); d != nil {
		t.Error("zero timestamp must yield nil")
	}
	d := DaysLeft(1700000000+3*86400+3600, now)
	if d == nil || *d != 3 {
		t.Errorf("want 3 days, got %v", d)
	}
	expired := DaysLeft(1700000000-86400, now)
	if expired == nil || *expired != -1 {
		t.Errorf("want -1 days, got %v", expired)
	}
}
