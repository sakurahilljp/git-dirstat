package filter

import (
	"testing"
)

func TestPathFilter_ShouldProcess(t *testing.T) {
	tests := []struct {
		name            string
		targetPrefix    string
		excludePatterns []string
		filePath        string
		want            bool
	}{
		{
			name:         "root target, no excludes, matches everything",
			targetPrefix: "",
			filePath:     "src/main.go",
			want:         true,
		},
		{
			name:         "target prefix matching",
			targetPrefix: "src/",
			filePath:     "src/components/button.go",
			want:         true,
		},
		{
			name:         "target prefix non-matching",
			targetPrefix: "src/",
			filePath:     "pkg/model/model.go",
			want:         false,
		},
		{
			name:            "exclude pattern matching glob",
			targetPrefix:    "",
			excludePatterns: []string{"**/*.test.go", "vendor/**"},
			filePath:        "pkg/util/util.test.go",
			want:            false,
		},
		{
			name:            "exclude pattern matching basename",
			targetPrefix:    "",
			excludePatterns: []string{"*.lock"},
			filePath:        "nested/dir/package.lock",
			want:            false,
		},
		{
			name:            "exclude pattern non-matching",
			targetPrefix:    "src/",
			excludePatterns: []string{"*.md"},
			filePath:        "src/button.go",
			want:            true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewPathFilter(tt.targetPrefix, tt.excludePatterns)
			got := f.ShouldProcess(tt.filePath)
			if got != tt.want {
				t.Errorf("ShouldProcess(%q) = %v, want %v", tt.filePath, got, tt.want)
			}
		})
	}
}

func TestPathFilter_ShouldProcessChange(t *testing.T) {
	f := NewPathFilter("src/", []string{"*.lock"})

	// Inside target -> inside target
	if !f.ShouldProcessChange("src/old.go", "src/new.go") {
		t.Errorf("expected true for within target change")
	}

	// Outside -> outside
	if f.ShouldProcessChange("pkg/old.go", "pkg/new.go") {
		t.Errorf("expected false for outside target change")
	}

	// Outside -> inside (file moved into target)
	if !f.ShouldProcessChange("pkg/old.go", "src/new.go") {
		t.Errorf("expected true when destination is in target")
	}

	// Inside -> outside (file moved out of target)
	if !f.ShouldProcessChange("src/old.go", "pkg/new.go") {
		t.Errorf("expected true when source is in target")
	}

	// Nil filter matches everything
	var nilFilter *PathFilter
	if !nilFilter.ShouldProcessChange("any/path", "other/path") {
		t.Errorf("expected true for nil filter")
	}
}
