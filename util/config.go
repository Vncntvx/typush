package util

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// AppName is the binary/config name.
const AppName = "typush"

// legacyAppNames are previous project names, newest first. Load falls back
// to their config files once (token migration) so existing users keep login.
var legacyAppNames = []string{"typkg", "typush"}

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
// If the new config does not exist yet but a legacy "typush" config does,
// it is migrated (copied) forward.
func Load() (*Config, error) {
	path, err := ConfigFile()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if migrated, err := migrateLegacy(path); err == nil && migrated {
			return Load()
		}
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

// migrateLegacy copies the first found legacy config forward once.
// Reports whether a migration happened.
func migrateLegacy(newPath string) (bool, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return false, err
	}
	var legacy string
	var data []byte
	for _, name := range legacyAppNames {
		p := filepath.Join(base, name, "config.toml")
		if d, err := os.ReadFile(p); err == nil {
			legacy, data = p, d
			break
		}
	}
	if data == nil {
		return false, fmt.Errorf("no legacy config found")
	}
	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
		return false, err
	}
	if err := os.WriteFile(newPath, data, 0o600); err != nil {
		return false, err
	}
	fmt.Fprintf(os.Stderr, "Migrated config from %s\n", legacy)
	return true, nil
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
