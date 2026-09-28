# CLI Tool Specification: git-dirstat

## 1. Overview

`git-dirstat` is a command-line tool that inspects changes between commits (or working tree state) in a Git repository and aggregates modifications—including changed file count, lines added, lines deleted, and net change—aggregated by directory.

Implemented in **Go** using the **`go-git`** (Pure Go) library, the tool runs as a self-contained executable binary without requiring the native `git` CLI installed on the host system.

---

## 2. Technical Stack & Environment

| Item | Specification / Technology |
| :--- | :--- |
| **Language** | Go (>= 1.25) |
| **Core Engine** | `github.com/go-git/go-git/v5` (Pure Go, CGO-free) |
| **CLI Parser** | `github.com/spf13/cobra` |
| **Pattern Matcher** | `github.com/bmatcuk/doublestar/v4` |
| **Build Target** | Statically linked binary (`CGO_ENABLED=0`) |
| **Supported Platforms** | Linux (x86_64, aarch64), macOS (Intel, Apple Silicon), Windows (x64) |
| **Prerequisites** | No native `git` required; valid `.git` folder in target repository |

---

## 3. Command-Line Syntax

```bash
git-dirstat [OPTIONS] [<commit> [<commit>] | <commit>..<commit> | <commit>...<commit>] [-- <target-path>]
```

or via the flag syntax:

```bash
git-dirstat [OPTIONS] [-t <target-path>] [<commit> [<commit>] | <commit>..<commit> | <commit>...<commit>]
```

### 3.1 Argument Resolution

* **Commit Specifications:**
  * **Zero arguments:** Compares current `HEAD` against working tree (aggregates both Staged and Unstaged changes; Untracked files are excluded).
  * **One argument:**
    * Single commit (`<commit>`): Compares `<commit>` against current `HEAD`.
    * Two-dot range (`<commit1>..<commit2>`): Compares `<commit1>` (base / from) against `<commit2>` (target / to). Equivalent to `<commit1> <commit2>`. If either side is omitted (e.g., `<commit>..` or `..<commit>`), the missing side defaults to `HEAD`.
    * Three-dot range (`<commit1>...<commit2>`): Compares the **merge base** (common ancestor) of `<commit1>` and `<commit2>` against `<commit2>`. This mirrors Git's pull request diff semantic. If either side is omitted, the missing side defaults to `HEAD`.
  * **Two arguments (`<commit1> <commit2>`):** Compares `<commit1>` (base / from) against `<commit2>` (target / to).
  * *Note:* Combining range notation (`..` / `...`) with additional commit positional arguments is prohibited and results in an error (Exit Code 2).
  * Supported revisions include full/short SHA hashes (>= 7 chars), branch names, tag names, and relative revisions such as `HEAD~n`.

* **Target Path (`<target-path>` / `-t, --target`):**
  * Evaluated relative to the execution current working directory (CWD), and normalized internally to a repository-root-relative path.
  * Defaults to repository root (`.`) if omitted.
  * Specifying both `-t / --target` and `-- <target-path>` simultaneously is prohibited and results in an error (Exit Code 2).

---

## 4. Options & Flags

| Flag | Short | Default | Type | Description |
| --- | --- | --- | --- | --- |
| `--target` | `-t` | `.` | string | Base directory path to scope aggregation (CWD-relative, equivalent to `-- <path>`) |
| `--depth` | `-d` | `1` | int | Directory tree depth relative to target path (must be >= 1) |
| `--sort` | `-s` | `added` | string | Sort field: `files`, `added`, `deleted`, `net`, `path`, `percent` |
| `--reverse` | `-r` | `false` | bool | Sort in ascending order (default: descending) |
| `--format` | `-f` | `table` | string | Output format: `table`, `json`, `csv`, `tsv`, `markdown` (alias: `md`), `tree` |
| `--tree` | | `false` | bool | Output hierarchical tree view (alias for `--format tree`) |
| `--interactive` | `-i` | `false` | bool | Launch interactive terminal UI (TUI) to explore directory hierarchy |
| `--percent` | | `false` | bool | Display change percentage column |
| `--graph` | | `false` | bool | Display proportional inline change bar graph column |
| `--stat` | | `false` | bool | Display both percentage and inline bar graph columns |
| `--exclude` | `-e` | None | []string | File/path patterns to exclude (doublestar `**` format, multi-value allowed) |
| `--exclude-from` | | None | []string | Read exclusion patterns from file(s) (one pattern per line, alias: `--exclude-file`) |
| `--no-color` | | `false` | bool | Suppress ANSI color escapes (auto-disabled if stdout is non-TTY) |
| `--version` | `-v` | - | - | Print version information and exit |
| `--help` | `-h` | - | - | Print command usage instructions and exit |

---

## 5. Functional Requirements & Core Logic

### 5.1 Diff Extraction via go-git

1. **Repository Discovery:**
   * Traverse working directory upwards to resolve `.git` (via `git.PlainOpenWithOptions` with `DetectDotGit: true` and `EnableDotGitCommonDir: true` for full `git worktree` compatibility).
   * If `.git` is not found, or the repository has no commits yet (empty repository), exit with Exit Code 1.

2. **Revision Resolution:**
   * Resolve passed references (tags, branches, short hashes) into `plumbing.Hash` using `repo.ResolveRevision()`.
   * **Two-dot (`..`) range:** Resolve left commit as `fromCommit` and right commit as `toCommit`.
   * **Three-dot (`...`) range:** Resolve both commits, find their common ancestor via `commit1.MergeBase(commit2)`. If multiple merge bases exist, choose the first; if no common ancestor exists, terminate with Exit Code 1. Set the merge base commit as `fromCommit` and `commit2` as `toCommit`.

3. **Diff & Patch Processing:**
   * **Commit comparison:** Fetch matching Tree instances (`fromCommit.Tree()` and `toCommit.Tree()`) and execute `object.DiffTree(fromTree, toTree)`. Parse `changes.Patch()` to extract additions, deletions, and file paths.
   * **Working tree comparison (0 arguments):** Inspect `worktree.Status()` / working tree modifications against `HEAD`. Combine staged and unstaged diffs. Untracked files are excluded.
   * **Binary Files:** Binary files are counted as `+1` in `Files`, with `Added: 0`, `Deleted: 0`, and `Net: 0`.
   * **Renames / Moves:** Rename detection is not performed. Moved files are aggregated as deletions in the source directory and additions in the destination directory.

### 5.2 Target Filtering & Exclusions

* Evaluate each file path against normalized `--target`. Files outside the base directory boundary are discarded prior to aggregation.
* Apply exclusions based on provided `--exclude` patterns and/or pattern files passed via `--exclude-from` (alias `--exclude-file`).
  * Matching is performed using `doublestar` against repository-root-relative paths (e.g. `vendor/**`, `*.lock`).
  * In pattern files, each line contains one pattern. Empty lines and comment lines (starting with `#`) are ignored. Leading and trailing whitespace is stripped.
  * Multiple files and CLI flags can be specified together; their patterns are merged.

### 5.3 Relative Pathing & Depth Aggregation

Paths are evaluated relative to the specified `--target`:

1. **Path Normalization:**
   * Strip target directory prefix.
   * Example: Given `--target src/` and file `src/components/button/index.tsx`, relative path becomes `components/button/index.tsx`.

2. **Depth Slicing (`--depth`):**
   * Segment path using `/`.
   * **`--depth 1`:** Extract root segment → `components/` (aggregated key: `src/components/`).
   * **`--depth 2`:** Extract up to second segment → `components/button/` (aggregated key: `src/components/button/`).
   * **Root files directly under target:** Files located directly inside the target directory without further subdirectories are grouped into:
     * Human-readable / Table / CSV / TSV: `<target>/ (root files)` (or `(root files)` if target is repository root `.`).
     * JSON: `path` contains the directory path (e.g., `src/` or `.`), with `is_root: true`.

3. **Accumulator Metrics:**
   * `Files`: Count of unique modified files in bucket.
   * `Added`: Sum of added lines.
   * `Deleted`: Sum of deleted lines.
   * `Net`: Calculated signed delta (`Added - Deleted`).

### 5.4 Sorting & Tie-Breaking

* Results are sorted by the field specified in `--sort` (default: `added`).
* When `--sort net` is specified, values are sorted by signed numerical value (e.g., `+10 > 0 > -20` descending).
* **Tie-Breaking:** If values are equal, rows are deterministically sorted by `path` in ascending alphabetical order.
* The `--reverse` (`-r`) flag reverses the overall sorted order.

---

## 6. Output Formats & Examples

### 6.1 Table Output (`--format table`, default)

Standard table with borders, column alignment, and a summary `TOTAL` row:

```bash
git-dirstat v1.0.0 HEAD -t src/ -d 1
```

```text
Target: src/ (Depth: 1)

Directory                     Files       Added     Deleted         Net
-----------------------------------------------------------------------
src/components/                  15         820         150        +670
src/services/                     8         450         120        +330
src/utils/                        3         120          35         +85
src/ (root files)                 2          30           5         +25
-----------------------------------------------------------------------
TOTAL                            28        1420         310       +1110
```

When no changes are detected, table header and zero total are displayed:

```text
Target: src/ (Depth: 1)

Directory                     Files       Added     Deleted         Net
-----------------------------------------------------------------------
TOTAL                             0           0           0           0
```

### 6.2 JSON Output (`--format json`)

Machine-readable JSON representation. Root file entries maintain clean directory paths with `is_root: true`:

```bash
git-dirstat main feature-x -t backend/ -d 2 -f json
```

```json
{
  "target": "backend/",
  "depth": 2,
  "summary": {
    "total_files": 12,
    "total_added": 580,
    "total_deleted": 95,
    "net": 485
  },
  "entries": [
    {
      "path": "backend/api/handlers/",
      "is_root": false,
      "files": 6,
      "added": 340,
      "deleted": 45,
      "net": 295
    },
    {
      "path": "backend/db/migrations/",
      "is_root": false,
      "files": 4,
      "added": 180,
      "deleted": 10,
      "net": 170
    },
    {
      "path": "backend/",
      "is_root": true,
      "files": 2,
      "added": 60,
      "deleted": 40,
      "net": 20
    }
  ]
}
```

When no changes are detected:

```json
{
  "target": "backend/",
  "depth": 2,
  "summary": {
    "total_files": 0,
    "total_added": 0,
    "total_deleted": 0,
    "net": 0
  },
  "entries": []
}
```

### 6.3 CSV & TSV Output (`--format csv`, `--format tsv`)

Delimited outputs contain a single header row followed by data records only (no `TOTAL` row):

```bash
git-dirstat -t src/ -f csv
```

```text
path,files,added,deleted,net
src/components/,15,820,150,670
src/services/,8,450,120,330
src/utils/,3,120,35,85
src/ (root files),2,30,5,25
```

TSV uses tab separators (`\t`) with the same schema. If no changes exist, only the header line is emitted.

### 6.4 Markdown Output (`--format markdown` / `-f md`)

GitHub Flavored Markdown (GFM) table suitable for pull request summaries and `$GITHUB_STEP_SUMMARY`. Paths are enclosed in code spans to protect underscore-separated identifiers, and totals are rendered in bold:

```bash
git-dirstat -t src/ -f markdown --stat
```

```markdown
**Target:** `src/` (Depth: 1)

| Directory | Files | Added | Deleted | Net | Percent | Graph |
| :--- | ---: | ---: | ---: | ---: | ---: | :--- |
| `src/components/` | 15 | 820 | 150 | +670 | 56.1% | `++++++++++--` |
| `src/services/` | 8 | 450 | 120 | +330 | 32.9% | `++++++-` |
| `src/utils/` | 3 | 120 | 35 | +85 | 9.0% | `+-` |
| `src/ (root files)` | 2 | 30 | 5 | +25 | 2.0% | `+` |
| **TOTAL** | **28** | **1420** | **310** | **+1110** | **100.0%** | `++++++++++++++++----` |
```

### 6.5 Tree Output (`--format tree` / `--tree`)

Hierarchical tree visualization with Unicode branch lines (`├──`, `└──`). Displays directory nesting with collapsible-like visual clarity:

```bash
git-dirstat -t src/ -f tree --stat
```

```text
Target: src/
Files: 28  Added: +1420  Deleted: -310  Net: +1110

src/
├── components/          15  +820  -150  +670   56.1%  ++++++++++--
│   ├── button/           8  +450   -80  +370   31.2%  ++++++-
│   └── modal/            7  +370   -70  +300   24.9%  +++++-
├── services/             8  +450  -120  +330   32.9%  ++++++-
├── utils/                3  +120   -35   +85    9.0%  +-
└── (root files)          2   +30    -5   +25    2.0%  +
```

---

## 7. Error Handling & Exit Codes

All errors are reported to `stderr`.

| Exit Code | Category | Error Scenario |
| --- | --- | --- |
| `0` | Success | Operation completed successfully (including zero diffs) |
| `1` | Git / Runtime | `.git` repository not found, repository has no commits, broken ref, invalid commit hash, or no common ancestor in `...` range |
| `2` | User Input | Invalid argument, malformed range notation, combining range syntax with extra commits, `--depth < 1`, unrecognized flag, conflicting `-t` and `-- <path>`, unreadable exclude pattern file specified via `--exclude-from` / `--exclude-file`, or running interactive TUI in a non-terminal environment |

---

## 8. Build & Distribution

Compile standalone static binaries using standard Go tooling:

```bash
# Linux (amd64)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/git-dirstat-linux-amd64 .

# macOS (Apple Silicon - arm64)
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o dist/git-dirstat-darwin-arm64 .

# Windows (x64)
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/git-dirstat-windows-amd64.exe .
```

---

## 9. Subcommands

In addition to the primary dirstat aggregation, `git-dirstat` provides specialized subcommands for historical hotspot analysis and interactive exploration.

### 9.1 Code Churn & Hotspot Analysis (`git-dirstat churn` / `git-dirstat hotspot`)

Analyzes Git commit history over time to detect architectural hotspots by calculating commit frequency and code churn (cumulative added and deleted lines) aggregated by directory.

#### Syntax

```bash
git-dirstat churn [OPTIONS] [<revision-range>] [-- <target-path>]
```

Alias: `git-dirstat hotspot`

#### Options & Flags

| Flag | Short | Default | Type | Description |
| --- | --- | --- | --- | --- |
| `--target` | `-t` | `.` | string | Base directory path to scope analysis (CWD-relative) |
| `--depth` | `-d` | `1` | int | Aggregation depth relative to target path (must be >= 1) |
| `--sort` | `-s` | `commits` | string | Sort field: `commits`, `churn`, `files`, `added`, `deleted`, `path`, `percent` |
| `--reverse` | `-r` | `false` | bool | Sort in ascending order (default: descending) |
| `--top` | | `0` | int | Limit output to top N directories (0: unlimited) |
| `--format` | `-f` | `table` | string | Output format: `table`, `json`, `csv`, `tsv`, `markdown` (alias: `md`) |
| `--since` | | `""` | string | Include commits after date (`YYYY-MM-DD` or `YYYY-MM-DDTHH:MM:SS`) |
| `--until` | | `""` | string | Include commits before date (`YYYY-MM-DD` or `YYYY-MM-DDTHH:MM:SS`) |
| `--max-count` | `-n` | `0` | int | Maximum number of commits to walk (0: unlimited) |
| `--no-merges` | | `true` | bool | Skip merge commits with multiple parents |
| `--first-parent` | | `false` | bool | Follow only the first parent commit upon merge commits |
| `--fast` | | `false` | bool | Skip line-level diffs; compute commit counts and modified files rapidly |
| `--percent` | | `false` | bool | Display percentage column (commit involvement rate or churn share) |
| `--graph` | | `false` | bool | Display visual proportional bar graph |
| `--stat` | | `false` | bool | Enable both `--percent` and `--graph` columns |
| `--exclude` | `-e` | None | []string | Doublestar exclusion patterns |
| `--exclude-from` | | None | []string | Read exclusion patterns from file (alias: `--exclude-file`) |
| `--no-color` | | `false` | bool | Suppress ANSI color escapes |

#### Output Example (Table with `--stat`)

```bash
git-dirstat churn -n 50 --depth 1 --stat
```

```text
Directory           Commits   Files     Added   Deleted     Churn   Percent   Graph
-----------------------------------------------------------------------------------------
pkg/                     32      18     +1450      -210      1660     64.0%   ++++++++++--
cmd/                     21       6      +480       -65       545     42.0%   ++++-
docs/                    14       4      +310       -40       350     28.0%   ++
(root files)              8       3       +95       -15       110     16.0%   +
-----------------------------------------------------------------------------------------
TOTAL                    50      31     +2335      -330      2665    100.0%   ++++++++++++-
```

---

### 9.2 Interactive Terminal UI (`git-dirstat tui` / `-i, --interactive`)

Launches a full-screen, interactive Terminal User Interface (TUI) to explore directory trees, drill down into nested packages, and inspect modifications.

#### Invocation

```bash
# Via dedicated subcommand:
git-dirstat tui [OPTIONS] [<commit> [<commit>] | <range>] [-- <target-path>]

# Or via the root interactive flag:
git-dirstat -i [OPTIONS] [<commit> [<commit>] | <range>] [-- <target-path>]
```

#### Terminal Requirement & Error Handling

- Requires an interactive TTY terminal (`os.Stdin` and `os.Stdout`).
- If invoked in non-interactive environments (pipes, scripts, CI/CD runners without TTY allocation), exits immediately with **Exit Code 2** (`cannot run interactive TUI in non-terminal environment`).

#### Keybindings & Controls

| Key | Action |
| --- | --- |
| `↑` / `k` | Move cursor up |
| `↓` / `j` | Move cursor down |
| `Enter` / `Space` | Expand or collapse selected directory branch |
| `←` / `h` | Collapse current directory / jump to parent |
| `→` / `l` | Expand selected directory |
| `s` | Cycle sort order (Added, Deleted, Net, Files, Path) |
| `q` / `Esc` / `Ctrl+C` | Exit interactive TUI |
