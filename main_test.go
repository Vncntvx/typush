package main

import (
	"testing"
)

func TestNewRootCommands(t *testing.T) {
	root := NewRoot()
	expected := []string{"search", "info", "outdated", "update", "check", "clean", "dev", "download", "bump"}
	cmdMap := make(map[string]bool)
	for _, c := range root.Commands() {
		cmdMap[c.Name()] = true
	}

	for _, name := range expected {
		if !cmdMap[name] {
			t.Errorf("expected command %q to be registered on root", name)
		}
	}
}
