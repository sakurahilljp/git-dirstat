# CLI Tool Specification: git-dirstat

## 1. Overview

`git-dirstat` is a command-line tool that inspects changes between commits (or working tree state) in a Git repository and aggregates modifications—including changed file count, lines added, lines deleted, and net change—aggregated by directory.

Implemented in **Go** using the **`go-git`** (Pure Go) library, the tool runs as a self-contained executable binary without requiring the native `git` CLI installed on the host system.

---

## 2. Technical Stack & Environment

| Item | Specification / Technology |
| :--- | :--- |
| **Language** | Go (>= 1.22 recommended) |
| **Core Engine** | `github.com/go-git/go-git/v5` (Pure Go, CGO-free) |
| **CLI Parser** | `github.com/spf13/cobra` or Go standard `flag` |
| **Build Target** | Statically linked binary (`CGO_ENABLED=0`) |
| **Supported Platforms** | Linux (x86_64, aarch64), macOS (Intel, Apple Silicon), Windows (x64) |
| **Prerequisites** | No native `git` required; valid `.git` folder in target repository |

---

## 3. Command-Line Syntax

```bash
git-dirstat [OPTIONS] [<commit> [<commit>]] [-- <target-path>]
```

or via the flag syntax:

```bash
git-dirstat [OPTIONS] [-t <target-path>] [<commit> [<commit>]]
```

### 3.1 Argument Resolution

* **Commit Specifications:**
* Two arguments (`<commit1> <commit2>`): Compares `<commit1>` against `<commit2>`.
* One argument (`<commit>`): Compares `<commit>` against current `HEAD`.
* Zero arguments: Compares current `HEAD` against working tree (uncommitted modifications).
* Supported revisions include full/short SHA hashes (>= 7 chars), branch names, tag names, and relative revisions such as `HEAD~n`.

* **Target Path (`<target-path>` / `-t, --target`):**
* Restricts aggregation to files located within the designated base directory.
* Defaults to repository root (`.`) if omitted.

---

## 4. Options & Flags

| Flag | Short | Default | Type | Description |
| --- | --- | --- | --- | --- |
| `--target` | `-t` | `.` | string | Base directory path to scope aggregation (equivalent to `-- <path>`) |
| `--depth` | `-d` | `1` | int | **Directory tree depth relative to target path** |
| `--sort` | `-s` | `added` | string | Sort field: `files`, `added`, `deleted`, `net`, `path` |
| `--reverse` | `-r` | `false` | bool | Sort in ascending order (default: descending) |
| `--format` | `-f` | `table` | string | Output format: `table`, `json`, `csv`, `tsv` |
| `--exclude` | `-e` | None | []string | File/path patterns to exclude (glob format, multi-value allowed) |
| `--no-color` | | `false` | bool | Suppress ANSI color escapes (auto-disabled if stdout is non-TTY) |
| `--version` | `-v` | - | - | Print version information and exit |
| `--help` | `-h` | - | - | Print command usage instructions and exit |

---

## 5. Functional Requirements & Core Logic

### 5.1 Diff Extraction via go-git

1. **Repository Discovery:**

* Traverse working directory upwards to resolve `.git` (via `git.PlainOpenWithOptions` with `DetectDotGit: true`).

1. **Revision Resolution:**

* Resolve passed references (tags, branches, short hashes) into `plumbing.Hash` using `repo.ResolveRevision()`.

1. **Diff & Patch Processing:**

* Fetch matching Tree instances and execute `object.DiffTree(fromTree, toTree)`.
* Parse `changes.Patch()` to extract `patch.Stats()` holding additions, deletions, and file paths.
* For working tree evaluations, inspect `worktree.Status()` / `worktree.Diff()` to include staged/unstaged changes.

### 5.2 Target Filtering

* Evaluate each file path against `--target`.
* Files outside the base directory boundary are discarded prior to aggregation.
* Apply exclusions based on provided `--exclude` glob patterns (e.g., `*.lock`, `vendor/**`).

### 5.3 Relative Pathing & Depth Aggregation

Paths are evaluated relative to the specified `--target`:

1. **Path Normalization:**

* Strip target directory prefix.
* Example: Given `--target src/` and file `src/components/button/index.tsx`, relative path becomes `components/button/index.tsx`.

1. **Depth Slicing (`--depth`):**

* Segment path using `/`.
* **`--depth 1`:** Extract root segment → `components/` (aggregated key: `src/components/`).
* **`--depth 2`:** Extract up to second segment → `components/button/` (aggregated key: `src/components/button/`).
* **Root files directly under target:** Files directly inside the target directory without deeper nesting are grouped into `<target>/ (root files)`.

1. **Accumulator Metrics:**

* `Files`: Unique modified file count per bucket.
* `Added`: Sum of added lines.
* `Deleted`: Sum of deleted lines.
* `Net`: Calculated delta (`Added - Deleted`).

---

## 6. Output Examples

### 6.1 Scoped Directory with Depth 1 (Table Output)

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

### 6.2 JSON Output (CI/CD Integration)

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
      "files": 6,
      "added": 340,
      "deleted": 45,
      "net": 295
    },
    {
      "path": "backend/db/migrations/",
      "files": 4,
      "added": 180,
      "deleted": 10,
      "net": 170
    },
    {
      "path": "backend/ (root files)",
      "files": 2,
      "added": 60,
      "deleted": 40,
      "net": 20
    }
  ]
}

```

---

## 7. Error Handling & Exit Codes

| Exit Code | Category | Error Scenario |
| --- | --- | --- |
| `0` | Success | Operation completed successfully (including zero diffs) |
| `1` | Git / Runtime | `.git` repository not found, broken ref, or invalid commit hash |
| `2` | User Input | Invalid argument, `--depth < 1`, or unrecognized option flag |

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

EOF
