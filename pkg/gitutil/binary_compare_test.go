package gitutil

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestCompareBinaryFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Initialize repo
	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}

	// 1. Identical files
	binPathIdentical := "identical.bin"
	binContentIdentical := []byte("identical binary \x00 content")
	err = os.WriteFile(filepath.Join(tempDir, binPathIdentical), binContentIdentical, 0644)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	_, err = wt.Add(binPathIdentical)
	if err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	// 2. Different content same size
	binPathDiff := "diff.bin"
	binContentDiffA := []byte("diff content A")
	err = os.WriteFile(filepath.Join(tempDir, binPathDiff), binContentDiffA, 0644)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	_, err = wt.Add(binPathDiff)
	if err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	// 3. Large files
	binPathLarge := "large.bin"
	binContentLargeA := append(bytes.Repeat([]byte("A"), 16384), 'A')
	err = os.WriteFile(filepath.Join(tempDir, binPathLarge), binContentLargeA, 0644)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	_, err = wt.Add(binPathLarge)
	if err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	// 4. Large files different
	binPathLargeDiff := "large_diff.bin"
	binContentLargeDiffA := append(bytes.Repeat([]byte("A"), 16384), 'A')
	err = os.WriteFile(filepath.Join(tempDir, binPathLargeDiff), binContentLargeDiffA, 0644)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	_, err = wt.Add(binPathLargeDiff)
	if err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	commitHash, err := wt.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	commit, err := repo.CommitObject(commitHash)
	if err != nil {
		t.Fatalf("failed to get commit: %v", err)
	}
	tree, err := commit.Tree()
	if err != nil {
		t.Fatalf("failed to get tree: %v", err)
	}

	// Modify diff.bin
	binContentDiffB := []byte("diff content B")
	err = os.WriteFile(filepath.Join(tempDir, binPathDiff), binContentDiffB, 0644)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Modify large_diff.bin
	binContentLargeDiffB := append(bytes.Repeat([]byte("A"), 16384), 'B')
	err = os.WriteFile(filepath.Join(tempDir, binPathLargeDiff), binContentLargeDiffB, 0644)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Unreadable disk file test setup
	binPathUnreadable := "missing.bin"

	tests := []struct {
		name       string
		path       string
		diskPath   string
		setupDisk  func(string)
		wantResult bool
		wantErr    bool
	}{
		{
			name:       "identical files",
			path:       binPathIdentical,
			diskPath:   filepath.Join(tempDir, binPathIdentical),
			wantResult: true,
			wantErr:    false,
		},
		{
			name:       "different content same size",
			path:       binPathDiff,
			diskPath:   filepath.Join(tempDir, binPathDiff),
			wantResult: false,
			wantErr:    false,
		},
		{
			name:       "large identical files",
			path:       binPathLarge,
			diskPath:   filepath.Join(tempDir, binPathLarge),
			wantResult: true,
			wantErr:    false,
		},
		{
			name:       "large different files",
			path:       binPathLargeDiff,
			diskPath:   filepath.Join(tempDir, binPathLargeDiff),
			wantResult: false,
			wantErr:    false,
		},
		{
			name:       "unreadable disk file",
			path:       binPathIdentical,
			diskPath:   filepath.Join(tempDir, binPathUnreadable), // This file doesn't exist
			wantResult: false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headFile, err := tree.File(tt.path)
			if err != nil {
				t.Fatalf("failed to get head file: %v", err)
			}
			var fi os.FileInfo
			if tt.diskPath != "" {
				fi, _ = os.Stat(tt.diskPath)
			}
			identical, err := compareBinaryFiles(headFile, tt.diskPath, fi)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if identical != tt.wantResult {
					t.Errorf("expected identical %v, got %v", tt.wantResult, identical)
				}
			}
		})
	}
}
