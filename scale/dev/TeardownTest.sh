#!/usr/bin/env bash

~/go/src/github.com/stackrox/workflow/bin/teardown || true
kubectl delete ns stackrox1 || true
