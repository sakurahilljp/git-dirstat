package cmd

import (
	"os"

	"github.com/mattn/go-isatty"
	"github.com/sakurahilljp/git-dirstat/pkg/aggregator"
	"github.com/sakurahilljp/git-dirstat/pkg/gitutil"
	"github.com/sakurahilljp/git-dirstat/pkg/model"
	"github.com/sakurahilljp/git-dirstat/pkg/tui"
	"github.com/spf13/cobra"
)

// NewTUICommand creates the 'tui' subcommand.
func NewTUICommand() *cobra.Command {
	var targetFlag string
	var depthFlag int
	var sortFlag string
	var reverseFlag bool
	var noColorFlag bool
	var percentFlag bool
	var graphFlag bool
	var statFlag bool

	tuiCmd := &cobra.Command{
		Use:           "tui [OPTIONS] [<commit> [<commit>] | <commit>..<commit> | <commit>...<commit>] [-- <target-path>]",
		Short:         "Launch interactive terminal UI to explore directory hierarchy",
		Long:          `tui launches an interactive terminal user interface to explore and drill down the directory diff tree.`,
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunInteractiveTUI(cmd, args)
		},
	}

	tuiCmd.Flags().StringVarP(&targetFlag, "target", "t", ".", "Base directory path to scope aggregation (CWD-relative)")
	tuiCmd.Flags().IntVarP(&depthFlag, "depth", "d", 1, "Directory tree depth relative to target path")
	tuiCmd.Flags().StringVarP(&sortFlag, "sort", "s", "added", "Sort field: files, added, deleted, net, path, percent")
	tuiCmd.Flags().BoolVarP(&reverseFlag, "reverse", "r", false, "Sort in ascending order (default: descending)")
	tuiCmd.Flags().StringArrayP("exclude", "e", []string{}, "File/path patterns to exclude (doublestar ** format)")
	tuiCmd.Flags().StringArray("exclude-from", []string{}, "File containing patterns to exclude (one per line, # for comments)")
	tuiCmd.Flags().StringArray("exclude-file", []string{}, "Alias for --exclude-from")
	tuiCmd.Flags().BoolVar(&noColorFlag, "no-color", false, "Suppress ANSI color escapes")
	tuiCmd.Flags().BoolVar(&percentFlag, "percent", false, "Show change percentage column")
	tuiCmd.Flags().BoolVar(&graphFlag, "graph", false, "Show inline change bar graph")
	tuiCmd.Flags().BoolVar(&statFlag, "stat", false, "Show both percentage and inline bar graph")

	return tuiCmd
}

// RunInteractiveTUI parses config and runs the interactive TUI.
func RunInteractiveTUI(cmd *cobra.Command, args []string) error {
	// Verify terminal environment
	if !isatty.IsTerminal(os.Stdin.Fd()) && !isatty.IsCygwinTerminal(os.Stdin.Fd()) {
		return model.NewInputError("cannot run interactive TUI in non-terminal environment")
	}

	cfg, err := ParseAndValidate(cmd, args)
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return model.NewRuntimeError("failed to get current working directory: %w", err)
	}

	repoCtx, err := gitutil.OpenRepository(cwd)
	if err != nil {
		return err
	}

	resolved, err := gitutil.ResolveCommits(repoCtx.Repo, cfg.CommitOpts)
	if err != nil {
		return err
	}

	treeAgg, err := aggregator.NewTreeAggregator(aggregator.AggregatorOptions{
		RepoRoot:   repoCtx.RepoRoot,
		Cwd:        cwd,
		TargetPath: cfg.TargetPath,
		Depth:      cfg.Depth,
		Sort:       cfg.Sort,
		Reverse:    cfg.Reverse,
		Exclude:    cfg.Exclude,
	})
	if err != nil {
		return err
	}

	pathFilter := treeAgg.PathFilter()

	if resolved.IsWorkingTree {
		err = gitutil.DiffWorkingTreeStream(repoCtx.Repo, resolved.HeadCommit, repoCtx.RepoRoot, pathFilter, treeAgg.Consume)
	} else {
		err = gitutil.DiffCommitsStream(resolved.FromCommit, resolved.ToCommit, pathFilter, treeAgg.Consume)
	}
	if err != nil {
		return err
	}

	report, err := treeAgg.Result()
	if err != nil {
		return err
	}

	return tui.Run(report, tui.Options{
		ShowPercent: cfg.ShowPercent,
		ShowGraph:   cfg.ShowGraph,
		NoColor:     cfg.NoColor,
		SortField:   cfg.Sort,
		Reverse:     cfg.Reverse,
	})
}
