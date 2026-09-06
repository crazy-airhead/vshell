package cert

import (
	"strconv"
	"strings"
	"time"

	"vshell/internal/models"
)

// ParseListOutput parses `acme.sh --list` output into RemoteCerts. The table
// format has drifted between acme.sh versions (pipe-separated vs dynamically
// padded whitespace), so both layouts are supported and any unrecognised
// output yields an empty slice rather than an error.
func ParseListOutput(output string) []models.RemoteCert {
	lines := strings.Split(output, "\n")
	headerIdx := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "Main_Domain") &&
			strings.Contains(trimmed, "KeyLength") &&
			strings.Contains(trimmed, "Created") &&
			strings.Contains(trimmed, "Renew") {
			headerIdx = i
			break
		}
	}
	if headerIdx < 0 {
		return nil
	}

	var certs []models.RemoteCert
	if strings.Contains(lines[headerIdx], "|") {
		cols := splitPipeRow(lines[headerIdx])
		for _, line := range lines[headerIdx+1:] {
			if strings.TrimSpace(line) == "" {
				continue
			}
			values := splitPipeRow(line)
			cert, ok := buildRemoteCert(cols, values)
			if ok {
				certs = append(certs, cert)
			}
		}
		return certs
	}

	// Whitespace-padded layout: fields never contain spaces (SAN domains are
	// comma-joined, timestamps ISO), so positional token parsing works:
	// domain keylength san [ca] created renew — the CA column may be absent.
	for _, line := range lines[headerIdx+1:] {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		var cert models.RemoteCert
		cert.MainDomain = fields[0]
		cert.KeyLength = strings.Trim(fields[1], `"`)
		cert.SANDomains = splitSAN(fields[2])
		created, renew := fields[len(fields)-2], fields[len(fields)-1]
		if len(fields) >= 6 {
			cert.CA = strings.Join(fields[3:len(fields)-2], " ")
		}
		cert.Created, cert.Renew = created, renew
		finalizeRemoteCert(&cert)
		certs = append(certs, cert)
	}
	return certs
}

// splitPipeRow splits a "a | b | c" row and trims each cell.
func splitPipeRow(line string) []string {
	parts := strings.Split(line, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// buildRemoteCert maps pipe-layout values onto a RemoteCert using the header
// column names as the source of truth.
func buildRemoteCert(cols, values []string) (models.RemoteCert, bool) {
	if len(values) < 4 {
		return models.RemoteCert{}, false
	}
	get := func(name string) string {
		for i, c := range cols {
			if c == name && i < len(values) {
				return values[i]
			}
		}
		return ""
	}
	var cert models.RemoteCert
	cert.MainDomain = get("Main_Domain")
	if cert.MainDomain == "" {
		return models.RemoteCert{}, false
	}
	cert.KeyLength = strings.Trim(get("KeyLength"), `"`)
	cert.SANDomains = splitSAN(get("SAN_Domains"))
	cert.CA = get("CA")
	cert.Created = get("Created")
	cert.Renew = get("Renew")
	finalizeRemoteCert(&cert)
	return cert, true
}

func splitSAN(san string) []string {
	if san == "" || san == "no" {
		return nil
	}
	var domains []string
	for _, d := range strings.Split(san, ",") {
		if d = strings.TrimSpace(d); d != "" {
			domains = append(domains, d)
		}
	}
	return domains
}

func finalizeRemoteCert(cert *models.RemoteCert) {
	cert.ECC = isECC(cert.KeyLength)
}

// ParseInfoOutput parses `acme.sh --info` key=value lines into a
// RemoteCertInfo. Values may be single- or double-quoted.
func ParseInfoOutput(output string) *models.RemoteCertInfo {
	info := &models.RemoteCertInfo{Fields: map[string]string{}}
	for _, line := range strings.Split(output, "\n") {
		idx := strings.Index(line, "=")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		value := unquote(strings.TrimSpace(line[idx+1:]))
		info.Fields[key] = value
		switch key {
		case "Le_Domain":
			info.Domain = value
		case "Le_NextRenewTime":
			if n, err := strconv.ParseInt(value, 10, 64); err == nil {
				info.NextRenewTime = n
			}
		case "Le_NextRenewTimeStr":
			info.NextRenewTimeStr = value
		case "Le_CertCreateTimeStr":
			info.CertCreateTime = value
		case "Le_RealFullChainPath":
			info.CertPath = value
		}
	}
	if len(info.Fields) == 0 {
		return nil
	}
	return info
}

// ParseDetectOutput extracts "vshell:key=value" marker lines, ignoring any
// shell noise mixed in.
func ParseDetectOutput(output string) models.CertEnvironment {
	env := models.CertEnvironment{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "vshell:") {
			continue
		}
		kv := strings.SplitN(strings.TrimPrefix(line, "vshell:"), "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "home":
			env.Home = kv[1]
			env.AcmeShPath = kv[1] + "/.acme.sh/acme.sh"
		case "curl":
			env.CurlPresent = kv[1] == "present"
		case "acmesh":
			env.Installed = kv[1] == "installed"
		case "cron":
			env.CronPresent = kv[1] == "present"
		}
	}
	return env
}

// ParseAcmeDirsOutput parses `ls -1d ~/.acme.sh/*/` output for the fallback
// domain discovery: "<domain>_ecc/" entries are ECC certs, plain ones RSA.
func ParseAcmeDirsOutput(output string) []models.RemoteCert {
	var certs []models.RemoteCert
	seen := map[string]int{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = strings.TrimSuffix(line, "/")
		if line == "" {
			continue
		}
		parts := strings.Split(line, "/")
		dir := parts[len(parts)-1]
		cert := models.RemoteCert{}
		if strings.HasSuffix(dir, "_ecc") {
			cert.MainDomain = strings.TrimSuffix(dir, "_ecc")
			cert.ECC = true
			cert.KeyLength = "ec-256"
		} else {
			cert.MainDomain = dir
			cert.ECC = false
		}
		if cert.MainDomain == "" {
			continue
		}
		// Prefer the ECC variant when both layouts exist for one domain.
		if idx, ok := seen[cert.MainDomain]; ok {
			if cert.ECC {
				certs[idx] = cert
			}
			continue
		}
		seen[cert.MainDomain] = len(certs)
		certs = append(certs, cert)
	}
	return certs
}

// DaysLeft computes whole days until t; negative means already expired.
func DaysLeft(unixSeconds int64, now time.Time) *int {
	if unixSeconds <= 0 {
		return nil
	}
	days := int(time.Unix(unixSeconds, 0).Sub(now).Hours() / 24)
	return &days
}

func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '"' && s[len(s)-1] == '"') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
