package model

type CommitSpecType int

const (
	CommitSpecWorkingTree  CommitSpecType = iota // 0 commit args: Working tree vs HEAD
	CommitSpecCommitToHead                       // 1 commit arg: <commit> vs HEAD
	CommitSpecTwoDot                             // 1 commit arg: <c1>..<c2>
	CommitSpecThreeDot                           // 1 commit arg: <c1>...<c2>
	CommitSpecTwoCommits                         // 2 commit args: <c1> <c2>
)

// CommitOptions represents the parsed commit comparison specifications.
type CommitOptions struct {
	SpecType   CommitSpecType
	FromCommit string // For TwoDot, CommitToHead, TwoCommits: base/from commit ref
	ToCommit   string // target/to commit ref (HEAD if omitted / single commit)
	Commit1    string // For ThreeDot: left commit ref
	Commit2    string // For ThreeDot: right commit ref
}

// Config represents all resolved options for running git-dirstat.
type Config struct {
	CommitOpts CommitOptions
	TargetPath string
	Depth      int
	Sort       string
	Reverse    bool
	Format     string
	Exclude    []string
	NoColor    bool
}
