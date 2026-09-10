# Release Helpers and Configuration

This directory contains a selection of helper scripts and configuration for the StackRox release process.

:warning: The scripts are bleeding-edge - there is no guarantee that they would work for you the way they work for me. Test coverage is currently not provided.:warning:

## Contents

### Long-running cluster

For the purposes of configuring workloads on the long-running cluster (created as part of the release process), there are directories containing configuration:

- `kube-burner-configs`
- `long-running-cluster`

### Cluster access

To access long-running and upgrade clusters created during the release process, there are two scripts and a library of shared functions.

### Merge Window

When on a release branches, the `MERGE_WINDOW` file describes whether or not changes are allowed to be merged to the respective release branch.
It contains either "OPEN" or "CLOSED".
These states are set through release automation.
Follow the documented process in case you need to bypass of a closed merge window.
