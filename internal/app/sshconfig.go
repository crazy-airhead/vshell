package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"vshell/internal/models"
)

// SSHConfigDirective represents a single key-value directive within an SSH config block.
type SSHConfigDirective struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// SSHConfigEntry represents a single Host or Match block from ~/.ssh/config.
type SSHConfigEntry struct {
	Type       string               `json:"type"`
	Pattern    string               `json:"pattern"`
	Directives []SSHConfigDirective `json:"directives"`
}

// ReadSSHConfig parses ~/.ssh/config and returns structured entries.
func (a *AppService) ReadSSHConfig() ([]SSHConfigEntry, error) {
	dir, err := sshDir()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filepath.Join(dir, "config"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read ssh config: %w", err)
	}

	return parseSSHConfig(string(data)), nil
}

// WriteSSHConfig serializes entries back to ~/.ssh/config.
func (a *AppService) WriteSSHConfig(entries []SSHConfigEntry) error {
	dir, err := sshDir()
	if err != nil {
		return err
	}
	os.MkdirAll(dir, 0700)

	var sb strings.Builder
	for i, entry := range entries {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(entry.Type)
		sb.WriteString(" ")
		sb.WriteString(entry.Pattern)
		sb.WriteString("\n")
		for _, d := range entry.Directives {
			sb.WriteString("    ")
			sb.WriteString(d.Key)
			sb.WriteString(" ")
			sb.WriteString(d.Value)
			sb.WriteString("\n")
		}
	}
	if sb.Len() > 0 {
		sb.WriteString("\n")
	}

	return os.WriteFile(filepath.Join(dir, "config"), []byte(sb.String()), 0600)
}

// ReadSSHConfigRaw returns the raw content of ~/.ssh/config.
func (a *AppService) ReadSSHConfigRaw() (string, error) {
	dir, err := sshDir()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(filepath.Join(dir, "config"))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read ssh config: %w", err)
	}
	return string(data), nil
}

// WriteSSHConfigRaw writes raw content to ~/.ssh/config.
func (a *AppService) WriteSSHConfigRaw(content string) error {
	dir, err := sshDir()
	if err != nil {
		return err
	}
	os.MkdirAll(dir, 0700)

	return os.WriteFile(filepath.Join(dir, "config"), []byte(content), 0600)
}

func parseSSHConfig(content string) []SSHConfigEntry {
	var entries []SSHConfigEntry
	var current *SSHConfigEntry

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimRight(line, " \t\r")
		if trimmed == "" {
			continue
		}

		if strings.HasPrefix(trimmed, "#") {
			if current != nil {
				current.Directives = append(current.Directives, SSHConfigDirective{
					Key:   "#",
					Value: trimmed[1:],
				})
			}
			continue
		}

		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			// Top-level keyword
			keyword, value, ok := splitKV(trimmed)
			if !ok {
				continue
			}
			upper := strings.ToUpper(keyword)
			if upper == "HOST" || upper == "MATCH" {
				if current != nil {
					entries = append(entries, *current)
				}
				current = &SSHConfigEntry{
					Type:    upper,
					Pattern: value,
				}
			}
			continue
		}

		// Indented directive
		if current == nil {
			continue
		}
		keyword, value, ok := splitKV(strings.TrimSpace(trimmed))
		if !ok {
			continue
		}
		current.Directives = append(current.Directives, SSHConfigDirective{
			Key:   keyword,
			Value: value,
		})
	}

	if current != nil {
		entries = append(entries, *current)
	}
	return entries
}

func splitKV(line string) (string, string, bool) {
	idx := strings.IndexAny(line, " \t")
	if idx < 0 {
		return line, "", true
	}
	return line[:idx], strings.TrimSpace(line[idx+1:]), true
}

// SSHConfigImportCandidate represents a Host entry from ~/.ssh/config that can be imported as a vShell connection.
type SSHConfigImportCandidate struct {
	Pattern      string `json:"pattern"`
	HostName     string `json:"hostname"`
	Port         int    `json:"port"`
	User         string `json:"user"`
	IdentityFile string `json:"identity_file"`
	HasKey       bool   `json:"has_key"`
}

// GetSSHConfigImportCandidates parses ~/.ssh/config and returns Host blocks as importable connection candidates.
func (a *AppService) GetSSHConfigImportCandidates() ([]SSHConfigImportCandidate, error) {
	entries, err := a.ReadSSHConfig()
	if err != nil {
		return nil, err
	}

	sshDir, err := sshDir()
	if err != nil {
		return nil, err
	}

	var candidates []SSHConfigImportCandidate
	for _, entry := range entries {
		if entry.Type != "HOST" || entry.Pattern == "" {
			continue
		}
		c := SSHConfigImportCandidate{
			Pattern: entry.Pattern,
			Port:    22,
		}
		for _, d := range entry.Directives {
			switch strings.ToLower(d.Key) {
			case "hostname":
				c.HostName = d.Value
			case "port":
				if p, err := strconv.Atoi(d.Value); err == nil {
					c.Port = p
				}
			case "user":
				c.User = d.Value
			case "identityfile":
				c.IdentityFile = d.Value
				c.HasKey = fileExists(resolveIdentityFile(d.Value, sshDir))
			}
		}
		candidates = append(candidates, c)
	}
	return candidates, nil
}

// ImportSSHConfigHosts imports selected Host entries from ~/.ssh/config as vShell connections.
func (a *AppService) ImportSSHConfigHosts(patterns []string) error {
	entries, err := a.ReadSSHConfig()
	if err != nil {
		return err
	}

	sshDir, err := sshDir()
	if err != nil {
		return err
	}

	// Build a map of pattern -> entry for quick lookup
	entryMap := make(map[string]*SSHConfigEntry)
	for i := range entries {
		if entries[i].Type == "HOST" && entries[i].Pattern != "" {
			entryMap[entries[i].Pattern] = &entries[i]
		}
	}

	for _, pattern := range patterns {
		entry, ok := entryMap[pattern]
		if !ok {
			continue
		}

		form := models.ConnectionForm{
			ID:         uuid.New().String(),
			Name:       pattern,
			Host:       pattern, // fallback
			Port:       22,
			Username:   "",
			AuthType:   models.AuthPassword,
			UploadPath: "/",
			SortOrder:  0,
		}

		for _, d := range entry.Directives {
			switch strings.ToLower(d.Key) {
			case "hostname":
				form.Host = d.Value
			case "port":
				if p, err := strconv.Atoi(d.Value); err == nil {
					form.Port = p
				}
			case "user":
				form.Username = d.Value
			case "identityfile":
				keyPath := resolveIdentityFile(d.Value, sshDir)
				if keyData, err := os.ReadFile(keyPath); err == nil {
					form.AuthType = models.AuthPrivateKey
					form.PrivateKey = strings.TrimSpace(string(keyData))
				}
			}
		}

		if err := a.CreateConnection(form); err != nil {
			return fmt.Errorf("import %s: %w", pattern, err)
		}
	}
	return nil
}

func resolveIdentityFile(value string, sshDir string) string {
	if strings.HasPrefix(value, "~") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, value[2:])
	}
	if filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(sshDir, value)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
