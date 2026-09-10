package gitutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsFileBinary(t *testing.T) {
	dir := t.TempDir()
	
	textFile := filepath.Join(dir, "text.txt")
	os.WriteFile(textFile, []byte("hello world\n"), 0644)
	
	binFile := filepath.Join(dir, "bin.dat")
	os.WriteFile(binFile, append([]byte("hello"), 0, 0, 0), 0644)
	
	emptyFile := filepath.Join(dir, "empty.txt")
	os.WriteFile(emptyFile, []byte(""), 0644)

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"text", textFile, false},
		{"binary", binFile, true},
		{"empty", emptyFile, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := isFileBinary(tt.path)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("isFileBinary() = %v, want %v", got, tt.want)
			}
		})
	}
}
