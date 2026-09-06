package cert

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"vshell/internal/models"

	"github.com/google/uuid"
)

// Providers is the built-in DNS provider registry. It maps a friendly id to
// the acme.sh dnsapi plugin plus the credential env vars that plugin reads.
// Keep in sync with https://github.com/acmesh-official/acme.sh/wiki/dnsapi.
var Providers = []models.DNSProvider{
	{
		ID:     "aliyun",
		Name:   "Aliyun DNS",
		Plugin: "dns_ali",
		Fields: []models.DNSFieldSpec{
			{Key: "Ali_Key", Label: "AccessKey ID", Required: true},
			{Key: "Ali_Secret", Label: "AccessKey Secret", Required: true, Secret: true},
		},
	},
	{
		ID:     "dnspod",
		Name:   "DNSPod (Tencent)",
		Plugin: "dns_dp",
		Fields: []models.DNSFieldSpec{
			{Key: "DP_Id", Label: "API ID", Required: true},
			{Key: "DP_Key", Label: "API Key", Required: true, Secret: true},
		},
	},
	{
		ID:     "cloudflare",
		Name:   "Cloudflare",
		Plugin: "dns_cf",
		Fields: []models.DNSFieldSpec{
			{Key: "CF_Token", Label: "API Token", Required: true, Secret: true, Placeholder: "Zone:DNS:Edit + Zone:Read"},
			{Key: "CF_Account_ID", Label: "Account ID (optional)"},
			{Key: "CF_Zone_ID", Label: "Zone ID (optional)"},
		},
	},
	{
		ID:     "custom",
		Name:   "Custom",
		Plugin: "",
	},
}

// GetProvider looks a provider up by id.
func GetProvider(id string) (models.DNSProvider, bool) {
	for _, p := range Providers {
		if p.ID == id {
			return p, true
		}
	}
	return models.DNSProvider{}, false
}

var (
	envKeyRe   = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	pluginName = regexp.MustCompile(`^dns_[a-z0-9_]+$`)
)

// PluginFor resolves the acme.sh plugin name for a provider id. Custom
// providers take the plugin name from user input and must match dns_xx.
func PluginFor(providerID, customPlugin string) (string, error) {
	if p, ok := GetProvider(providerID); ok && p.ID != "custom" {
		return p.Plugin, nil
	}
	if !pluginName.MatchString(customPlugin) {
		return "", fmt.Errorf("invalid acme.sh DNS plugin name %q (expected dns_<name>)", customPlugin)
	}
	return customPlugin, nil
}

// ValidateCredentials checks required fields are filled and every env key is
// a legal shell identifier (guards the credential file against injection).
func ValidateCredentials(providerID string, creds map[string]string) error {
	p, ok := GetProvider(providerID)
	if !ok {
		return fmt.Errorf("unknown DNS provider %q", providerID)
	}
	for key := range creds {
		if !envKeyRe.MatchString(key) {
			return fmt.Errorf("invalid credential key %q", key)
		}
	}
	for _, f := range p.Fields {
		if f.Required && strings.TrimSpace(creds[f.Key]) == "" {
			return fmt.Errorf("DNS credential %s is required", f.Key)
		}
	}
	return nil
}

// BuildEnvFileContent renders credentials as an `export KEY='value'` shell
// file with sorted keys (deterministic) and safely quoted values.
func BuildEnvFileContent(creds map[string]string) (string, error) {
	if len(creds) == 0 {
		return "", fmt.Errorf("no DNS credentials provided")
	}
	keys := make([]string, 0, len(creds))
	for k := range creds {
		if !envKeyRe.MatchString(k) {
			return "", fmt.Errorf("invalid credential key %q", k)
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "export %s=%s\n", k, shQuote(creds[k]))
	}
	return b.String(), nil
}

// TempEnvPath returns a random /tmp path for the credential file so it
// cannot be guessed and does not collide between concurrent tasks.
func TempEnvPath() string {
	return "/tmp/.vshell_acme_" + strings.ReplaceAll(uuid.New().String(), "-", "") + ".env"
}
