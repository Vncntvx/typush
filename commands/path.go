package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Vncntvx/typush/util"
)

// Path outputs the local Typst packages directory.
// If namespace is specified (e.g. "preview" or "local"), it outputs that namespace's path.
func Path(namespace string) error {
	base, err := util.TypstLocalDir()
	if err != nil {
		return err
	}

	target := strings.TrimPrefix(strings.TrimSpace(namespace), "@")
	if strings.ContainsAny(target, `/\`) || strings.Contains(target, "..") {
		return fmt.Errorf("invalid namespace %q: must not contain path separators or '..'", namespace)
	}
	var targetDir string
	if target == "" {
		targetDir = base
	} else {
		targetDir = filepath.Join(base, target)
	}

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "note: directory does not exist yet: %s\n", targetDir)
	}

	fmt.Println(targetDir)
	return nil
}
