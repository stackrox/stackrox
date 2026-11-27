# Prow retest

This is a utility tool to trigger re-runs of Prow (postsubmit) jobs, because it is not possible through the Prow UI.

## Usage

In this directory, run the CLI:

```
go run . \
--job-name <job name> \
--base-ref <branch name> \
--base-sha <40-digit SHA>
```

The tool will prompt you to login to the app.ci cluster. A browser will open for you to complete the login with your RHSSO. The auth token is then obtained automatically via `oc whoami -t`.
