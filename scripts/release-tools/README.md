# DEPRECATION NOTICE

:warning: The scripts are deprecated and will be removed on October 10, 2022. The functionality is now implemented by GitHub Actions.

# Upstream Release Tools

This directory contains a selection of scripts that automate selected mundane manual steps (a.k.a toil) in the upstream release process.

:warning: The scripts are bleeding-edge - there is no guarantee that they would work for you the way they work for me. Test coverage is currently not provided.:warning:

## Meta

This document describes the intention behind the scripts in this folder.
We acknowledge that different developers may run some parts of the release differently, use different tools, and prefer different programming languages.
Some parts could have been definitely done better than they currently are, however the balance of time, effort, and usability at the time of the release resulted exactly in this state.
Thus, it is fully okay to:

- use these scripts partially or not at all,
- replace them with something better,
- add functionalities, bugfixes, or tests.

## Prerequisites

The scripts described here require the following env variables to exist and have correct values:

```shell
export RELEASE=3.69
export RC_NUMBER=1

export INFRA_TOKEN="xxx" # generate under https://infra.stackrox.com/
export JIRA_TOKEN="yyy" # generate under https://issues.redhat.com/secure/ViewProfile.jspa?selectedTab=com.atlassian.pats.pats-plugin:jira-user-personal-access-tokens
```

I recommend placing these `export`s in `~/.config/stackrox` and sourcing it in `~/.{bash,zsh,your_shell}rc`.

### Prompt

I highly recommend to display the values of `$RELEASE` and `$RC_NUMBER` in the shell prompt to avoid mistakes.

Example prompt configuration for [Starship](https://starship.rs/) users (`cat ~/.config/starship.toml`):

```toml
[env_var.RELEASE]
default = ""
format = "RELEASE=[$env_value]($style) "
style = "bold red"

[env_var.RC_NUMBER]
default = ""
format = "RC_NUMBER=[$env_value]($style) "
style = "bold red"
```

### Functions placed in `.{bash,zsh}rc`

The following shortcuts extremely useful for frequent edits of the config files (names were chosen arbitrarily).

Function `je` opens the most important config files and sources the changes afterwards.
Function `re` does only the sourcing and is used to apply the changes done by `je` to other terminal windows.

```bash
je() {
  vim ~/.zshrc ~/.config/rox
  . ~/.zshrc
}

re() {
  . ~/.zshrc
}
```

## Cluster script

The `cluster.sh` script automates the following tasks:

1. Update and merge of kubeconfigs
2. Creating OpenShift demo cluster for RCs
3. Generating Slack message that shares accesses to the demo cluster

### Merging of kubeconfigs

This script helps keep a list of kubeconfigs up to date and with unified naming.
It expects that each cluster has its own directory named `<cluster_name>` in the `artifacts/` folder.
An example may look like this:

```text
$ tree -L 1 artifacts
artifacts
├── gke_srox
├── os4-9-demo-3-68-rc5
├── os4-9-demo-3-68-rc7
├── os4-9-demo-3-68-rc8
├── test1
└── test2
```

The script iterates over the list of folders inside the `artifacts/` folder and executes `infractl artifacts <cluster_name> -d artifacts/<cluster_name>` to refresh the artifacts for each cluster.
Clusters that do not exist anymore are skipped.

Next, for each of the freshly redownloaded artifacts, the following operations are conducted:

- Rename `kubectl` user to `admin-<cluster_name>` (to avoid conflicts)
- Rename `kubectl` context to `ctx-<cluster_name>` (to unify naming)

Finally, all kubeconfig files (`artifacts/<cluster_name>/kubeconfig`) are merged into one and written to the standard location defined in `KUBECONFIG` env variable.
A standard location `$HOME/.kube/config` is used if `KUBECONFIG` is empty or contains multiple concatenated paths.
Before overwriting `$KUBECONFIG` a backup is made in `${KUBECONFIG}.bak`.
If a backup already exists, it will not be overwritten.

### Creating OpenShift demo cluster for RCs

The goal of OpenShift cluster creation is to orchestrate the following steps:

1. Creating cluster using `infractl` (flavor `openshift-4-demo`)
2. Waiting util it is ready
3. Downloading infra artifacts.
4. Making kubeconfig items unique (e.g., username, cluster name) by leveraging _Merging of kubeconfigs_.
5. Upgrading the ACS deployment to the desired version (flavor `openshift-4-demo` is deployed with an older version of ACS, so we need to upgrade all deployments to the current RC version)

### Generating Slack message sharing access to the Openshift demo cluster

This script generates Slack message to share the links and credentials for accessing the Openshift demo cluster.
It works only for OpenShift clusters and requires the cluster artifacts to be downloaded into `artifacts/os4-9-demo-${RELEASE//./-}-rc${RC_NUMBER}`

## Jira script

The `jira.sh` script is a wrapper for running custom JQL queries against Red Hat Jira.
It generating Slack messages (to manually paster) to inform colleagues about current status of tickets targeted for the release.

The operations include:

1. _"Show tickets with FixVersions that are not done yet"_: Informing about tickets that are planned to be included in the release but are not done yet
2. _"Show tickets that need to be verified on demo cluster"_: Informing about tickets that are already included in the release but have been not manually tested yet.

The goal of the first message is to remind people that the release is approaching soon and they should slowly wrap-up work on a given ticket or consider it for the next release.

The second message aims at ensuring that last-minute merged features or fixes work correctly on the rc-demo clusters.

### Jira-to-Slack username mapping

Mapping of RedHar Jira names to Slack handles is done fully manually.
Not all colleagues have been added to the mapping (some might have also changed their Slack-handles), so the future release-engineers are kindly asked to update the mapping in `name2slack` function if possible.

### Marking tickets are verified

When running _"Show tickets that need to be verified on demo cluster"_, the release-engineer would normally ping a lot of people on Slack and get (hopefully) a lot of confirmations from them.
Marking that a given ticket has been manually verified (and the responsible colleague should not be pinged again) is done manually in the code - function `is_ticket_checked`.
The mapping consist of bash associative array, where the key is a Jira ticket ID (e.g., `ROX-1234`) and the value is one of the following strings `verified`, `N/A`.

The value `verified` is used to mark that a given function has been manually verified by the author who implemented it (they know the best how it should work).
The `N/A` is used to mark a feature that is not verifiable on the demo cluster - for example: CI changes, unit-tests. If the implementer does not report anything on a ticket, then the release-engineer can decide on their own to verify the ticket with `N/A` by, for example, looking at the code.

## Ideas for future work

1. Introduce new Jira field for marking tickets are verified (or `n/a`). This would allow to completely drop the manually maintained mapping in `is_ticket_checked`.
2. Modify `infractl artifacts` to produce artifacts in unified format for OpenShift and GKE cluster. This would allow to add support for GKE in `clusters.sh`.
3. Modify `infractl artifacts` to produce kubeconfig files with unique and systematically-named: usernames, cluster names, context names (e.g., `admin-${CLUSTER_NAME}`, `cluster-${CLUSTER_NAME}`, `ctx-${CLUSTER_NAME}`). This would allow to drastically simplify parts of `clusters.sh`
