Overview
--------

CURRENTLY WORKS ONLY FOR OSD.

This directory contains codified steps to setup for local testing.

Useful for troubleshooting 'qa-test-backend' test failures.

This is just an example. Edit scripts and test targets as needed.


Example Development Workflow
----------------------------

1. Setup local macos system for local testing against a remote cluster
   (jre + gradle + groovy + spock + fabric8)

```
cd $GOPATH/src/github.com/stackrox/stackrox/qa-tests-backend
./step1-setup-macos.sh
```

2. Use https://infra.rox.systems/ to provision a remote cluster

3. Configure local kubeconfig, kubecontext, and verify you have access to the cluster

```
set_temp_kubeconfig_from_paste_buffer () {
  export KUBECONFIG=/tmp/kubeconfig
  pbpaste > "$KUBECONFIG"
  chmod 600 "$KUBECONFIG"
  kubectl get no
}
```

4. Install ACS on the remote cluster

```
cd $GOPATH/src/github.com/stackrox/stackrox/qa-tests-backend
./step2-install-acs.sh
```

5. Build and run tests locally

```
cd $GOPATH/src/github.com/stackrox/stackrox/qa-tests-backend
vim src/test/groovy/LocalQaPropsTest.groovy
gradle build -x test
./step3-run-qa-tests-backend.sh
```
