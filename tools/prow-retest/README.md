# Prow retest

This is a utility tool to trigger re-runs of Prow (postsubmit) jobs, because it is not possible through the Prow UI.

## Usage

In this directory, perform the following steps:

1. Login to app.ci (Prow/OSCI) cluster:

```
oc login --server=https://api.ci.l2s4.p1.openshiftapps.com:6443 --web
````

2. A browser will open with the login page of the app.ci cluster. Login with your RHSSO.

3. Then run the CLI:

```
export PROW_TOKEN="$(oc whoami -t)"
go run . \
--job-name <job name> \
--base-ref <branch name> \
--base-sha <40-digit SHA>
```
