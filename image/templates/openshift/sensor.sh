#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

OC_PROJECT={{.Namespace}}
OC_NAMESPACE={{.Namespace}}
OC_SA="${OC_SA:-sensor}"
OC_BENCHMARK_SA="${OC_BENCHMARK_SA:-benchmark}"

oc create -f "$DIR/sensor-rbac.yaml"

oc adm policy add-scc-to-user sensor "system:serviceaccount:$OC_PROJECT:$OC_SA"
oc adm policy add-scc-to-user benchmark "system:serviceaccount:$OC_PROJECT:$OC_BENCHMARK_SA"

oc create secret -n "{{.Namespace}}" generic sensor-tls --from-file="$DIR/sensor-cert.pem" --from-file="$DIR/sensor-key.pem" --from-file="$DIR/ca.pem"
oc create -f "$DIR/sensor.yaml"
