package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestChurnCommand_Help(t *testing.T) {
	cmd := NewRootCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"churn", "--help"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("churn --help failed: %v", err)
	}

	out := b.String()
	if !strings.Contains(out, "churn analyzes git commit history") {
		t.Errorf("churn help output missing description: %s", out)
	}
	if !strings.Contains(out, "--since") || !strings.Contains(out, "--max-count") || !strings.Contains(out, "--fast") {
		t.Errorf("churn help output missing expected flags: %s", out)
	}
}

func TestChurnCommand_InvalidArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "invalid depth",
			args:    []string{"churn", "--depth", "0"},
			wantErr: "depth must be greater than or equal to 1",
		},
		{
			name:    "invalid sort",
			args:    []string{"churn", "--sort", "unknown"},
			wantErr: "invalid sort field",
		},
		{
			name:    "invalid format",
			args:    []string{"churn", "--format", "invalid"},
			wantErr: "invalid format",
		},
		{
			name:    "invalid since date",
			args:    []string{"churn", "--since", "not-a-date"},
			wantErr: "invalid --since date",
		},
		{
			name:    "negative top",
			args:    []string{"churn", "--top", "-1"},
			wantErr: "top must be non-negative",
		},
		{
			name:    "too many positional args",
			args:    []string{"churn", "c1", "c2"},
			wantErr: "too many positional arguments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewRootCommand()
			b := bytes.NewBufferString("")
			cmd.SetOut(b)
			cmd.SetErr(b)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestParseDate(t *testing.T) {
	// Standard ISO 8601
	d, err := parseDate("2026-01-15")
	if err != nil {
		t.Fatalf("parseDate failed: %v", err)
	}
	if d.Year() != 2026 || d.Month() != 1 || d.Day() != 15 {
		t.Errorf("unexpected date parsed: %v", d)
	}

	// Relative time
	dRel, err := parseDate("7 days ago")
	if err != nil {
		t.Fatalf("parseDate relative failed: %v", err)
	}
	if dRel == nil {
		t.Fatal("expected non-nil relative date")
	}

	// Invalid
	_, err = parseDate("invalid")
	if err == nil {
		t.Fatal("expected error for invalid date")
	}
}

func TestParseDateExtended(t *testing.T) {
	tests := []struct {
		input   string
		isError bool
	}{
		{"2 hours ago", false},
		{"1 week ago", false},
		{"2 weeks ago", false},
		{"1 month ago", false},
		{"3 months ago", false},
		{"1 year ago", false},
		{"2 years ago", false},
		{"2026/01/02", false},
		{"2026-01-02 15:04:05", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := parseDate(tt.input)
			if tt.isError && err == nil {
				t.Errorf("expected error for %q", tt.input)
			}
			if !tt.isError && err != nil {
				t.Errorf("unexpected error for %q: %v", tt.input, err)
			}
		})
	}
}

func TestChurnCommand_ValidArgs(t *testing.T) {
	cmd := NewRootCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"churn", "--depth", "2", "--sort", "added", "--reverse", "--top", "5", "--format", "json", "--since", "1 week ago", "--max-count", "10", "--fast", "--stat", "--percent", "--graph", "--exclude", "vendor/**", "--exclude-from", "empty.txt"})

	// It will try to run and since it's a real command, it might fail on run because we are outside a git repo or something, but the args parsing will pass
	// Actually we expect it to fail with "not a git repository" or similar if we don't mock it, but we can verify the error message is not about argument validation.
	err := cmd.Execute()
	if err != nil && (strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "must be")) {
		t.Errorf("unexpected validation error: %v", err)
	}
}
