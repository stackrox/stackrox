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
