package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/sakurahilljp/git-dirstat/pkg/model"
	"github.com/spf13/cobra"
)

func newTestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "git-dirstat",
	}
	cmd.Flags().StringP("target", "t", ".", "Target path")
	cmd.Flags().IntP("depth", "d", 1, "Depth")
	cmd.Flags().StringP("sort", "s", "added", "Sort field")
	cmd.Flags().BoolP("reverse", "r", false, "Reverse sort")
	cmd.Flags().StringP("format", "f", "table", "Format")
	cmd.Flags().StringArrayP("exclude", "e", []string{}, "Exclude patterns")
	cmd.Flags().Bool("no-color", false, "No color")
	return cmd
}

func TestParseAndValidate(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		flags     []string
		wantType  model.CommitSpecType
		wantFrom  string
		wantTo    string
		wantC1    string
		wantC2    string
		wantTgt   string
		wantDepth int
		wantErr   bool
	}{
		{
			name:     "default working tree",
			args:     []string{},
			wantType: model.CommitSpecWorkingTree,
			wantTgt:  ".",
		},
		{
			name:     "single commit",
			args:     []string{"v1.0.0"},
			wantType: model.CommitSpecCommitToHead,
			wantFrom: "v1.0.0",
			wantTo:   "HEAD",
			wantTgt:  ".",
		},
		{
			name:     "two-dot range",
			args:     []string{"main..feature"},
			wantType: model.CommitSpecTwoDot,
			wantFrom: "main",
			wantTo:   "feature",
			wantTgt:  ".",
		},
		{
			name:     "two-dot missing to",
			args:     []string{"main.."},
			wantType: model.CommitSpecTwoDot,
			wantFrom: "main",
			wantTo:   "HEAD",
			wantTgt:  ".",
		},
		{
			name:     "two-dot missing from",
			args:     []string{"..feature"},
			wantType: model.CommitSpecTwoDot,
			wantFrom: "HEAD",
			wantTo:   "feature",
			wantTgt:  ".",
		},
		{
			name:     "two-dot both missing",
			args:     []string{".."},
			wantType: model.CommitSpecTwoDot,
			wantFrom: "HEAD",
			wantTo:   "HEAD",
			wantTgt:  ".",
		},
		{
			name:     "three-dot range",
			args:     []string{"main...feature"},
			wantType: model.CommitSpecThreeDot,
			wantC1:   "main",
			wantC2:   "feature",
			wantTgt:  ".",
		},
		{
			name:     "three-dot missing to",
			args:     []string{"main..."},
			wantType: model.CommitSpecThreeDot,
			wantC1:   "main",
			wantC2:   "HEAD",
			wantTgt:  ".",
		},
		{
			name:     "three-dot missing from",
			args:     []string{"...feature"},
			wantType: model.CommitSpecThreeDot,
			wantC1:   "HEAD",
			wantC2:   "feature",
			wantTgt:  ".",
		},
		{
			name:     "two commits",
			args:     []string{"commitA", "commitB"},
			wantType: model.CommitSpecTwoCommits,
			wantFrom: "commitA",
			wantTo:   "commitB",
			wantTgt:  ".",
		},
		{
			name:     "target with dash syntax",
			args:     []string{"HEAD", "--", "src/"},
			wantType: model.CommitSpecCommitToHead,
			wantFrom: "HEAD",
			wantTo:   "HEAD",
			wantTgt:  "src/",
		},
		{
			name:     "dash syntax only",
			args:     []string{"--", "pkg/"},
			wantType: model.CommitSpecWorkingTree,
			wantTgt:  "pkg/",
		},
		{
			name:    "range with extra commit arg",
			args:    []string{"a..b", "c"},
			wantErr: true,
		},
		{
			name:    "three commits",
			args:    []string{"a", "b", "c"},
			wantErr: true,
		},
		{
			name:    "conflicting target flag and dash",
			args:    []string{"--", "src/"},
			flags:   []string{"-t", "other/"},
			wantErr: true,
		},
		{
			name:    "invalid depth 0",
			args:    []string{},
			flags:   []string{"-d", "0"},
			wantErr: true,
		},
		{
			name:    "invalid sort",
			args:    []string{},
			flags:   []string{"-s", "unknown"},
			wantErr: true,
		},
		{
			name:    "invalid format",
			args:    []string{},
			flags:   []string{"-f", "xml"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newTestCmd()
			fullArgs := append(tt.flags, tt.args...)
			cmd.SetArgs(fullArgs)
			// Execute flag parsing by calling ParseFlags on fullArgs
			if err := cmd.ParseFlags(fullArgs); err != nil {
				if !tt.wantErr {
					t.Fatalf("unexpected flag parse error: %v", err)
				}
				return
			}

			cfg, err := ParseAndValidate(cmd, cmd.Flags().Args())
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseAndValidate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				var exitErr *model.ExitCodeError
				if !errors.As(err, &exitErr) || exitErr.Code != 2 {
					t.Errorf("expected ExitCode 2 error, got %v", err)
				}
				return
			}

			if cfg.CommitOpts.SpecType != tt.wantType {
				t.Errorf("SpecType = %v, want %v", cfg.CommitOpts.SpecType, tt.wantType)
			}
			if cfg.CommitOpts.FromCommit != tt.wantFrom {
				t.Errorf("FromCommit = %q, want %q", cfg.CommitOpts.FromCommit, tt.wantFrom)
			}
			if cfg.CommitOpts.ToCommit != tt.wantTo {
				t.Errorf("ToCommit = %q, want %q", cfg.CommitOpts.ToCommit, tt.wantTo)
			}
			if cfg.CommitOpts.Commit1 != tt.wantC1 {
				t.Errorf("Commit1 = %q, want %q", cfg.CommitOpts.Commit1, tt.wantC1)
			}
			if cfg.CommitOpts.Commit2 != tt.wantC2 {
				t.Errorf("Commit2 = %q, want %q", cfg.CommitOpts.Commit2, tt.wantC2)
			}
			if cfg.TargetPath != tt.wantTgt {
				t.Errorf("TargetPath = %q, want %q", cfg.TargetPath, tt.wantTgt)
			}
		})
	}
}

func TestParseAndValidate_Errors(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectedErr string
	}{
		{
			name:        "too many commits",
			args:        []string{"commit1", "commit2", "commit3"},
			expectedErr: "too many commit arguments",
		},
		{
			name:        "malformed two dots",
			args:        []string{"commit1..commit2..commit3"},
			expectedErr: "malformed range notation",
		},
		{
			name:        "malformed three dots",
			args:        []string{"commit1...commit2...commit3"},
			expectedErr: "malformed range notation",
		},
		{
			name:        "range with extra",
			args:        []string{"commit1..commit2", "extra"},
			expectedErr: "cannot combine range notation",
		},
		{
			name:        "four dots",
			args:        []string{"commit1....commit2"},
			expectedErr: "malformed range notation",
		},
		{
			name:        "multiple target paths after dash",
			args:        []string{"--", "path1", "path2"},
			expectedErr: "only one target path can be specified after '--'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewRootCommand()
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.expectedErr)
			}
			if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("expected error containing %q, got %q", tt.expectedErr, err.Error())
			}
		})
	}
}
