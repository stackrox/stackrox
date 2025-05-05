#!/usr/bin/env bash
set -eoux pipefail

git clone https://github.com/stackrox/stackrox.git
cd "${HOME}/stackrox"
git checkout jv-automate-perf-tests-stable-use-correct-helm
cd "${HOME}"

git clone https://github.com/stackrox/workflow.git
export PATH=${PATH}:/home/jvirtane/workflow/bin

curl -LO https://dl.k8s.io/release/v1.30.0/bin/linux/amd64/kubectl
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

export KUBE_BURNER_VERSION=1.4.3

mkdir -p ./kube-burner
curl --silent --location "https://github.com/cloud-bulldozer/kube-burner/releases/download/v${KUBE_BURNER_VERSION}/kube-burner-${KUBE_BURNER_VERSION}-$(uname -s)-$(uname -m).tar.gz" --output "./kube-burner/kube-burner-${KUBE_BURNER_VERSION}.tar.gz"

tar -zxvf "./kube-burner/kube-burner-${KUBE_BURNER_VERSION}.tar.gz" --directory ./kube-burner

export KUBE_BURNER_PATH="$(pwd)/kube-burner/kube-burner"
echo "export KUBE_BURNER_PATH=$KUBE_BURNER_PATH" >> ~/.bashrc

sudo cp "${HOME}/oc" /usr/bin

curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
helm repo add rhacs https://mirror.openshift.com/pub/rhacs/charts
helm repo update

arch="$(uname -m | sed "s/x86_64//")"; arch="${arch:+-$arch}"
roxctl_version=4.6.5
curl -f -o roxctl "https://mirror.openshift.com/pub/rhacs/assets/${roxctl_version}/bin/Linux/roxctl${arch}"
chmod +x roxctl
sudo cp roxctl /usr/local/bin

sudo apt-get install jq -y
sudo snap install yq

sudo snap install go --channel=1.21/stable --classic
