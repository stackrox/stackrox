---
name: bump-apollo-ci-openshift-release
description: Bump apollo-ci image tags in the openshift/release repository CI operator configs for specific StackRox/Scanner branches
user-invocable: true
---

# Bump Apollo CI in openshift/release

Updates apollo-ci `tag:` fields in ci-operator configs for specific project/branch combinations.

## File Pattern

Files: `ci-operator/config/stackrox/{PROJECT}/{PROJECT}-{PROJECT}-{BRANCH}*.yaml`

Examples: `stackrox-stackrox-master*.yaml` (master + OCP variants), `stackrox-scanner-release-2.36.yaml`

## Procedure

### 1. Get information

**IMPORTANT**: Use `AskUserQuestion` to gather the following parameters interactively with clickable options:

**Question 1 - Project:**
- Options: `stackrox` (Main StackRox platform repository) | `scanner` (StackRox Scanner repository)

**Question 2 - Branch:**
- Options: `master` (Main development branch) | `release-4.7` | `release-2.36` | `nightlies`
- Always include an "Other" option for custom branches

**Question 3 - New Version:**
- Always query the user for this one -- no auto-detection and/or pre-filled-in suggestions.

**Question 4 - Old Version:**
- Options: `Auto-detect (Recommended)` (Automatically find current version in config files) | `Specify version` (Manually specify old version)

**Question 5 - Release Repo Path:**
- Ask for path to openshift/release checkout
- Suggest common locations if detectable

After gathering all parameters via AskUserQuestion, proceed with the following steps.

### 2. Check repository is clean

Before making any changes, verify the repository has no uncommitted changes:
```bash
if ! git -C RELEASE_REPO_PATH diff-index --quiet HEAD --; then
  echo "❌ Repository has uncommitted changes. Please commit or stash them before proceeding."
  git -C RELEASE_REPO_PATH status --short
  exit 1
fi
```

If the repository is dirty, inform the user and stop. They must clean up the repository before proceeding.

### 3. Create branch from origin/main

Fetch latest changes and create new branch:
```bash
git -C RELEASE_REPO_PATH fetch origin
BRANCH_NAME="PROJECT-BRANCH-apollo-ci-bump-NEW_VERSION"
git -C RELEASE_REPO_PATH checkout -b "$BRANCH_NAME" origin/main
```

### 4. Show matching files and detect current versions
```bash
PATTERN="ci-operator/config/stackrox/PROJECT/PROJECT-PROJECT-BRANCH*.yaml"
git -C RELEASE_REPO_PATH ls-files "$PATTERN"

# Show what versions currently exist across all matching files
git -C RELEASE_REPO_PATH grep -h "tag: \(stackrox-test\|scanner-test\|stackrox-ui-test\)-" -- "$PATTERN" | \
  sed 's/.*tag: //' | sort -u
```
**Important**: Different files may have different versions. The bulk replace will update ALL versions to NEW_VERSION.

### 5. Bulk replace (handles mixed versions)

**If user selected "Auto-detect" for old version:**
Replace ANY version pattern with the new version:
```bash
git -C RELEASE_REPO_PATH ls-files "$PATTERN" | \
  xargs -I {} sed -i '' -E \
    's/tag: (stackrox-test|scanner-test|stackrox-ui-test)-[0-9]+\.[0-9]+\.[0-9]+$/tag: \1-NEW_VERSION/' \
    "RELEASE_REPO_PATH/{}"
```

**If user specified a specific OLD_VERSION:**
Replace only that specific version:
```bash
git -C RELEASE_REPO_PATH ls-files "$PATTERN" | \
  xargs -I {} sed -i '' -E \
    's/tag: (stackrox-test|scanner-test|stackrox-ui-test)-OLD_VERSION$/tag: \1-NEW_VERSION/' \
    "RELEASE_REPO_PATH/{}"
```

### 6. Verify replacement succeeded
```bash
# Show what versions remain after replacement
git -C RELEASE_REPO_PATH grep -h "tag: \(stackrox-test\|scanner-test\|stackrox-ui-test\)-" -- "$PATTERN" | \
  sed 's/.*tag: //' | sort -u
# Expected: Only NEW_VERSION should appear

# If a specific OLD_VERSION was targeted, verify it's gone:
git -C RELEASE_REPO_PATH grep -c "tag: .*-OLD_VERSION" -- "$PATTERN" 2>/dev/null | wc -l
# Expected: 0
```

### 7. Normalize configs
Run to ensure configs are properly formatted (note: this may show errors for other projects' configs - ignore those):
```bash
make -C RELEASE_REPO_PATH ci-operator-config
```

### 8. Review changes
```bash
git -C RELEASE_REPO_PATH diff --stat
# Expected: Only ci-operator/config/ files changed (jobs/ unchanged for image bumps)
```

### 9. Commit changes

Add and commit the changes:
```bash
git -C RELEASE_REPO_PATH add ci-operator/config/stackrox/PROJECT/
git -C RELEASE_REPO_PATH commit -m "$(cat <<'EOF'
Bump StackRox apollo-ci for PROJECT/BRANCH from OLD_VERSION to NEW_VERSION

Updates apollo-ci image tags in CI operator configs for PROJECT BRANCH.
EOF
)"
```

### 10. Inform user

Display a summary and exit:
```text
✅ Successfully bumped apollo-ci to version NEW_VERSION!

**Summary:**
- Repository: RELEASE_REPO_PATH
- Branch: BRANCH_NAME
- Files changed: N files
- Version bump: OLD_VERSION → NEW_VERSION
- Commit: COMMIT_HASH
```

## Important

- **Normalize configs:** Always run `make ci-operator-config` after editing (may show errors for other projects - ignore those)
- **Variants:** stackrox-test, scanner-test, stackrox-ui-test (no stackrox-build)
- **Pattern includes OCP variants:** `BRANCH*.yaml` matches all (e.g., master__ocp-4-18)
- **Clean repository required:** The repository must have no uncommitted changes before starting
- **Branch naming:** New branches follow the pattern `PROJECT-BRANCH-apollo-ci-bump-NEW_VERSION`
