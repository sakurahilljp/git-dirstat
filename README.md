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
  - File/directory exclusions using standard glob patterns (`--exclude` / `-e` with `**` doublestar support).
  - Direct root file grouping (`(root files)` / `<target>/ (root files)`).
- **Multiple Output Formats**:
  - **Table**: Formatted ANSI-colored CLI table with thousand separators and automatic TTY detection.
  - **JSON**: Machine-readable JSON output with detailed per-directory statistics.
  - **CSV / TSV**: Delimited plain text for spreadsheet analysis and CI/CD pipelines.
- **Flexible Sorting**: Sort results by `added` (default), `deleted`, `net`, `files`, or `path`, with reverse order support (`--reverse` / `-r`).

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

```text
git-dirstat [OPTIONS] [<commit> [<commit>] | <commit>..<commit> | <commit>...<commit>] [-- <target-path>]
```

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
Exclude test files, generated documentation, or vendor folders using glob patterns:
```bash
git-dirstat -e "**/*_test.go" -e "docs/**" -e "vendor/**"
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
| `--no-color` | | `false` | Suppress ANSI color codes in table output. |
| `--version` | `-v` | | Print the version (`v0.2.0`). |
| `--help` | `-h` | | Print help and usage information. |

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

| Code | Name | Description |
| :---: | :--- | :--- |
| `0` | **Success** | Operation completed successfully. |
| `1` | **Runtime Error** | Git error, repository not found, uncommitted or revision lookup failure. |
| `2` | **Input Error** | Invalid flags, incompatible arguments, or depth $< 1$. |

---

## Documentation

- [Tool Specification](docs/git-dirstat-spec.md): Detailed specification and functional requirements.
- [Software Design Document](docs/software-design.md): Architectural design, package structure, and data flow.
- [Changelog](CHANGELOG.md): History of notable changes.

---

## License

This project is licensed under the [MIT License](LICENSE).
