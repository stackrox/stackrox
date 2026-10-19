Investigate this StackRox CI failure by launching the `stackrox-ci-failure-investigator`
subagent (via the Agent tool). Pass it the input below — a Prow build ID or URL, a failing
job name, a ROX-XXXXX ticket, or pasted logs — and have it download and read the CI
artifacts (Prow step logs plus the collected k8s service logs, pod descriptions, events, and
node capacity) before forming a hypothesis.

Do not guess at a cause or attribute the failure to flakiness or a version being behind until
the artifacts rule everything else out. Follow the causal chain to its origin — a failed
check is usually a downstream symptom. Do not write code until the root cause is confirmed
with the user.

$ARGUMENTS
