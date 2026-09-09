# Project Rules & Guidelines

This document defines the operational rules and guidelines for AI agents and contributors working on the `git-dirstat` project.

---

## 1. Git Operation Rules (CRITICAL)

All AI agents and automated tools must strictly adhere to the following Git operation protocols:

### 1.1. Commits (`git commit`)
- **Explicit Approval Required**: Never run `git commit` without explicit user permission.
- **Workflow**:
  1. Stage changes using `git add <files>` (approval NOT required for staging).
  2. Verify staged diff using `git diff --cached` and check `git status`.
  3. Present the proposed commit message and a summary of staged changes to the user.
  4. Wait for explicit user confirmation/approval before executing `git commit`.

### 1.2. Remote Operations (`git push`, Remote Branches & Tags)
- **Explicit Approval Required**: Never execute any remote modification command without explicit user permission.
- **Restricted Commands**:
  - `git push` (branches, tags, or force push)
  - Creating, updating, or deleting remote branches or tags
  - Modifying remote configurations (`git remote set-url`, etc.)
- **Workflow**:
  1. Clearly describe the target remote, branch, or tag to be pushed or modified.
  2. Ask the user for explicit approval.
  3. Execute only after confirmation is granted.

### 1.3. Local Staging & Read-Only Operations (No Approval Needed)
- The following operations do NOT require prior approval:
  - `git add <file>` (staging local edits)
  - `git status`, `git diff`, `git log`, `git show`, `git tag -l` (inspection commands)
  - Local unit and integration tests (`go test ./...`, `make test`, `make build`)

---

## 2. Code Quality & Build Standards

- **Go Version**: Go >= 1.25.
- **No External Git Dependencies**: The core engine must rely solely on `go-git` (`github.com/go-git/go-git/v5`). Do not invoke the `git` binary from Go code.
- **Testing**: All unit tests (`pkg/...`, `cmd/...`) and E2E tests (`test/...`) must pass before requesting commit approval (`make test`).
- **Formatting**: Preserve formatting with `go fmt` and ensure static checks pass cleanly.
- **Documentation**:
  - Keep [README.md](README.md), [CHANGELOG.md](CHANGELOG.md), and [docs/](docs/) up to date with user-facing and architectural changes.
