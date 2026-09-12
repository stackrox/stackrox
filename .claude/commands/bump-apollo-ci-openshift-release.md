---
name: bump-apollo-ci-openshift-release
description: Bump apollo-ci in openshift/release for specific project/branch
---

Bump apollo-ci image tags in the openshift/release repository for a specific StackRox or Scanner branch.

Follow `.claude/skills/bump-apollo-ci-openshift-release/SKILL.md` for detailed instructions.

Ask the user:
- PROJECT: `stackrox` or `scanner`
- BRANCH: `master`, `release-4.7`, `release-2.36`, `nightlies`, etc.
- OLD_VERSION (e.g., "0.4.8")
- NEW_VERSION (e.g., "0.5.4")
- RELEASE_REPO_PATH: Path to openshift/release checkout

This skill updates only files matching the pattern: `PROJECT-PROJECT-BRANCH*.yaml`
