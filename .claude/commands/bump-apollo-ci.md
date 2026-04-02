---
name: bump-apollo-ci
description: Bump all apollo-ci container images to a new version
---

Bump all apollo-ci container image references in the stackrox/stackrox repository.

CRITICAL: Follow the comprehensive checklist in `.claude/skills/bump-apollo-ci/SKILL.md` to ensure ALL files are updated.

Ask the user:
- OLD_VERSION (current version to replace, e.g., "0.5.4")
- NEW_VERSION (target version, e.g., "0.5.5")
- Also bump in openshift/release repository? If yes, use `bump-apollo-ci-openshift-release` skill after completing stackrox bump

Then systematically update ALL occurrences following the skill documentation:
1. Use pattern-based search and replace for standard files
2. Explicitly update special cases (especially BUILD_IMAGE_VERSION)
3. Verify zero old references remain
4. Show git diff summary
