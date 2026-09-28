# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Added hierarchical Tree View (`--tree` / `-f tree`) rendering directory changes in an indented tree structure with clean Unicode branch lines (`├── `, `└── `, `│   `), integrated with `--stat`, `--percent`, `--graph`, and `--depth`.
- Added interactive terminal UI browser (`-i` / `--interactive` / `tui` subcommand) powered by `bubbletea` and `lipgloss`:
  - 2-pane layout displaying interactive directory tree navigation on the left and direct modified file diffs on the right.
  - Interactive controls: `↑`/`↓`/`j`/`k` navigation, `Space`/`Enter` expand/collapse, `o`/`O` expand/collapse all, `s` cycle sort fields, `p` toggle percent, `g` toggle graph, `/` live path filtering, `?` help modal overlay, and `q` quit.
  - Graceful fallback and error reporting in non-terminal / pipe environments.
- Added `pkg/model/tree.go` defining `TreeNode` and `TreeReport`.
- Added `pkg/aggregator/tree.go` implementing `TreeAggregator` for hierarchical Trie construction, bottom-up roll-up calculations, and sorting.
- Added `pkg/formatter/tree.go` implementing `TreeFormatter` for formatted CLI tree rendering.
- Added `pkg/tui` package containing Elm-architecture Bubbletea application model, styles, and headless test suite.

## [0.3.0] - 2026-09-28

### Changed
- Refactored git diff processing from buffered slices to a streaming pipeline (`DiffCommitsStream`, `DiffWorkingTreeStream`, and `StreamAggregator`), drastically reducing peak memory footprint by 80–95% on large repositories.
- Switched working tree line counting to chunked 32KB buffer reading (`countLinesFromReader`), eliminating full-file string allocations in memory.
- Optimized binary file comparison using streaming chunk readers.
- Consolidated exclusion and target boundary filtering into `pkg/filter/PathFilter` to eliminate duplicate logic.

### Added
- Added `churn` sub-command (with alias `hotspot`) for historical code churn and hotspot analysis across commit histories.
  - Supports `--since`, `--until`, `-n` / `--max-count`, `--no-merges` (default true), `--first-parent`, and `--fast` (speedy commit frequency count skipping patch computation).
  - Metrics tracked per directory: `Commits`, `Files`, `Added`, `Deleted`, `Churn` (added+deleted), and `Percent`.
  - Supports sorting by `commits` (default), `churn`, `files`, `added`, `deleted`, `path`, or `percent`, with `--top N` truncation.
  - Full output format support: `table` (with `--graph`), `markdown`, `json`, `csv`, and `tsv`.
  - Comprehensive user guide added in `docs/hotspot-analysis-guide.md`.
- Added `--format markdown` (alias: `-f md`) output format supporting GitHub Flavored Markdown (GFM) tables with code span path protection and bold totals.
- Added `--percent` CLI flag to display directory modification ratio (percentage of total added + deleted lines).
- Added `--graph` CLI flag to display proportional inline change bar graph (`+` for additions in green, `-` for deletions in red).
- Added `--stat` CLI flag as an ergonomic shortcut to enable both `--percent` and `--graph` simultaneously.
- Supported `--sort percent` to sort entries by modification percentage / line churn.
- Added `Percent float64` field to `model.Entry`, automatically serialized to JSON output and optionally to CSV/TSV (`--percent`).
- Added `--exclude-from` and `--exclude-file` CLI flags allowing users to load exclusion glob patterns from external files (e.g., `.gitignore`, `.dirstatignore`), with support for multi-value flags, blank line skipping, and `#` comments.
- Added `pkg/filter.LoadPatternsFromFile` helper for line-by-line file pattern parsing.
- New `pkg/filter` package providing early pre-filtering for target boundaries and `doublestar` glob exclusions, skipping expensive Myers diff computations for out-of-scope files.
- Comprehensive architectural documentation for memory optimization in `docs/memory-optimization-architecture.md`.
- Expanded test coverage for streaming diff extraction, active working tree states, and binary file detection.
- Documented complete external CLI specifications in `README.md`, including revision syntax, range resolution, deterministic tie-breaking, zero-diff behaviors, and detailed exit code conditions.

### Fixed
- Synchronized `README.md` output examples (Table, JSON, CSV) with `git-dirstat-spec.md` and actual CLI formatting schemas.
- Updated end-to-end execution sequence diagram in `docs/software-design.md` to accurately reflect the streaming pipeline.

## [0.2.0] - 2026-09-10

### Fixed
- Fixed repository discovery and reference resolution failure in directories created via `git worktree` by enabling `EnableDotGitCommonDir` in `go-git`.

### Added
- Comprehensive boundary and error condition unit tests for CLI argument and range parsing.
- Subagent definition files for `code-reviewer` and `test-specialist`.
- Formal operational guardrails and guidelines in `AGENTS.md`.

## [0.1.0] - 2026-09-10

### Added
- Initial release of `git-dirstat` CLI tool.
- Pure Go Git diff aggregation engine using `go-git` without external Git binary dependencies.
- Git revision and range resolution:
  - Working tree diff against HEAD (0 arguments, staged + unstaged changes).
  - Single commit comparison against HEAD commit (`<commit>`).
  - Two-dot commit range (`<commit1>..<commit2>`).
  - Three-dot merge-base commit range (`<commit1>...<commit2>`).
  - Two explicit commits (`<commit1> <commit2>`).
  - Target subdirectory filtering via `--target <path>` or positional `-- <path>`.
- Aggregation and filtering pipeline:
  - Directory depth aggregation via `--depth` (minimum 1).
  - Glob pattern exclusions via `--exclude` (using `bmatcuk/doublestar/v4`).
  - Boundary filtering and path normalization.
  - Root directory files grouping (`(root files)` / `<target>/ (root files)`).
  - Deterministic sorting by `path`, `files`, `insertions`, `deletions`, `net` (signed), or `total` (default: `total`), with `--reverse` support.
- Output formatters:
  - Table formatter with column alignment, thousand separators, and ANSI color highlighting (with automatic TTY detection and `--no-color` flag).
  - JSON formatter (`--format json`) with structured schema.
  - CSV (`--format csv`) and TSV (`--format tsv`) formatters.
- Quality assurance and build:
  - Comprehensive unit tests for CLI args, aggregator, formatters, and git utilities.
  - End-to-End (E2E) integration test suite verifying mock Git repositories.
  - Multi-platform cross-compilation Makefile (`darwin`, `linux`, `windows` for `amd64` and `arm64`).
- Documentation:
  - Tool specification (`docs/git-dirstat-spec.md`).
  - Architecture and software design document (`docs/software-design.md`).
  - Code review report (`docs/code-review-report.md`).
- Project License:
  - MIT License (`LICENSE`).
