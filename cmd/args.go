package cmd

import (
	"strings"

	"github.com/sakurahilljp/git-dirstat/pkg/model"
	"github.com/spf13/cobra"
)

// ParseAndValidate parses positional arguments and flags from a Cobra command.
func ParseAndValidate(cmd *cobra.Command, args []string) (*model.Config, error) {
	// 1. Validate flags
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
	case model.SortFiles, model.SortAdded, model.SortDeleted, model.SortNet, model.SortPath:
	default:
		return nil, model.NewInputError("invalid sort field: %q (allowed: files, added, deleted, net, path)", sortField)
	}

	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return nil, model.NewInputError("invalid format flag: %w", err)
	}
	switch format {
	case model.FormatTable, model.FormatJSON, model.FormatCSV, model.FormatTSV:
	default:
		return nil, model.NewInputError("invalid format: %q (allowed: table, json, csv, tsv)", format)
	}

	reverse, _ := cmd.Flags().GetBool("reverse")
	noColor, _ := cmd.Flags().GetBool("no-color")
	exclude, _ := cmd.Flags().GetStringArray("exclude")
	flagTarget, _ := cmd.Flags().GetString("target")
	targetChanged := cmd.Flags().Changed("target")

	// 2. Separate commit positional args from `-- <target-path>`
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

	// 3. Resolve target path & check for conflict
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

	// 4. Parse commit arguments
	commitOpts, err := parseCommitArgs(commitArgs)
	if err != nil {
		return nil, err
	}

	return &model.Config{
		CommitOpts: *commitOpts,
		TargetPath: targetPath,
		Depth:      depth,
		Sort:       sortField,
		Reverse:    reverse,
		Format:     format,
		Exclude:    exclude,
		NoColor:    noColor,
	}, nil
}

func parseCommitArgs(args []string) (*model.CommitOptions, error) {
	if len(args) == 0 {
		return &model.CommitOptions{
			SpecType: model.CommitSpecWorkingTree,
		}, nil
	}

	// Check if any argument contains range notation
	hasRange := false
	for _, arg := range args {
		if strings.Contains(arg, "..") {
			hasRange = true
			break
		}
	}

	if hasRange {
		if len(args) > 1 {
			return nil, model.NewInputError("cannot combine range notation ('..' or '...') with additional commit arguments")
		}
		rangeArg := args[0]
		// Check for 4 or more dots, which is malformed
		if strings.Contains(rangeArg, "....") {
			return nil, model.NewInputError("malformed range notation: %q", rangeArg)
		}
		// Three dots
		if strings.Contains(rangeArg, "...") {
			parts := strings.Split(rangeArg, "...")
			if len(parts) != 2 {
				return nil, model.NewInputError("malformed range notation: %q", rangeArg)
			}
			c1 := parts[0]
			c2 := parts[1]
			if c1 == "" {
				c1 = "HEAD"
			}
			if c2 == "" {
				c2 = "HEAD"
			}
			return &model.CommitOptions{
				SpecType: model.CommitSpecThreeDot,
				Commit1:  c1,
				Commit2:  c2,
			}, nil
		}

		// Two dots
		parts := strings.Split(rangeArg, "..")
		if len(parts) != 2 {
			return nil, model.NewInputError("malformed range notation: %q", rangeArg)
		}
		c1 := parts[0]
		c2 := parts[1]
		if c1 == "" {
			c1 = "HEAD"
		}
		if c2 == "" {
			c2 = "HEAD"
		}
		return &model.CommitOptions{
			SpecType:   model.CommitSpecTwoDot,
			FromCommit: c1,
			ToCommit:   c2,
		}, nil
	}

	if len(args) == 1 {
		return &model.CommitOptions{
			SpecType:   model.CommitSpecCommitToHead,
			FromCommit: args[0],
			ToCommit:   "HEAD",
		}, nil
	}

	if len(args) == 2 {
		return &model.CommitOptions{
			SpecType:   model.CommitSpecTwoCommits,
			FromCommit: args[0],
			ToCommit:   args[1],
		}, nil
	}

	return nil, model.NewInputError("too many commit arguments (expected at most 2, got %d)", len(args))
}
