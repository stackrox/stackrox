---
name: update-go-dep
description: Update a Go package version across all go.mod files in the repository, run go mod tidy, and commit the changes
---

# Update Go Dependency

This skill updates a Go package version across all go.mod files in the repository.

## Arguments

The skill takes two arguments:
- **Package name**: The full import path of the Go package to update (e.g., `github.com/stretchr/testify`)
- **Version**: The target version (e.g., `v1.8.4`, `v0.5.0`, or `latest`)

Parse these from the user's invocation of `/update-go-dep <package> <version>`

## Workflow

Follow these steps **sequentially**:

### 1. Find all go.mod files

Use Glob to find all go.mod files in the repository:
- Pattern: `**/go.mod`
- This will find all go.mod files recursively

### 2. Update each go.mod file

For each directory containing a go.mod file:

a. Navigate to the directory (cd to the parent directory of go.mod)
b. Run `go get <package>@<version>` to update the dependency
c. Run `go mod tidy` to clean up the module dependencies

Use Bash commands to execute these steps. Run them sequentially in each directory.

**IMPORTANT**: 
- Some go.mod files might not use the specified package - this is OK, `go get` will simply do nothing
- If `go get` or `go mod tidy` fails in one directory, continue with the next directory
- Use `|| true` to prevent failures from stopping the process

### 3. Check what changed

After updating all go.mod files, run:
```bash
git status
```

To see which go.mod and go.sum files were modified.

### 4. Stage the changes

Add all modified go.mod and go.sum files:
```bash
git add -A '*.mod' '*.sum'
```

Or more specifically:
```bash
find . -name 'go.mod' -o -name 'go.sum' | xargs git add
```

### 5. Commit the changes

Create a commit with the message format:
```
chore(deps): update <package> to <version>
```

Use the Bash tool to execute:
```bash
git commit -m "chore(deps): update <package> to <version>"
```

### 6. Report completion

Tell the user:
- How many go.mod files were found
- Which files were modified (from git status)
- The commit message used
- Remind them they can push the changes with `git push`

## Example Usage

User invokes: `/update-go-dep github.com/stretchr/testify v1.9.0`

You should:
1. Find all go.mod files
2. For each directory with go.mod, run:
   - `go get github.com/stretchr/testify@v1.9.0`
   - `go mod tidy`
3. Stage changes: `git add -A '*.mod' '*.sum'`
4. Commit: `git commit -m "chore(deps): update github.com/stretchr/testify to v1.9.0"`
5. Report what was done

## Error Handling

- If no go.mod files are found, inform the user
- If git add finds no changes, inform the user that the package might not be used in this repository
- If the commit fails (e.g., nothing to commit), explain why
