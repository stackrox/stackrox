#!/usr/bin/zsh

kubectl -n stackrox-tekton delete pr --all
kubectl -n stackrox-tekton create -f ~/workspace/src/stackrox/dev-tools/tekton/pipelinerun-local-dev.yaml
tkn -n stackrox-tekton pr logs -f $(kubectl -n stackrox-tekton get pr -o name | cut -f 2 -d /)
