#!/usr/bin/env bash
set -eoux pipefail

GO_VERSION="${GO_VERSION:-1.25.9}"
KUBECTL_VERSION="${KUBECTL_VERSION:-v1.30.0}"
KUBE_BURNER_VERSION="${KUBE_BURNER_VERSION:-1.4.3}"
STACKROX_BRANCH="${STACKROX_BRANCH:-jv-automate-perf-tests-4.11}"

git clone https://github.com/stackrox/stackrox.git
cd "${HOME}/stackrox"
git checkout "${STACKROX_BRANCH}"
cd "${HOME}"

git clone https://github.com/stackrox/workflow.git
export PATH="${PATH}:${HOME}/workflow/bin"

curl -LO "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

mkdir -p ./kube-burner
curl --silent --location "https://github.com/cloud-bulldozer/kube-burner/releases/download/v${KUBE_BURNER_VERSION}/kube-burner-${KUBE_BURNER_VERSION}-$(uname -s)-$(uname -m).tar.gz" --output "./kube-burner/kube-burner-${KUBE_BURNER_VERSION}.tar.gz"

tar -zxvf "./kube-burner/kube-burner-${KUBE_BURNER_VERSION}.tar.gz" --directory ./kube-burner

export KUBE_BURNER_PATH="$(pwd)/kube-burner/kube-burner"
echo "export KUBE_BURNER_PATH=$KUBE_BURNER_PATH" >> ~/.bashrc

sudo cp "${HOME}/oc" /usr/bin

curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
helm repo add rhacs https://mirror.openshift.com/pub/rhacs/charts
helm repo update

sudo install -o root -g root -m 0755 "${HOME}/roxctl" /usr/local/bin/roxctl

sudo apt-get install jq -y
sudo snap install yq

sudo snap install go --channel="${GO_VERSION}/stable" --classic
