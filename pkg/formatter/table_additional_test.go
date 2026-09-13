package formatter

import (
	"bytes"
	"github.com/sakurahilljp/git-dirstat/pkg/model"
	"os"
	"testing"
)

func TestTableFormatter_Colors(t *testing.T) {
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	f := NewTableFormatter(false)
	var buf bytes.Buffer
	rep := &model.Report{
		Entries: []model.Entry{
			{Path: "dir/", Files: 1, Added: 10, Deleted: 5},
			{Path: "dir2/", Files: 1, Added: 5, Deleted: 10},
		},
		Summary: model.Summary{TotalFiles: 2, TotalAdded: 15, TotalDeleted: 15},
	}
	err := f.Format(&buf, rep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Unset NO_COLOR to trigger checking isatty (which should return false for bytes.Buffer)
	os.Unsetenv("NO_COLOR")
	f2 := NewTableFormatter(false)
	err = f2.Format(&buf, rep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
