package util

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func dataDir() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support"), nil
	case "windows":
		if v := os.Getenv("AppData"); v != "" {
			return v, nil
		}
		if v := os.Getenv("LOCALAPPDATA"); v != "" {
			return v, nil
		}
		return "", fmt.Errorf("failed to get the data directory: %%AppData%% not set")
	default: // linux + others: XDG
		if v := os.Getenv("XDG_DATA_HOME"); v != "" {
			return v, nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "share"), nil
	}
}
