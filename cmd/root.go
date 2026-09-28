package cmd

import (
	"os"

	"github.com/sakurahilljp/git-dirstat/pkg/aggregator"
	"github.com/sakurahilljp/git-dirstat/pkg/formatter"
	"github.com/sakurahilljp/git-dirstat/pkg/gitutil"
	"github.com/sakurahilljp/git-dirstat/pkg/model"
	"github.com/spf13/cobra"
)

var Version = "v0.0.0"

// NewRootCommand creates the root cobra command.
func NewRootCommand() *cobra.Command {
	var targetFlag string
	var depthFlag int
	var sortFlag string
	var reverseFlag bool
	var formatFlag string
	var noColorFlag bool
	var percentFlag bool
	var graphFlag bool
	var statFlag bool
	var treeFlag bool
	var interactiveFlag bool
	var workersFlag int

	rootCmd := &cobra.Command{
		Use:           "git-dirstat [OPTIONS] [<commit> [<commit>] | <commit>..<commit> | <commit>...<commit>] [-- <target-path>]",
		Short:         "git-dirstat aggregates git changes by directory",
		Long:          `git-dirstat inspects changes between commits or working tree in a Git repository and aggregates modifications by directory.`,
		Version:       Version,
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := ParseAndValidate(cmd, args)
			if err != nil {
				return err
			}

			// If interactive mode requested, invoke TUI handler
			if cfg.Interactive {
				return RunInteractiveTUI(cmd, args)
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

			// 3. Tree output mode (--tree or -f tree)
			if cfg.Tree || cfg.Format == model.FormatTree {
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
					err = gitutil.DiffWorkingTreeStream(repoCtx.Repo, resolved.HeadCommit, repoCtx.RepoRoot, gitutil.DiffOptions{PathFilter: pathFilter, Workers: cfg.Workers}, treeAgg.Consume)
				} else {
					err = gitutil.DiffCommitsStream(resolved.FromCommit, resolved.ToCommit, gitutil.DiffOptions{PathFilter: pathFilter, Workers: cfg.Workers}, treeAgg.Consume)
				}
				if err != nil {
					return err
				}

				report, err := treeAgg.Result()
				if err != nil {
					return err
				}

				tf := formatter.NewTreeFormatterWithOptions(cfg.NoColor, cfg.ShowPercent, cfg.ShowGraph, cfg.Depth)
				return tf.Format(cmd.OutOrStdout(), report)
			}

			// 4. Setup Stream Aggregator & Pre-filter for flat reports
			agg, err := aggregator.NewStreamAggregator(aggregator.AggregatorOptions{
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

			pathFilter := agg.PathFilter()

			// 5. Stream Diff & Aggregate
			if resolved.IsWorkingTree {
				err = gitutil.DiffWorkingTreeStream(repoCtx.Repo, resolved.HeadCommit, repoCtx.RepoRoot, gitutil.DiffOptions{PathFilter: pathFilter, Workers: cfg.Workers}, agg.Consume)
			} else {
				err = gitutil.DiffCommitsStream(resolved.FromCommit, resolved.ToCommit, gitutil.DiffOptions{PathFilter: pathFilter, Workers: cfg.Workers}, agg.Consume)
			}
			if err != nil {
				return err
			}

			report, err := agg.Result()
			if err != nil {
				return err
			}

			// 6. Format & Output
			var f formatter.Formatter
			switch cfg.Format {
			case model.FormatJSON:
				f = formatter.NewJSONFormatter()
			case model.FormatCSV:
				f = formatter.NewCSVFormatterWithPercent(cfg.ShowPercent)
			case model.FormatTSV:
				f = formatter.NewTSVFormatterWithPercent(cfg.ShowPercent)
			case model.FormatMarkdown:
				f = formatter.NewMarkdownFormatterWithOptions(cfg.ShowPercent, cfg.ShowGraph)
			case model.FormatTable:
				fallthrough
			default:
				f = formatter.NewTableFormatterWithOptions(cfg.NoColor, cfg.ShowPercent, cfg.ShowGraph)
			}

			return f.Format(cmd.OutOrStdout(), report)
		},
	}

	rootCmd.Flags().StringVarP(&targetFlag, "target", "t", ".", "Base directory path to scope aggregation (CWD-relative)")
	rootCmd.Flags().IntVarP(&depthFlag, "depth", "d", 1, "Directory tree depth relative to target path")
	rootCmd.Flags().StringVarP(&sortFlag, "sort", "s", "added", "Sort field: files, added, deleted, net, path, percent")
	rootCmd.Flags().BoolVarP(&reverseFlag, "reverse", "r", false, "Sort in ascending order (default: descending)")
	rootCmd.Flags().StringVarP(&formatFlag, "format", "f", "table", "Output format: table, json, csv, tsv, markdown, tree")
	rootCmd.Flags().StringArrayP("exclude", "e", []string{}, "File/path patterns to exclude (doublestar ** format)")
	rootCmd.Flags().StringArray("exclude-from", []string{}, "File containing patterns to exclude (one per line, # for comments)")
	rootCmd.Flags().StringArray("exclude-file", []string{}, "Alias for --exclude-from")
	rootCmd.Flags().BoolVar(&noColorFlag, "no-color", false, "Suppress ANSI color escapes")
	rootCmd.Flags().BoolVar(&percentFlag, "percent", false, "Show change percentage column")
	rootCmd.Flags().BoolVar(&graphFlag, "graph", false, "Show inline change bar graph")
	rootCmd.Flags().BoolVar(&statFlag, "stat", false, "Show both percentage and inline bar graph")
	rootCmd.Flags().BoolVar(&treeFlag, "tree", false, "Output hierarchical tree view")
	rootCmd.Flags().BoolVarP(&interactiveFlag, "interactive", "i", false, "Start interactive TUI browser")
	rootCmd.Flags().IntVarP(&workersFlag, "workers", "W", 1, "Number of concurrent workers for diff calculation (1 = serial, 0 = auto)")

	rootCmd.AddCommand(NewChurnCommand())
	rootCmd.AddCommand(NewTUICommand())

	return rootCmd
}
