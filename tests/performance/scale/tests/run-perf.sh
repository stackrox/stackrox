#!/usr/bin/env bash
set -eoux pipefail

ELASTIC_USERNAME=$1
ELASTIC_PASSWORD=$2

export ELASTICSEARCH_URL=https://${ELASTIC_USERNAME}:${ELASTIC_PASSWORD}@search-acs-perfscale-koafbspz7ynsknj7r6cxxlmqh4.us-east-1.es.amazonaws.com

export ARTIFACTS_DIR="${HOME}/artifacts"

git clone https://github.com/stackrox/stackrox.git
cd stackrox
git checkout jv-automate-perf-tests
cd ..

export KUBE_BURNER_VERSION=1.4.3

mkdir -p ./kube-burner
curl --silent --location "https://github.com/cloud-bulldozer/kube-burner/releases/download/v${KUBE_BURNER_VERSION}/kube-burner-${KUBE_BURNER_VERSION}-$(uname -s)-$(uname -m).tar.gz" --output "./kube-burner/kube-burner-${KUBE_BURNER_VERSION}.tar.gz"

tar -zxvf "./kube-burner/kube-burner-${KUBE_BURNER_VERSION}.tar.gz" --directory ./kube-burner

export KUBE_BURNER_PATH="$(pwd)/kube-burner/kube-burner"

export KUBECONFIG="${ARTIFACTS_DIR}/kubeconfig"

#wget https://github.com/openshift/origin/releases/download/v3.10.0/openshift-origin-client-tools-v3.10.0-dd10d17-linux-64bit.tar.gz
#
#tar -xzf openshift-origin-client-tools-v3.10.0-dd10d17-linux-64bit.tar.gz

#export PATH="$HOME/openshift-origin-client-tools-v3.10.0-dd10d17-linux-64bit":$PATH
sudo cp "${HOME}/oc" /usr/bin

export KUBECONFIG=$HOME/artifacts/kubeconfig

export PROMETHEUS_URL="https://$(oc get route --namespace openshift-monitoring prometheus-k8s --output jsonpath='{.spec.host}' | xargs)"

export PROMETHEUS_TOKEN="$(oc serviceaccounts new-token --namespace openshift-monitoring prometheus-k8s)"

curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
helm repo add rhacs https://mirror.openshift.com/pub/rhacs/charts
helm repo update

arch="$(uname -m | sed "s/x86_64//")"; arch="${arch:+-$arch}"
curl -f -o roxctl "https://mirror.openshift.com/pub/rhacs/assets/4.4.2/bin/Linux/roxctl${arch}"
chmod +x roxctl
sudo cp roxctl /usr/local/bin

sudo apt-get install jq -y

# Set number of pods per node
oc create --filename=$HOME/stackrox/tests/performance/scale/utilities/examples/set-max-pods.yml

cd ${HOME}/stackrox/tests/performance/scale/utilities
./start-central-and-scanner.sh "${ARTIFACTS_DIR}"
./wait-for-pods.sh "${ARTIFACTS_DIR}"
./get-bundle.sh "${ARTIFACTS_DIR}"
./start-secured-cluster.sh $ARTIFACTS_DIR

sudo snap install go --channel=1.21/stable --classic

cd ${HOME}/stackrox/tests/performance/scale/tests/kube-burner/cluster-density

#./run-workload.sh --kube-burner-path "${KUBE_BURNER_PATH}" --num-namespaces 1250 --num-deployments 20 --num-pods 1
#./run-workload.sh --kube-burner-path "${KUBE_BURNER_PATH}" --num-namespaces 1000 --num-deployments 6 --num-pods 4
#./run-workload.sh --kube-burner-path "${KUBE_BURNER_PATH}" --num-namespaces 800 --num-deployments 10 --num-pods 3
#./run-workload.sh --kube-burner-path "${KUBE_BURNER_PATH}" --num-namespaces 950 --num-deployments 9 --num-pods 3


./run-workload.sh --kube-burner-path "${KUBE_BURNER_PATH}" --num-namespaces 10 --num-deployments 5 --num-pods 1
