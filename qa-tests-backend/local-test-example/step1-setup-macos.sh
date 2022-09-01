#!/bin/bash
set -eu
source "local-test-example/common.sh"
source "local-test-example/config.sh"

function install_golang_1_17_12 {
  # Default path to 'go' is /usr/local/go/bin/go
  # But ACS requires 1.17.12 which will be installed at $HOME/go/bin/go1.17.12

  if ! go version | grep 'go1.17.12' &> /dev/null; then
    echo "Installing go v1.17.12"
    go install golang.org/dl/go1.17.12@latest
    ln -sf "$HOME/go/bin/go1.17.12" "$HOME/go/bin/go"
    path_prepend "$HOME/go/bin/"
  fi

  echo "go location: $(which go)"
  echo "go version: $(go version)"
}

function install_gradle {
  if ! command -v gradle; then
    brew install gradle
  fi
  gradle --version
}
function install_helm {
  if ! command -v helm; then
    brew install helm
  fi
  helm version
}

function install_openjdk11 {
  hr
  brew ls --versions openjdk@11 || brew install openjdk@11
  export JAVA_HOME="/usr/local/Cellar/openjdk@11/11.0.12"
  echo "JAVA_HOME is [$JAVA_HOME]"
  java --version
  javac --version
}

function install_rocksdb {
  hr
  # Install dependencies
  brew ls --versions snappy || brew install snappy
  brew ls --versions lz4 || brew install lz4
  brew ls --versions zstd || brew install zstd

  # Clone rocksdb repo
  if ! [[ -d "$HOME/go/src/github.com/facebook/rocksdb" ]]; then
    mkdir -p ~/go/src/github.com/facebook/
    cd ~/go/src/github.com/facebook/
    git clone https://github.com/facebook/rocksdb.git
  fi

  # Build rocksdb
  if ! ls /usr/local/lib/librocksdb.*.dylib &>/dev/null; then
    cd ~/go/src/github.com/facebook/rocksdb
    git checkout v6.15.5
    make shared_lib install-shared
  fi

  # Validate use of rocksdb -- BROKEN
  ### cd $GOPATH/src/github.com/stackrox/stackrox
  ### go get github.com/stackrox/rox/central/vulnerabilityrequest/manager
  ### go test github.com/stackrox/rox/central/vulnerabilityrequest/manager -count=1
}

function test_stackrox_workflow_roxhelp {
  hr
  echo "test_stackrox_workflow_roxhelp()"
  "$GOPATH/src/github.com/stackrox/workflow/bin/roxhelp" --list-all
}

function build_stackrox {
  hr
  cd "$GOPATH/src/github.com/stackrox/stackrox/"
  echo "SKIPPING STACKROX BUILD -- NOT WORKING AND NOT NEEDED"
  #export STORAGE=pvc
  #export SKIP_UI_BUILD=1
  #make install-dev-tools
  #make proto-generated-srcs
  #make image  # BROKEN???
}

function setup_roxctl {
  local target url
  target="$GOPATH/bin/roxctl"
  if is_linux; then
    url="https://mirror.openshift.com/pub/rhacs/assets/$MAIN_IMAGE_TAG/bin/linux/roxctl"
  elif is_darwin; then
    url="https://mirror.openshift.com/pub/rhacs/assets/$MAIN_IMAGE_TAG/bin/darwin/roxctl"
  else
    error "Unknown OS [$(uname)]"
  fi
  hr
  (set -x; curl --output "$target" --silent "$url"; chmod +x "$target")
  which roxctl && roxctl version
}

function verify_cluster_access_osd {
  hr
  echo "KUBECONFIG: [$KUBECONFIG]"
  kubectl config current-context
  #kubectl config view --raw --minify
  kubectl config view
  kubectl get no
}

function verify_cluster_access_rosa {
  rosa login --token="$OPENSHIFT_CLUSTER_MANAGER_API_TOKEN"
  echo "logging in openshift client"
  oc_login_command=$(grep "oc login https.*username.*password" "$SCRATCH/log/rosa-create-admin.log")
  eval "$oc_login_command --insecure-skip-tls-verify=true"
  oc get nodes
}

function docker_login {
  hr
  set -x
  docker login docker.io
  docker login stackrox.io
  docker login collector.stackrox.io
  set +x
}


# __MAIN__
install_gradle
install_helm
install_golang_1_17_12
install_openjdk11
install_rocksdb
test_stackrox_workflow_roxhelp
build_stackrox
setup_roxctl
docker_login
verify_cluster_access_osd
