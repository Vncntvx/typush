package config

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// AppName is kept as "typush" (not typush-go) so config/data paths stay
// compatible with the Rust original.
const AppName = "typush"

const defaultPackagesSubdir = "typst/packages" // from typst-kit

type Config struct {
	Tokens RegistryTokens `toml:"tokens"`
}

type RegistryTokens struct {
	Universe *string `toml:"universe,omitempty"`
}

func ConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get the config directory: %w", err)
	}
	return filepath.Join(base, AppName), nil
}

func ConfigFile() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

// Load always returns a valid config, creating it on first run.
func Load() (*Config, error) {
	path, err := ConfigFile()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		cfg := &Config{}
		if err := Save(cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read the configuration file: %w", err)
	}
	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse the configuration file: %w", err)
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	path, err := ConfigFile()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create the configuration directory: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to write the configuration file: %w", err)
	}
	defer f.Close()
	if err := toml.NewEncoder(f).Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode the configuration file: %w", err)
	}
	return nil
}

// TypstLocalDir is the data dir (not cache) where packages are installed.
func TypstLocalDir() (string, error) {
	var base string
	// Prefer XDG_DATA_HOME on all platforms for predictability, fall back to OS default.
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		base = xdg
	} else {
		var err error
		base, err = dataDir()
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(base, defaultPackagesSubdir), nil
}

func TempSubdir(id string) string {
	sum := sha256.Sum256([]byte(id))
	return filepath.Join(os.TempDir(), fmt.Sprintf("%s-%x", AppName, sum[:8]))
}
