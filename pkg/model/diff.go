package model

// FileDiff represents changes made to a single file.
type FileDiff struct {
	// Path is the file path relative to the repository root.
	Path string
	// Added is the number of added lines.
	Added int
	// Deleted is the number of deleted lines.
	Deleted int
	// IsBinary indicates if the file is binary.
	IsBinary bool
}
