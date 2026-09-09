package cmd

import (
	"os"

	"github.com/sakurahilljp/git-dirstat/pkg/aggregator"
	"github.com/sakurahilljp/git-dirstat/pkg/formatter"
	"github.com/sakurahilljp/git-dirstat/pkg/gitutil"
	"github.com/sakurahilljp/git-dirstat/pkg/model"
	"github.com/spf13/cobra"
)

var Version = "v0.2.0"

// NewRootCommand creates the root cobra command.
func NewRootCommand() *cobra.Command {
	var targetFlag string
	var depthFlag int
	var sortFlag string
	var reverseFlag bool
	var formatFlag string
	var noColorFlag bool

	rootCmd := &cobra.Command{
		Use:   "git-dirstat [OPTIONS] [<commit> [<commit>] | <commit>..<commit> | <commit>...<commit>] [-- <target-path>]",
		Short: "git-dirstat aggregates git changes by directory",
		Long:  `git-dirstat inspects changes between commits or working tree in a Git repository and aggregates modifications by directory.`,
		Version:       Version,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := ParseAndValidate(cmd, args)
			if err != nil {
				return err
			}

			cwd, err := os.Getwd()
			if err != nil {
				return model.NewRuntimeError("failed to get current working directory: %w", err)
			}

			// 1. Open Git Repository
			repoCtx, err := gitutil.OpenRepository(cwd)
			if err != nil {
				return err
			}

			// 2. Resolve commits
			resolved, err := gitutil.ResolveCommits(repoCtx.Repo, cfg.CommitOpts)
			if err != nil {
				return err
			}

			// 3. Diff
			var diffs []model.FileDiff
			if resolved.IsWorkingTree {
				diffs, err = gitutil.DiffWorkingTree(repoCtx.Repo, resolved.HeadCommit, repoCtx.RepoRoot)
			} else {
				diffs, err = gitutil.DiffCommits(resolved.FromCommit, resolved.ToCommit)
			}
			if err != nil {
				return err
			}

			// 4. Aggregate
			report, err := aggregator.Aggregate(diffs, aggregator.AggregatorOptions{
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

			// 5. Format & Output
			var f formatter.Formatter
			switch cfg.Format {
			case model.FormatJSON:
				f = formatter.NewJSONFormatter()
			case model.FormatCSV:
				f = formatter.NewCSVFormatter()
			case model.FormatTSV:
				f = formatter.NewTSVFormatter()
			case model.FormatTable:
				fallthrough
			default:
				f = formatter.NewTableFormatter(cfg.NoColor)
			}

			return f.Format(cmd.OutOrStdout(), report)
		},
	}

	rootCmd.Flags().StringVarP(&targetFlag, "target", "t", ".", "Base directory path to scope aggregation (CWD-relative)")
	rootCmd.Flags().IntVarP(&depthFlag, "depth", "d", 1, "Directory tree depth relative to target path")
	rootCmd.Flags().StringVarP(&sortFlag, "sort", "s", "added", "Sort field: files, added, deleted, net, path")
	rootCmd.Flags().BoolVarP(&reverseFlag, "reverse", "r", false, "Sort in ascending order (default: descending)")
	rootCmd.Flags().StringVarP(&formatFlag, "format", "f", "table", "Output format: table, json, csv, tsv")
	rootCmd.Flags().StringArrayP("exclude", "e", []string{}, "File/path patterns to exclude (doublestar ** format)")
	rootCmd.Flags().BoolVar(&noColorFlag, "no-color", false, "Suppress ANSI color escapes")

	return rootCmd
}
