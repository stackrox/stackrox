These files serve as a migration path to run CI using OpenShift CI. OpenShift CI
is configured to run test jobs against stackrox/rox-openshift-ci-mirror where
`migrate.sh` is used to invoke the job in the context of a matching branch in
stackrox/stackrox.

OpenShift CI: https://github.com/openshift/release
Configuration for StackRox: https://github.com/openshift/release/tree/master/ci-operator/config/stackrox
