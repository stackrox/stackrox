---
name: bulk-update-go-dep
description: Update a Go package version across multiple branches by creating child branches for each target
---

# Bulk Update Go Dependency

This skill updates a Go package version across multiple branches by creating a separate child branch for each target branch.

## Arguments

The skill takes two arguments (same as `update-go-dep`):
- **Package name**: The full import path of the Go package to update (e.g., `github.com/stretchr/testify`)
- **Version**: The target version (e.g., `v1.8.4`, `v0.5.0`, or `latest`)

Parse these from the user's invocation of `/bulk-update-go-dep <package> <version>`

## Workflow

Follow these steps **sequentially**:

### 1. Parse inputs

Extract the package name and version from the arguments.

### 2. Get the username

Run `git config user.name` to get the current git username. This will be used in the branch naming.

Convert it to lowercase and replace spaces with hyphens for the branch name.

### 3. Ask for target branches

Use AskUserQuestion to ask the user which branches to target.

**Prompt**: "Which branches should I update? (comma-separated list, e.g., master, release-4.9, release-4.8)"

Parse the user's response by splitting on commas and trimming whitespace.

### 4. Confirm the plan

Show the user:
- The dependency being updated: `<package>@<version>`
- The target branches
- The branch naming pattern that will be used: `<username>/<target-branch>/bump_<dependency-name>_to_<version>`

Where `<dependency-name>` is the last component of the package path (e.g., `api` from `kubevirt.io/api`).

Use AskUserQuestion to confirm: "Proceed with creating these branches and applying updates? (yes/no)"

If the user says no, abort gracefully.

### 5. Save current branch

Before starting, save the current branch name:
```bash
git rev-parse --abbrev-ref HEAD
```

This will be used to return to the original branch after all updates.

### 6. For each target branch

For each target branch in the list:

#### a. Fetch latest

```bash
git fetch origin <target-branch>
```

#### b. Checkout target branch

```bash
git checkout <target-branch>
```

If this fails, report the error and continue to the next branch.

#### c. Pull latest changes

```bash
git pull origin <target-branch>
```

#### d. Create child branch

Generate the branch name:
- Format: `<username>/<target-branch>/bump_<dependency-name>_to_<version>`
- Example: `yann-brillouet/master/bump_api_to_v1.8.1`
- Sanitize the dependency name by:
  - Taking the last component after the last `/` or `.`
  - Replacing any remaining `/` or `.` with `_`
  - Converting to lowercase

Create and checkout the new branch:
```bash
git checkout -b <new-branch-name>
```

#### e. Apply the update

Use the Skill tool to invoke the `update-go-dep` skill:
```
/update-go-dep <package> <version>
```

This will update all go.mod files and create a commit.

#### f. Report progress

After each branch is processed, report to the user:
- Target branch: `<target-branch>`
- New branch created: `<new-branch-name>`
- Status: Success or failure with error details

### 7. Return to original branch

After all branches are processed, return to the original branch:
```bash
git checkout <original-branch>
```

### 8. Summary report

Provide a final summary:
- Total branches processed
- Successful updates (list branch names)
- Failed updates (list branch names with errors)
- Reminder that they can push the branches with `git push -u origin <branch-name>`
- Suggestion to create PRs for each branch if desired

## Example Usage

User invokes: `/bulk-update-go-dep kubevirt.io/api v1.8.1`

You should:
1. Get git username (e.g., "Yann Brillouet" → "yann-brillouet")
2. Ask: "Which branches should I update? (comma-separated)"
3. User responds: "master, release-4.9, release-4.8"
4. Confirm the plan
5. For each branch:
   - Fetch and checkout `master`
   - Create `yann-brillouet/master/bump_api_to_v1.8.1`
   - Run `/update-go-dep kubevirt.io/api v1.8.1`
   - Repeat for `release-4.9` and `release-4.8`
6. Return to original branch
7. Report summary

## Error Handling

- If a target branch doesn't exist, skip it and report the error
- If branch creation fails (branch already exists), report and skip
- If the update-go-dep skill fails, report and continue to next branch
- If git checkout fails, report and continue to next branch
- Always attempt to return to the original branch at the end

## Branch Naming

The branch name format is: `<username>/<target-branch>/bump_<dependency-name>_to_<version>`

**Sanitization rules**:
- Username: lowercase, spaces → hyphens
- Dependency name: take last component, replace `/` and `.` with `_`, lowercase
- Version: as provided (with or without `v` prefix)

**Examples**:
- `kubevirt.io/api` → `api`
- `github.com/stretchr/testify` → `testify`
- `golang.org/x/crypto` → `crypto`
- `k8s.io/client-go` → `client_go`

## Notes

- Each branch will have its own commit from the `update-go-dep` skill
- The branches are created but NOT automatically pushed to remote
- The user can review each branch and push them individually
- The user can create PRs from each branch to its parent branch
