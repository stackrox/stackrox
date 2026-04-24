# bulk-update-go-dep

Update a Go package version across multiple branches by creating child branches for each target.

## Description

This skill automates the process of updating a Go dependency across multiple release branches. For each target branch, it creates a new child branch, applies the dependency update using the `update-go-dep` logic, and commits the changes.

This is especially useful for maintaining consistency across multiple release branches when updating dependencies.

## Usage

```
/bulk-update-go-dep <package> <version>
```

### Arguments

- `<package>`: The full import path of the Go package to update (e.g., `github.com/stretchr/testify`)
- `<version>`: The target version (e.g., `v1.8.4`, `v0.5.0`, or `latest`)

**Note**: The skill will prompt you interactively for the list of target branches.

## Workflow

1. **Prompts for target branches**: You'll be asked which branches to update (e.g., `master, release-4.9, release-4.8`)
2. **Confirms the plan**: Shows you what will happen before proceeding
3. **For each branch**:
   - Fetches the latest from origin
   - Creates a new child branch: `<username>/<target-branch>/bump_<dependency>_to_<version>`
   - Applies the dependency update (updates all go.mod files)
   - Commits the changes
4. **Returns to original branch**: Restores your working state

## Examples

### Update across multiple release branches

```
/bulk-update-go-dep kubevirt.io/api v1.8.1
```

**Interactive prompts**:
```
Which branches should I update?
> release-4.9, release-4.8, release-4.10

Proceed with creating these branches and applying updates?
> Yes, proceed
```

**Result**: Creates three branches:
- `yann-brillouet/release-4.9/bump_api_to_v1.8.1`
- `yann-brillouet/release-4.8/bump_api_to_v1.8.1`
- `yann-brillouet/release-4.10/bump_api_to_v1.8.1`

Each branch contains the dependency update committed and ready to push.

### Update across master and release branches

```
/bulk-update-go-dep golang.org/x/crypto v0.17.0
```

**Interactive prompts**:
```
Which branches should I update?
> master, release-4.9, release-4.8

Proceed?
> Yes
```

## Branch Naming Convention

Branches are created with the following pattern:
```
<username>/<target-branch>/bump_<dependency-name>_to_<version>
```

Where:
- `<username>`: Your git username, lowercased with spaces replaced by hyphens (e.g., `yann-brillouet`)
- `<target-branch>`: The target branch name (e.g., `release-4.9`)
- `<dependency-name>`: Last component of the package path (e.g., `api` from `kubevirt.io/api`)
- `<version>`: The target version (e.g., `v1.8.1`)

### Examples

| Package | Target Branch | Resulting Branch Name |
|---------|---------------|----------------------|
| `kubevirt.io/api` | `master` | `yann-brillouet/master/bump_api_to_v1.8.1` |
| `github.com/stretchr/testify` | `release-4.9` | `yann-brillouet/release-4.9/bump_testify_to_v1.9.0` |
| `golang.org/x/crypto` | `release-4.8` | `yann-brillouet/release-4.8/bump_crypto_to_v0.17.0` |
| `k8s.io/client-go` | `master` | `yann-brillouet/master/bump_client_go_to_v0.29.0` |

## Output

After completion, you'll receive:
- A summary of successful updates (branch names and commit hashes)
- A list of any failures (with error details)
- Instructions for pushing the branches
- Suggestions for creating pull requests

## Next Steps

### Push the branches

```bash
git push -u origin yann-brillouet/release-4.9/bump_api_to_v1.8.1
git push -u origin yann-brillouet/release-4.8/bump_api_to_v1.8.1
git push -u origin yann-brillouet/release-4.10/bump_api_to_v1.8.1
```

Or push all at once:
```bash
git push -u origin 'yann-brillouet/*/bump_api_to_v1.8.1'
```

### Create Pull Requests

Using `gh` CLI:
```bash
gh pr create --base release-4.9 \
  --head yann-brillouet/release-4.9/bump_api_to_v1.8.1 \
  --title "chore(deps): update kubevirt.io/api to v1.8.1" \
  --body "Updates kubevirt.io/api to v1.8.1"
```

Or use the GitHub web interface to create PRs from each branch.

## Error Handling

The skill handles several error scenarios gracefully:

- **Branch doesn't exist**: Skips the branch and reports the error
- **Dependency not used**: If a target branch doesn't use the specified package, the branch is created but no changes are committed (it's then deleted)
- **Branch already exists**: Reports the conflict and skips to the next branch
- **Update fails**: Reports the error and continues with remaining branches
- **Stashed changes**: Automatically stashes local changes before starting and restores them when done

## Notes

- Branches are created locally but **not pushed** automatically
- Each branch gets its own commit with the standardized message format
- The skill preserves your working directory state (stashes and restores changes)
- You're returned to your original branch when the skill completes
- Pre-commit hooks run for each commit if configured

## Use Cases

### Security patch across all supported releases

When a security vulnerability is discovered in a dependency, you need to update it across all active release branches:

```
/bulk-update-go-dep golang.org/x/crypto v0.17.1
> master, release-4.10, release-4.9, release-4.8
```

### Standardizing dependency versions

Ensure all release branches use the same version of a critical dependency:

```
/bulk-update-go-dep k8s.io/client-go v0.29.0
> release-4.10, release-4.9, release-4.8
```

### Quarterly dependency updates

As part of routine maintenance, update dependencies across all active branches:

```
/bulk-update-go-dep github.com/prometheus/client_golang v1.18.0
> master, release-4.10, release-4.9
```

## Related Skills

- `/update-go-dep` - Update a dependency on the current branch only

## Tips

- **Review before pushing**: Inspect each branch locally before pushing to ensure the updates are correct
- **Batch PR creation**: Use a script to create all PRs at once if you have many branches
- **Testing**: Consider running tests on each branch before creating PRs
- **Backport order**: Push and merge from oldest to newest release branch to catch any version-specific issues early
