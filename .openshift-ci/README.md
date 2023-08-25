This folder supports OpenShift CI from https://github.com/openshift/release/tree/master/ci-operator/config/stackrox/stackrox

- [dispatch.sh](dispatch.sh) is the entrypoint for tests and builds (binary_build_commands, test_binary_build_commands).
- [build/](build) is for openshift/release image support.
- *.py provides some semantics useful to e2e and system tests e.g.:
  - create a cluster
  - run test
  - gather and examine state
  - teardown

## Workflow aliases for test/format/lint

```
alias osci='cdrox; cd .openshift-ci'
alias osci-format='osci; ack -f --python | entr black .'
alias osci-lint='osci; ack -f --python | entr pylint --rcfile .pylintrc *.py tests'
alias osci-test='osci; ack -f --python --shell | entr python -m unittest discover'
```
