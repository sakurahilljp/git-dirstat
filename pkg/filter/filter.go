package filter

import (
	"bufio"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// PathFilter determines whether a file path falls within the target scope and is not excluded.
type PathFilter struct {
	targetPrefix    string
	excludePatterns []string
}

// NewPathFilter creates a new PathFilter with the given target prefix and exclude patterns.
// targetPrefix should be normalized with a trailing slash if non-empty (e.g., "src/"), or empty for root.
func NewPathFilter(targetPrefix string, excludePatterns []string) *PathFilter {
	prefix := ""
	if targetPrefix != "" && targetPrefix != "." {
		prefix = strings.TrimSuffix(targetPrefix, "/") + "/"
	}
	return &PathFilter{
		targetPrefix:    prefix,
		excludePatterns: excludePatterns,
	}
}

// TargetPrefix returns the normalized target prefix.
func (f *PathFilter) TargetPrefix() string {
	if f == nil {
		return ""
	}
	return f.targetPrefix
}

// ShouldProcess returns true if the file path is within the target prefix and not excluded.
func (f *PathFilter) ShouldProcess(filePath string) bool {
	if f == nil {
		return true
	}
	p := filepath.ToSlash(filePath)

	// Check exclusion
	if f.IsExcluded(p) {
		return false
	}

	// Check target boundary
	if f.targetPrefix != "" && !strings.HasPrefix(p, f.targetPrefix) {
		return false
	}

	return true
}

// ShouldProcessChange returns true if either the fromPath or toPath should be processed.
// This handles file moves/renames across target boundaries.
func (f *PathFilter) ShouldProcessChange(fromPath, toPath string) bool {
	if f == nil {
		return true
	}
	if fromPath != "" && f.ShouldProcess(fromPath) {
		return true
	}
	if toPath != "" && f.ShouldProcess(toPath) {
		return true
	}
	return false
}

// IsExcluded returns true if the path matches any exclude pattern.
func (f *PathFilter) IsExcluded(filePath string) bool {
	if f == nil || len(f.excludePatterns) == 0 {
		return false
	}
	p := filepath.ToSlash(filePath)
	for _, pat := range f.excludePatterns {
		pat = filepath.ToSlash(pat)
		if matched, err := doublestar.Match(pat, p); err == nil && matched {
			return true
		}
		// If pattern contains no slash, check matching on file basename
		if !strings.Contains(pat, "/") {
			if matched, err := doublestar.Match(pat, path.Base(p)); err == nil && matched {
				return true
			}
		}
	}
	return false
}

// LoadPatternsFromFile reads patterns from a file, skipping empty lines and comments starting with '#'.
func LoadPatternsFromFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var patterns []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return patterns, nil
}
