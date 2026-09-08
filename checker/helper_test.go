package checker_test

import (
	"os"
	"path/filepath"
)

func mkfile(p string) error {
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte("x"), 0o644)
}
