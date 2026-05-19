package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
