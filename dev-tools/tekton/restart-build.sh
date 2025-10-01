#!/usr/bin/zsh

kubectl delete pr --all
kubectl create -f ~/workspace/src/stackrox/dev-tools/tekton/pipelinerun-local-dev.yaml
tkn pr logs -f $(kubectl -n stackrox-tekton get pr -o name | cut -f 2 -d /)
