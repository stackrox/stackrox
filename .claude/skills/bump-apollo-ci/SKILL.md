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
Ask user if not provided:
- OLD_VERSION (e.g., "0.5.4")
- NEW_VERSION (e.g., "0.5.5")
- **Also bump in openshift/release?** If yes, use the `bump-apollo-ci-openshift-release` skill after completing the stackrox repo bump

Note: OLD_VERSION can also be "whatever is currently in the config" if you want to auto-detect it.

### 2. Check repository is clean

Before making any changes, verify the repository has no uncommitted changes:
```bash
if ! git diff-index --quiet HEAD --; then
  echo "❌ Repository has uncommitted changes. Please commit or stash them before proceeding."
  git status --short
  exit 1
fi
```

If the repository is dirty, inform the user and stop. They must clean up the repository before proceeding.

### 3. Create branch from origin/master

Fetch latest changes and create new branch with format `apollo-ci-bump-NEW_VERSION`:
```bash
git fetch origin
BRANCH_NAME="apollo-ci-bump-NEW_VERSION"
git checkout -b "$BRANCH_NAME" origin/master
```

### 4. Find occurrences
```bash
git grep -n "apollo-ci.*OLD_VERSION\|stackrox-build-OLD_VERSION" -- \
  '*.yaml' '*.yml' '*.sh' '*.txt' '*.json' 'Dockerfile*' 'BUILD_IMAGE_VERSION'
```

### 5. Bulk replace
```bash
git ls-files '*.yaml' '*.yml' '*.sh' '*.txt' '*.json' 'Dockerfile*' | \
  xargs sed -i '' 's/apollo-ci:\(stackrox-test\|scanner-test\|stackrox-ui-test\|stackrox-build\)-OLD_VERSION/apollo-ci:\1-NEW_VERSION/g'
```

### 6. Update BUILD_IMAGE_VERSION
```bash
echo "stackrox-build-NEW_VERSION" > BUILD_IMAGE_VERSION
```

### 7. Verify zero old references
```bash
git grep -c "apollo-ci.*OLD_VERSION\|stackrox-build-OLD_VERSION" -- \
  '*.yaml' '*.yml' '*.sh' '*.txt' '*.json' 'Dockerfile*' 'BUILD_IMAGE_VERSION' | wc -l
# Expected: 0
```

### 8. Review changes
```bash
git diff --stat
# Expected: ~14-16 files changed
```

### 9. Commit changes

Add and commit all changes:
```bash
git add -A
git commit -m "$(cat <<'EOF'
Bump apollo-ci from OLD_VERSION to NEW_VERSION

Updates all apollo-ci container image references from version OLD_VERSION to NEW_VERSION.
This includes stackrox-test, scanner-test, stackrox-ui-test, and stackrox-build variants.
EOF
)"
```

### 10. Inform user

Display the branch name and next steps:
```
✅ Changes committed to branch: BRANCH_NAME

Branch: BRANCH_NAME
Files changed: N files

Next steps:
1. Review the changes: git show
2. Push the branch: git push origin BRANCH_NAME
3. Create a PR in the stackrox/stackrox repository
4. If you selected "Also bump in openshift/release", run the bump-apollo-ci-openshift-release skill next
```

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
