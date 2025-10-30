# Apollo CI Image Bump Tool

Automates updating apollo-ci image references across the repository.

## Usage

```bash
./tools/bump-apollo-ci/bump.sh [--dry-run] --branch <branch_name> <new_version>
```

Examples:

```bash
./tools/bump-apollo-ci/bump.sh --branch chore-bump-apollo-ci-0.4.10 0.4.10
./tools/bump-apollo-ci/bump.sh --branch chore-bump-apollo-ci-0.4.12-6-gf8cb14d205 0.4.12-6-gf8cb14d205
./tools/bump-apollo-ci/bump.sh --branch chore-bump-apollo-ci-0.4.12-6-gf8cb14d205 0.4.13
./tools/bump-apollo-ci/bump.sh --dry-run --branch chore-bump-apollo-ci-0.4.10 0.4.10
```

The version can be in standard format (e.g., `0.4.10`) or git describe format (e.g., `0.4.12-6-gf8cb14d205`).

Use `--dry-run` to preview changes without creating a branch, commit, or PR.

The `--branch <branch_name>` parameter is required. If the branch exists remotely, it will be checked out and updated. If it doesn't exist, it will be created.

## Typical Workflow

1. **Initial bump with git describe format:**

   ```bash
   ./tools/bump-apollo-ci/bump.sh --branch chore-bump-apollo-ci-0.4.12-6-gf8cb14d205 0.4.12-6-gf8cb14d205
   ```

   This creates a new branch `chore-bump-apollo-ci-0.4.12-6-gf8cb14d205` and PR with the git describe version.

2. **Later, bump to semver on the same branch:**

   ```bash
   ./tools/bump-apollo-ci/bump.sh --branch chore-bump-apollo-ci-0.4.12-6-gf8cb14d205 0.4.13
   ```

   This checks out the existing branch, updates all references to `0.4.13`, pushes to the same branch, and updates the PR title to match the new version.

## What it does

1. Pulls latest master
2. Finds all apollo-ci image references
3. Updates BUILD_IMAGE_VERSION file
4. Updates all direct image references (stackrox-test, scanner-test, stackrox-ui-test variants)
5. Lists comment-only references (not updated automatically)
6. Creates the specified branch (if it doesn't exist) or checks out the existing branch (if it exists remotely)
7. Commits and pushes changes
8. Opens a draft Pull Request (or updates existing PR title if branch already has one)

## Prerequisites

- `gh` CLI tool configured for creating PRs
- Write access to the repository
