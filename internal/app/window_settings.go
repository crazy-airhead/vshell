package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	defaultWindowWidth  = 1280
	defaultWindowHeight = 800
	minWindowWidth      = 960
	minWindowHeight     = 600
)

type WindowSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

func LoadWindowSize() WindowSize {
	size := WindowSize{
		Width:  defaultWindowWidth,
		Height: defaultWindowHeight,
	}

	path, err := windowSizeFilePath()
	if err != nil {
		return size
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return size
	}

	var saved WindowSize
	if err := json.Unmarshal(data, &saved); err != nil {
		return size
	}

	return normalizeWindowSize(saved)
}

func (a *AppService) SaveWindowSize(width int, height int) error {
	return SaveWindowSize(width, height)
}

func SaveWindowSize(width int, height int) error {
	size := normalizeWindowSize(WindowSize{Width: width, Height: height})

	path, err := windowSizeFilePath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create window config dir: %w", err)
	}

	data, err := json.MarshalIndent(size, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal window size: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write window size: %w", err)
	}

	return nil
}

func normalizeWindowSize(size WindowSize) WindowSize {
	if size.Width < minWindowWidth {
		size.Width = minWindowWidth
	}
	if size.Height < minWindowHeight {
		size.Height = minWindowHeight
	}
	return size
}

func windowSizeFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("get user config dir: %w", err)
	}

	return filepath.Join(configDir, "vshell", "window.json"), nil
}
