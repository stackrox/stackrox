# update-go-dep

Update a Go package version across all go.mod files in the repository.

## Description

This skill automates the process of updating a Go dependency to a specific version across all `go.mod` files in the repository. It handles the update, runs `go mod tidy`, stages the changes, and creates a commit.

## Usage

```
/update-go-dep <package> <version>
```

### Arguments

- `<package>`: The full import path of the Go package to update (e.g., `github.com/stretchr/testify`)
- `<version>`: The target version (e.g., `v1.8.4`, `v0.5.0`, or `latest`)

**Note**: The `v` prefix is optional. The skill will add it automatically if needed.

## Examples

### Update to a specific version

```
/update-go-dep kubevirt.io/api v1.8.1
```

This will:
1. Find all go.mod files in the repository
2. Update `kubevirt.io/api` to version `v1.8.1` in each module
3. Run `go mod tidy` for each module
4. Stage all modified go.mod and go.sum files
5. Create a commit with message: `chore(deps): update kubevirt.io/api to v1.8.1`

### Update to latest version

```
/update-go-dep golang.org/x/crypto latest
```

This will update `golang.org/x/crypto` to the latest available version.

### Update a GitHub dependency

```
/update-go-dep github.com/stretchr/testify v1.9.0
```

## What it does

1. **Discovers all go.mod files** in the repository using the glob pattern `**/go.mod`
2. **Updates each module** by running:
   - `go get <package>@<version>`
   - `go mod tidy`
3. **Handles errors gracefully**: If a module doesn't use the specified package, the update is skipped
4. **Stages changes**: All modified `go.mod` and `go.sum` files are staged for commit
5. **Creates a commit** with a standardized message format:
   ```
   chore(deps): update <package> to <version>
   
   Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
   ```

## Output

After completion, the skill provides:
- Total number of go.mod files found
- List of files that were modified
- The commit hash and message
- Reminder to push changes if desired

## Error Handling

- If the version doesn't exist, `go get` will fail with an error message
- If a go.mod file doesn't use the specified package, the update is silently skipped
- If no changes are made (package not used anywhere), the commit will fail with a message explaining why

## Notes

- The skill updates **all** go.mod files in the repository, including those in subdirectories
- Changes are committed but **not pushed** automatically
- The skill follows the project's commit message convention: `chore(deps): ...`
- Pre-commit hooks will run if configured

## Related Skills

- `/bulk-update-go-dep` - Apply this update across multiple branches
