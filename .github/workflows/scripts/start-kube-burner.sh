#!/usr/bin/bash
set -euox pipefail

export KUBE_BURNER_VERSION=1.4.3

mkdir -p ./kube-burner

curl --silent --location "https://github.com/cloud-bulldozer/kube-burner/releases/download/v${KUBE_BURNER_VERSION}/kube-burner-${KUBE_BURNER_VERSION}-$(uname -s)-$(uname -m).tar.gz" --output "./kube-burner/kube-burner-${KUBE_BURNER_VERSION}.tar.gz"

tar -zxvf "./kube-burner/kube-burner-${KUBE_BURNER_VERSION}.tar.gz" --directory ./kube-burner

kube_burner_config_file="$STACKROX_DIR"/.github/workflows/kube-burner-configs/cluster-density-kube-burner.yml 
kube_burner_gen_config_file="$STACKROX_DIR"/.github/workflows/kube-burner-configs/cluster-density-kube-burner_gen.yml 

sed "s|STACKROX_DIR|$STACKROX_DIR|" "$kube_burner_config_file" > "$kube_burner_gen_config_file" 

nohup "$STACKROX_DIR"/.github/workflows/scripts/repeate-kube-burner.sh ./kube-burner/kube-burner "$kube_burner_gen_config_file" &
