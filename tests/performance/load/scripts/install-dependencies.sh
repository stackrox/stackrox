#!/usr/bin/env bash
set -eoux pipefail

if [ ! -d "${HOME}/stackrox" ]; then
  git clone https://github.com/stackrox/stackrox.git
fi
cd "${HOME}/stackrox"
git checkout jv-ROX-scripts-for-k6-load-testing
cd "${HOME}"

curl -LO https://dl.k8s.io/release/v1.30.0/bin/linux/amd64/kubectl
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

curl -O https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-435.0.1-linux-x86_64.tar.gz
tar -xf google-cloud-cli-435.0.1-linux-x86_64.tar.gz
./google-cloud-sdk/install.sh --quiet
#source .bashrc
#./google-cloud-sdk/bin/gcloud init
mkdir .config || true
rm -rf ~/.config/gcloud || true
cp -r ~/gcloud .config
export PATH=/home/jvirtane/google-cloud-sdk/bin:${PATH}
gcloud components install gke-gcloud-auth-plugin --quiet

curl -s https://dl.k6.io/key.gpg | sudo gpg --dearmor -o /usr/share/keyrings/k6-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list

sudo apt update
sudo apt install npm -y
sudo apt install k6
