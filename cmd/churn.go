package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/sakurahilljp/git-dirstat/pkg/aggregator"
	"github.com/sakurahilljp/git-dirstat/pkg/filter"
	"github.com/sakurahilljp/git-dirstat/pkg/formatter"
	"github.com/sakurahilljp/git-dirstat/pkg/gitutil"
	"github.com/sakurahilljp/git-dirstat/pkg/model"
	"github.com/spf13/cobra"
)

// NewChurnCommand creates the churn/hotspot cobra command.
func NewChurnCommand() *cobra.Command {
	var targetFlag string
	var depthFlag int
	var sortFlag string
	var reverseFlag bool
	var topFlag int
	var formatFlag string
	var sinceFlag string
	var untilFlag string
	var maxCountFlag int
	var noMergesFlag bool
	var firstParentFlag bool
	var fastFlag bool
	var noColorFlag bool
	var percentFlag bool
	var graphFlag bool
	var statFlag bool

	churnCmd := &cobra.Command{
		Use:     "churn [OPTIONS] [<revision-range>] [-- <target-path>]",
		Aliases: []string{"hotspot"},
		Short:   "Analyze code churn and hotspots across commit history",
		Long: `churn analyzes git commit history to detect hotspots and calculate code churn (commit frequency and lines modified) aggregated by directory.

Examples:
  git-dirstat churn
  git-dirstat churn --since="2026-01-01"
  git-dirstat churn -n 100 --depth 2 --graph
  git-dirstat churn main..feature --sort churn --top 10
  git-dirstat churn --fast --sort commits
`,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := parseAndValidateChurn(cmd, args)
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

			sinceStr := ""
			if cfg.Since != nil {
				sinceStr = cfg.Since.Format("2006-01-02")
			}
			untilStr := ""
			if cfg.Until != nil {
				untilStr = cfg.Until.Format("2006-01-02")
			}

			agg, err := aggregator.NewChurnAggregator(aggregator.ChurnAggregatorOptions{
				RepoRoot:    repoCtx.RepoRoot,
				Cwd:         cwd,
				TargetPath:  cfg.TargetPath,
				Depth:       cfg.Depth,
				Sort:        cfg.Sort,
				Reverse:     cfg.Reverse,
				Top:         cfg.Top,
				Exclude:     cfg.Exclude,
				Since:       sinceStr,
				Until:       untilStr,
				CommitRange: cfg.CommitRange,
			})
			if err != nil {
				return err
			}

			pathFilter := agg.PathFilter()

			err = gitutil.WalkCommitHistoryStream(repoCtx.Repo, gitutil.ChurnWalkOptions{
				CommitRange: cfg.CommitRange,
				Since:       cfg.Since,
				Until:       cfg.Until,
				MaxCount:    cfg.MaxCount,
				NoMerges:    cfg.NoMerges,
				FirstParent: cfg.FirstParent,
				Fast:        cfg.Fast,
			}, pathFilter, func(commit *object.Commit, diffs []model.FileDiff) error {
				return agg.ConsumeCommit(commit.Hash.String(), diffs)
			})
			if err != nil {
				return err
			}

			report, err := agg.Result()
			if err != nil {
				return err
			}

			var f formatter.ChurnFormatter
			switch cfg.Format {
			case model.FormatJSON:
				f = formatter.NewChurnJSONFormatter()
			case model.FormatCSV:
				f = formatter.NewChurnCSVFormatterWithPercent(cfg.ShowPercent)
			case model.FormatTSV:
				f = formatter.NewChurnTSVFormatterWithPercent(cfg.ShowPercent)
			case model.FormatMarkdown:
				f = formatter.NewChurnMarkdownFormatterWithOptions(cfg.ShowPercent, cfg.ShowGraph)
			case model.FormatTable:
				fallthrough
			default:
				f = formatter.NewChurnTableFormatter(cfg.NoColor, cfg.ShowPercent, cfg.ShowGraph)
			}

			return f.FormatChurn(cmd.OutOrStdout(), report)
		},
	}

	flags := churnCmd.Flags()
	flags.StringVarP(&targetFlag, "target", "t", ".", "target directory to analyze")
	flags.IntVarP(&depthFlag, "depth", "d", 1, "aggregation depth relative to target directory")
	flags.StringVarP(&sortFlag, "sort", "s", model.SortCommits, "sort field (commits, churn, files, added, deleted, path, percent)")
	flags.BoolVarP(&reverseFlag, "reverse", "r", false, "reverse sort order")
	flags.IntVar(&topFlag, "top", 0, "limit output to top N directories")
	flags.StringVarP(&formatFlag, "format", "f", model.FormatTable, "output format (table, json, csv, tsv, markdown)")
	flags.StringVar(&sinceFlag, "since", "", "show commits more recent than a specific date")
	flags.StringVar(&untilFlag, "until", "", "show commits older than a specific date")
	flags.IntVarP(&maxCountFlag, "max-count", "n", 0, "limit the number of commits to walk")
	flags.BoolVar(&noMergesFlag, "no-merges", true, "do not analyze merge commits with multiple parents")
	flags.BoolVar(&firstParentFlag, "first-parent", false, "follow only the first parent commit upon seeing a merge commit")
	flags.BoolVar(&fastFlag, "fast", false, "skip line diff calculations and count commit frequencies and touched files quickly")
	flags.BoolVar(&noColorFlag, "no-color", false, "disable colored output")
	flags.BoolVar(&percentFlag, "percent", false, "show percent column")
	flags.BoolVar(&graphFlag, "graph", false, "show visual bar graph")
	flags.BoolVar(&statFlag, "stat", false, "enable both --percent and --graph")
	flags.StringArrayP("exclude", "e", []string{}, "glob pattern to exclude (can be specified multiple times)")
	flags.StringArray("exclude-from", []string{}, "file containing exclude patterns")
	flags.StringArray("exclude-file", []string{}, "alias for --exclude-from")

	return churnCmd
}

func parseAndValidateChurn(cmd *cobra.Command, args []string) (*model.ChurnConfig, error) {
	depth, err := cmd.Flags().GetInt("depth")
	if err != nil {
		return nil, model.NewInputError("invalid depth flag: %w", err)
	}
	if depth < 1 {
		return nil, model.NewInputError("depth must be greater than or equal to 1, got %d", depth)
	}

	sortField, err := cmd.Flags().GetString("sort")
	if err != nil {
		return nil, model.NewInputError("invalid sort flag: %w", err)
	}
	switch sortField {
	case model.SortCommits, model.SortChurn, model.SortFiles, model.SortAdded, model.SortDeleted, model.SortPath, model.SortPercent:
	default:
		return nil, model.NewInputError("invalid sort field: %q (allowed: commits, churn, files, added, deleted, path, percent)", sortField)
	}

	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return nil, model.NewInputError("invalid format flag: %w", err)
	}
	switch format {
	case model.FormatTable, model.FormatJSON, model.FormatCSV, model.FormatTSV, model.FormatMarkdown, "md":
		if format == "md" {
			format = model.FormatMarkdown
		}
	default:
		return nil, model.NewInputError("invalid format: %q (allowed: table, json, csv, tsv, markdown)", format)
	}

	reverse, _ := cmd.Flags().GetBool("reverse")
	top, _ := cmd.Flags().GetInt("top")
	if top < 0 {
		return nil, model.NewInputError("top must be non-negative, got %d", top)
	}
	noColor, _ := cmd.Flags().GetBool("no-color")
	percent, _ := cmd.Flags().GetBool("percent")
	graph, _ := cmd.Flags().GetBool("graph")
	stat, _ := cmd.Flags().GetBool("stat")
	if stat {
		percent = true
		graph = true
	}

	maxCount, _ := cmd.Flags().GetInt("max-count")
	if maxCount < 0 {
		return nil, model.NewInputError("max-count must be non-negative, got %d", maxCount)
	}
	noMerges, _ := cmd.Flags().GetBool("no-merges")
	firstParent, _ := cmd.Flags().GetBool("first-parent")
	fast, _ := cmd.Flags().GetBool("fast")

	sinceStr, _ := cmd.Flags().GetString("since")
	var sinceTime *time.Time
	if sinceStr != "" {
		t, err := parseDate(sinceStr)
		if err != nil {
			return nil, model.NewInputError("invalid --since date %q: %w", sinceStr, err)
		}
		sinceTime = t
	}

	untilStr, _ := cmd.Flags().GetString("until")
	var untilTime *time.Time
	if untilStr != "" {
		t, err := parseDate(untilStr)
		if err != nil {
			return nil, model.NewInputError("invalid --until date %q: %w", untilStr, err)
		}
		untilTime = t
	}

	exclude, _ := cmd.Flags().GetStringArray("exclude")
	excludeFrom, _ := cmd.Flags().GetStringArray("exclude-from")
	excludeFile, _ := cmd.Flags().GetStringArray("exclude-file")

	allExcludeFiles := append(excludeFrom, excludeFile...)
	for _, ef := range allExcludeFiles {
		patterns, err := filter.LoadPatternsFromFile(ef)
		if err != nil {
			return nil, model.NewInputError("failed to read exclude file %q: %w", ef, err)
		}
		exclude = append(exclude, patterns...)
	}

	flagTarget, _ := cmd.Flags().GetString("target")
	targetChanged := cmd.Flags().Changed("target")

	// Separate commit range positional args from `-- <target-path>`
	dashIndex := cmd.ArgsLenAtDash()
	var commitArgs []string
	var dashTarget string

	if dashIndex >= 0 {
		commitArgs = args[:dashIndex]
		postDash := args[dashIndex:]
		if len(postDash) == 1 {
			dashTarget = postDash[0]
		} else if len(postDash) > 1 {
			return nil, model.NewInputError("only one target path can be specified after '--'")
		}
	} else {
		commitArgs = args
	}

	targetPath := "."
	if dashTarget != "" && targetChanged {
		return nil, model.NewInputError("cannot specify both -t/--target and '-- <target-path>'")
	} else if dashTarget != "" {
		targetPath = dashTarget
	} else if targetChanged {
		targetPath = flagTarget
	} else if flagTarget != "" {
		targetPath = flagTarget
	}

	commitRange := ""
	if len(commitArgs) == 1 {
		commitRange = commitArgs[0]
	} else if len(commitArgs) > 1 {
		return nil, model.NewInputError("too many positional arguments: expected at most 1 revision range, got %d", len(commitArgs))
	}

	return &model.ChurnConfig{
		CommitRange: commitRange,
		TargetPath:  targetPath,
		Depth:       depth,
		Since:       sinceTime,
		Until:       untilTime,
		MaxCount:    maxCount,
		Sort:        sortField,
		Reverse:     reverse,
		Top:         top,
		Format:      format,
		Exclude:     exclude,
		NoMerges:    noMerges,
		FirstParent: firstParent,
		Fast:        fast,
		ShowPercent: percent,
		ShowGraph:   graph,
		NoColor:     noColor,
	}, nil
}

var relativeTimeRegex = regexp.MustCompile(`^(\d+)\s*(days?|weeks?|months?|years?|hours?)\s+ago$`)

func parseDate(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	lower := strings.ToLower(s)

	if m := relativeTimeRegex.FindStringSubmatch(lower); m != nil {
		num, _ := strconv.Atoi(m[1])
		unit := m[2]
		now := time.Now()
		var d time.Duration
		switch {
		case strings.HasPrefix(unit, "hour"):
			d = time.Duration(num) * time.Hour
		case strings.HasPrefix(unit, "day"):
			d = time.Duration(num) * 24 * time.Hour
		case strings.HasPrefix(unit, "week"):
			d = time.Duration(num) * 7 * 24 * time.Hour
		case strings.HasPrefix(unit, "month"):
			d = time.Duration(num) * 30 * 24 * time.Hour
		case strings.HasPrefix(unit, "year"):
			d = time.Duration(num) * 365 * 24 * time.Hour
		}
		t := now.Add(-d)
		return &t, nil
	}

	layouts := []string{
		"2006-01-02",
		"2006/01/02",
		"2006-01-02 15:04:05",
		time.RFC3339,
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return &t, nil
		}
	}

	return nil, fmt.Errorf("unsupported date format: %q", s)
}
