package util_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vncntvx/typkg/util"
)

func TestLegacyMigration(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "") // ensure OS-default path is used
	t.Setenv("XDG_DATA_HOME", "")

	// Legacy typush config with a token.
	legacyDir := filepath.Join(home, "Library", "Application Support", "typush")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	token := "legacy-token-123"
	if err := os.WriteFile(filepath.Join(legacyDir, "config.toml"),
		[]byte("[tokens]\nuniverse = \""+token+"\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := util.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Tokens.Universe == nil || *cfg.Tokens.Universe != token {
		t.Fatalf("expected migrated token, got %+v", cfg.Tokens)
	}
	// New location must now exist.
	path, _ := util.ConfigFile()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected new config at %s: %v", path, err)
	}
}
