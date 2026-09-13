# git-dirstat

A fast, lightweight CLI tool written in Go that analyzes and aggregates Git diffs (insertions, deletions, net line changes, and modified file counts) grouped by directory.

Built entirely on pure Go ([go-git](https://github.com/go-git/go-git)), `git-dirstat` runs as a single self-contained binary with zero dependency on the external `git` command-line executable.

---

## Features

- **Pure Go Engine**: Powered by `go-git`, ensuring fast, deterministic, cross-platform execution without invoking external Git processes.
- **Git Worktree Support**: Full compatibility with `git worktree` environments via automatic pointer and `commondir` reference resolution.
- **Rich Revision & Range Support**:
  - Uncommitted working tree diff against `HEAD` (staged + unstaged modifications).
  - Single commit or branch against `HEAD` (`<commit>`).
  - Two-dot commit range (`<commit1>..<commit2>`).
  - Three-dot merge-base range (`<commit1>...<commit2>`).
  - Two explicit commits (`<commit1> <commit2>`).
- **Flexible Scoping & Aggregation**:
  - Hierarchical directory depth control (`--depth` / `-d`).
  - Scoped subdirectories (`--target` / `-t` or `-- <target-path>`).
  - File/directory exclusions using standard glob patterns (`--exclude` / `-e`) or external pattern files (`--exclude-from` / `--exclude-file`).
  - Direct root file grouping (`(root files)` / `<target>/ (root files)`).
- **Multiple Output Formats**:
  - **Table**: Formatted ANSI-colored CLI table with thousand separators and automatic TTY detection.
  - **JSON**: Machine-readable JSON output with detailed per-directory statistics.
  - **CSV / TSV**: Delimited plain text for spreadsheet analysis and CI/CD pipelines.
- **Flexible Sorting**: Sort results by `added` (default), `deleted`, `net`, `files`, or `path`, with reverse order support (`--reverse` / `-r`).
- **Binary File Support**: Modified binary files (e.g. images, archives, compiled binaries) are counted in `Files` (`+1`), with line counts set to zero (`Added: 0`, `Deleted: 0`, `Net: 0`).

---

## Installation

### Prerequisites
- Go 1.25 or later (if building from source)

### From Source

Clone the repository and build using the provided `Makefile`:

```bash
git clone https://github.com/sakurahilljp/git-dirstat.git
cd git-dirstat
make build
```

The compiled binary will be placed at `./dist/git-dirstat`. You can move it to any directory in your `PATH`:

```bash
sudo install -m 755 dist/git-dirstat /usr/local/bin/git-dirstat
```

### Cross-Compilation

To build binaries for all supported platforms (Linux, macOS, and Windows for both `amd64` and `arm64`):

```bash
make build-all
```

The output binaries will be created under the `dist/` directory.

---

## Usage

```bash
git-dirstat [OPTIONS] [<commit> [<commit>] | <commit>..<commit> | <commit>...<commit>] [-- <target-path>]
```

or via the flag syntax:

```bash
git-dirstat [OPTIONS] [-t <target-path>] [<commit> [<commit>] | <commit>..<commit> | <commit>...<commit>]
```

### Argument Resolution & Revision Syntax

- **Supported Revisions**: Branch names (`main`), tag names (`v1.0.0`), full SHA hashes, short SHA hashes ($\ge 7$ chars), and relative revisions (`HEAD~1`, `HEAD~3`).
- **Commit Range Resolution**:
  - **Zero arguments**: Compares current `HEAD` against the working tree (combining staged and unstaged modifications; untracked files excluded).
  - **One commit argument (`<commit>`)**: Compares `<commit>` against current `HEAD`.
  - **Two-dot range (`<commit1>..<commit2>`)**: Compares `<commit1>` against `<commit2>`. If either side is omitted (e.g. `..feature` or `main..`), the missing side automatically defaults to `HEAD`.
  - **Three-dot range (`<commit1>...<commit2>`)**: Compares the **merge base** (common ancestor) of `<commit1>` and `<commit2>` against `<commit2>`. If either side is omitted, the missing side defaults to `HEAD`.
  - **Two arguments (`<commit1> <commit2>`)**: Compares `<commit1>` against `<commit2>`.
- **Syntax Restrictions**:
  - Combining range notation (`..` or `...`) with additional commit arguments is prohibited (Exit Code 2).
  - Specifying both `-t / --target` and `-- <target-path>` simultaneously is prohibited (Exit Code 2).

### Examples

#### 1. Analyze Uncommitted Changes in Working Tree
Inspect all staged and unstaged modifications against the current `HEAD`:
```bash
git-dirstat
```

#### 2. Compare a Branch or Commit with HEAD
```bash
git-dirstat main
```

#### 3. Compare Commit Ranges
Two-dot range:
```bash
git-dirstat v1.0.0..v1.1.0
```

Three-dot range (compares changes from the merge base of `main` and `feature` to `feature`):
```bash
git-dirstat main...feature
```

Compare two arbitrary commits:
```bash
git-dirstat a1b2c3d e4f5g6h
```

#### 4. Scope to a Subdirectory
You can filter changes to a target directory using `--target` or `-- <target-path>`:
```bash
git-dirstat --target pkg/
# or equivalently:
git-dirstat -- pkg/
```

#### 5. Adjust Directory Depth
Aggregate changes at depth 2 (e.g. `pkg/aggregator/` instead of `pkg/`):
```bash
git-dirstat -d 2
```

#### 6. Exclude Patterns
Exclude test files, generated documentation, or vendor folders using glob patterns or pattern files:
```bash
# Via command-line patterns
git-dirstat -e "**/*_test.go" -e "docs/**" -e "vendor/**"

# Via pattern file (one pattern per line, ignores blank lines and '#' comments)
git-dirstat --exclude-from .dirstatignore

# Both can be combined
git-dirstat --exclude-file .gitignore -e "tmp/**"
```

#### 7. Change Sorting
Sort by modified file count in ascending order:
```bash
git-dirstat -s files -r
```

Sort by net lines changed:
```bash
git-dirstat -s net
```

#### 8. Output Formats
Output as JSON:
```bash
git-dirstat -f json
```

Output as CSV or TSV:
```bash
git-dirstat -f csv > stats.csv
git-dirstat -f tsv > stats.tsv
```

---

## Options

| Flag | Short | Default | Description |
| :--- | :---: | :---: | :--- |
| `--target` | `-t` | `.` | Base directory path to scope aggregation (relative to CWD). |
| `--depth` | `-d` | `1` | Directory tree depth relative to target path (must be $\ge 1$). |
| `--sort` | `-s` | `added` | Sort column: `added`, `deleted`, `net`, `files`, or `path`. |
| `--reverse` | `-r` | `false` | Sort in ascending order instead of descending. |
| `--format` | `-f` | `table` | Output format: `table`, `json`, `csv`, or `tsv`. |
| `--exclude` | `-e` | `[]` | Exclude paths matching glob patterns (`doublestar` syntax). Repeatable. |
| `--exclude-from` | | `[]` | Exclude paths matching patterns from file(s). Alias: `--exclude-file`. Repeatable. |
| `--no-color` | | `false` | Suppress ANSI color codes in table output. |
| `--version` | `-v` | | Print the version (`v0.2.0`). |
| `--help` | `-h` | | Print help and usage information. |

---

## Diff & Aggregation Behavior

- **Root Files Grouping**: Files located directly inside the target directory without deeper subdirectories are grouped into:
  - Table / CSV / TSV: `<target>/ (root files)` (or `(root files)` if target is repository root `.`).
  - JSON: directory `path` set to target (e.g. `src/` or `.`) with `"is_root": true`.
- **Sorting & Deterministic Tie-Breaking**:
  - Default sort field is `added` descending.
  - When sorting by `net`, values are evaluated as signed numbers (`+10 > 0 > -20` descending).
  - Deterministic tie-breaking: rows with identical values are always sorted alphabetically by `path` in ascending order.
  - The `--reverse` (`-r`) flag reverses the overall sorted order.
- **Binary Files**: Modified binary files (e.g. images, PDFs, archives, compiled binaries) are counted as `+1` in `Files`, with `Added: 0`, `Deleted: 0`, and `Net: 0`.
- **Renames & Moves**: Rename detection is intentionally not performed. Moved files are aggregated as deletions at the source path and additions at the destination path.
- **Exclude Pattern Files**: When passing `--exclude-from` (or `--exclude-file`), patterns are loaded line by line from the designated files. Empty lines and lines starting with `#` are ignored as comments. Leading/trailing whitespace is trimmed. Patterns from multiple files and command-line `-e / --exclude` flags are merged together.
- **Untracked Files**: When analyzing uncommitted changes in the working tree, untracked files are excluded. Only tracked files with staged or unstaged modifications are included.
- **Zero-Diff Behavior**: When no changes exist:
  - **Table**: Prints table headers and a `TOTAL` row with all zeroes.
  - **JSON**: Outputs `"summary"` with all zeroes and `"entries": []` (empty array).
  - **CSV / TSV**: Emits only the column header line without data rows.

---

## Output Examples

### Table Output (Default)

```text
Target: . (Depth: 1)

Directory                         Files       Added     Deleted          Net
----------------------------------------------------------------------------
pkg/                                  4         704          90         +614
cmd/                                  3         354          30         +324
(root files)                          5          68          10          +58
----------------------------------------------------------------------------
TOTAL                                12        1126         130         +996
```

### JSON Output (`-f json`)

```json
{
  "target": ".",
  "depth": 1,
  "summary": {
    "total_files": 12,
    "total_added": 1126,
    "total_deleted": 130,
    "net": 996
  },
  "entries": [
    {
      "path": "pkg/",
      "is_root": false,
      "files": 4,
      "added": 704,
      "deleted": 90,
      "net": 614
    },
    {
      "path": "cmd/",
      "is_root": false,
      "files": 3,
      "added": 354,
      "deleted": 30,
      "net": 324
    },
    {
      "path": ".",
      "is_root": true,
      "files": 5,
      "added": 68,
      "deleted": 10,
      "net": 58
    }
  ]
}
```

### CSV Output (`-f csv`)

```csv
path,files,added,deleted,net
pkg/,4,704,90,614
cmd/,3,354,30,324
(root files),5,68,10,58
```

---

## Exit Codes

All error messages are written to standard error (`stderr`).

| Code | Category | Description |
| :---: | :--- | :--- |
| `0` | **Success** | Operation completed successfully (including zero diffs found). |
| `1` | **Git / Runtime Error** | `.git` repository not found, empty repository without commits, unresolvable commit/branch/tag ref, invalid commit hash, or no common ancestor found in three-dot range (`...`). |
| `2` | **User Input Error** | Unrecognized flags, `--depth < 1`, invalid `--sort` or `--format` values, conflicting `-t` and `-- <target-path>`, malformed range syntax, combining range notation with extra commit arguments, or unreadable exclude pattern file specified via `--exclude-from` / `--exclude-file`. |

---

## Documentation

- [Tool Specification](docs/git-dirstat-spec.md): Detailed specification and functional requirements.
- [Software Design Document](docs/software-design.md): Architectural design, package structure, and data flow.
- [Changelog](CHANGELOG.md): History of notable changes.

---

## License

This project is licensed under the [MIT License](LICENSE).
