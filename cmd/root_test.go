package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRootCommand_VersionAndHelp(t *testing.T) {
	// 1. Version flag
	root := NewRootCommand()
	var outBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetArgs([]string{"--version"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(outBuf.String(), Version) {
		t.Errorf("expected version %q in output, got: %s", Version, outBuf.String())
	}

	// 2. Short version flag -v
	outBuf.Reset()
	root = NewRootCommand()
	root.SetOut(&outBuf)
	root.SetArgs([]string{"-v"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(outBuf.String(), Version) {
		t.Errorf("expected version %q in output, got: %s", Version, outBuf.String())
	}

	// 3. Help flag
	outBuf.Reset()
	root = NewRootCommand()
	root.SetOut(&outBuf)
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(outBuf.String(), "git-dirstat") || !strings.Contains(outBuf.String(), "Usage:") {
		t.Errorf("expected usage in help output: %s", outBuf.String())
	}
}

func TestRootCommand_NonGitDir(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "root-test-nongit-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	origDir, _ := os.Getwd()
	_ = os.Chdir(tempDir)
	defer func() { _ = os.Chdir(origDir) }()

	root := NewRootCommand()
	root.SetArgs([]string{})
	err = root.Execute()
	if err == nil {
		t.Fatalf("expected error running in non-git directory")
	}
}
