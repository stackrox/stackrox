---
name: bump-apollo-ci
description: ALWAYS use when user asks to "bump apollo-ci", "update apollo-ci", or "upgrade apollo-ci" to any version. Updates all apollo-ci/stackrox-test/scanner-test/stackrox-ui-test/stackrox-build container image references across this repository.
user-invocable: true
---

# Bump Apollo CI Images

Updates all apollo-ci container image references using git-aware commands that only modify tracked files on the current branch.

## Apollo CI Image Variants

Four variants exist:
- `stackrox-test` - Main test container
- `scanner-test` - Scanner-specific tests
- `stackrox-ui-test` - UI-specific tests
- `stackrox-build` - Build container (special format: see BUILD_IMAGE_VERSION below)

## What Gets Updated

### Pattern-Based (automatic via regex)

- GitHub Actions workflows: `image: quay.io/stackrox-io/apollo-ci:(variant)-VERSION`
- Dockerfiles: `FROM quay.io/stackrox-io/apollo-ci:(variant)-VERSION`
- Dev containers: `"image":"quay.io/stackrox-io/apollo-ci:(variant)-VERSION"`
- Shell scripts: `quay.io/stackrox-io/apollo-ci:(variant)-VERSION`
- Comments: Any mention of `apollo-ci:*-VERSION` or `stackrox-build-VERSION`

### Special Cases (explicit handling)

**`BUILD_IMAGE_VERSION`** - CRITICAL! Contains only `stackrox-build-X.X.X` (no `quay.io/stackrox-io/apollo-ci:` prefix)

## Procedure

### 1. Get versions and scope

**IMPORTANT**: Use `AskUserQuestion` to gather the following parameters interactively with clickable options:

**Question 1 - Old Version:**
- Options: `Auto-detect (Recommended)` (Automatically find current version from BUILD_IMAGE_VERSION) | `Specify version` (Manually specify old version like "0.5.4")

**Question 2 - New Version:**
- Always query the user for this one -- no auto-detection and/or pre-filled-in suggestions.
- User must provide version in format like "0.5.5"

**Question 3 - Also bump in openshift/release?**
- Options: `Yes` (After completing stackrox repo bump, also update openshift/release) | `No` (Only update stackrox repository)

After gathering all parameters via AskUserQuestion, proceed with the following steps.

### 2. Auto-detect old version (if selected)

If user selected "Auto-detect" for old version, extract it from BUILD_IMAGE_VERSION:
```bash
cat BUILD_IMAGE_VERSION
# Expected format: stackrox-build-X.X.X
# Extract version number: X.X.X
```

Parse the version from the output (e.g., if output is `stackrox-build-0.5.4`, OLD_VERSION is `0.5.4`).

### 3. Check repository is clean

Before making any changes, verify the repository has no uncommitted changes:
```bash
if ! git diff-index --quiet HEAD --; then
  echo "❌ Repository has uncommitted changes. Please commit or stash them before proceeding."
  git status --short
  exit 1
fi
```

If the repository is dirty, inform the user and stop. They must clean up the repository before proceeding.

### 4. Create branch from origin/master

Fetch latest changes and create new branch with format `apollo-ci-bump-NEW_VERSION`:
```bash
git fetch origin
BRANCH_NAME="apollo-ci-bump-NEW_VERSION"
git checkout -b "$BRANCH_NAME" origin/master
```

### 5. Find occurrences
```bash
git grep -n "apollo-ci.*OLD_VERSION\|stackrox-build-OLD_VERSION" -- \
  '*.yaml' '*.yml' '*.sh' '*.txt' '*.json' 'Dockerfile*' 'BUILD_IMAGE_VERSION'
```

### 6. Bulk replace
```bash
git ls-files '*.yaml' '*.yml' '*.sh' '*.txt' '*.json' 'Dockerfile*' | \
  xargs sed -i.bak 's/apollo-ci:\(stackrox-test\|scanner-test\|stackrox-ui-test\|stackrox-build\)-OLD_VERSION/apollo-ci:\1-NEW_VERSION/g'

# Clean up backup files
find . -name '*.bak' -type f -delete
```

### 7. Update BUILD_IMAGE_VERSION
```bash
echo "stackrox-build-NEW_VERSION" > BUILD_IMAGE_VERSION
```

### 8. Verify zero old references
```bash
git grep -c "apollo-ci.*OLD_VERSION\|stackrox-build-OLD_VERSION" -- \
  '*.yaml' '*.yml' '*.sh' '*.txt' '*.json' 'Dockerfile*' 'BUILD_IMAGE_VERSION' | wc -l
# Expected: 0
```

### 9. Review changes
```bash
git diff --stat
# Expected: ~14-16 files changed
```

### 10. Commit changes

Add only modified tracked files and commit (git add -u stages only tracked files, avoiding untracked files):
```bash
git add -u
git commit -m "$(cat <<'EOF'
Bump apollo-ci from OLD_VERSION to NEW_VERSION

Updates all apollo-ci container image references from version OLD_VERSION to NEW_VERSION.
This includes stackrox-test, scanner-test, stackrox-ui-test, and stackrox-build variants.
EOF
)"
```

### 11. Inform user

Display the branch name and next steps:
```text
✅ Successfully bumped apollo-ci to version NEW_VERSION!

**Summary:**
- Branch: BRANCH_NAME
- Files changed: N files
- Version bump: OLD_VERSION → NEW_VERSION

**Next steps:**
1. Review the changes: git show
2. Push the branch: git push origin BRANCH_NAME
3. Create a PR in the stackrox/stackrox repository
```

If the user selected "Yes" for "Also bump in openshift/release", invoke the `bump-apollo-ci-openshift-release` skill next.

## Important

- **Scope:** Only git-tracked files on current branch. Multiple checkouts must be updated independently.
- **All variants must match:** All four variants use the same version number.
- **BUILD_IMAGE_VERSION:** Easy to forget! No prefix, just `stackrox-build-X.X.X`.
- **Clean repository required:** The repository must have no uncommitted changes before starting.
- **Branch naming:** New branches follow the pattern `apollo-ci-bump-NEW_VERSION`.

## Troubleshooting

If files are missed, check:
```bash
git grep "OLD_VERSION"  # Find remaining references
git status              # Check if files are tracked
```
