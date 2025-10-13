#!/bin/bash

central_namespace=${1:-${CENTRAL_NAMESPACE:-stackrox}}
sensor_namespace=${2:-${SENSOR_NAMESPACE:-stackrox}}

kubectl -n "${sensor_namespace}" patch svc/sensor -p '{"spec":{"ports":[{"name":"monitoring","port":9090,"protocol":"TCP","targetPort":9090}]}}'
kubectl -n "${central_namespace}" patch svc/central -p '{"spec":{"ports":[{"name":"monitoring","port":9090,"protocol":"TCP","targetPort":9090}]}}'
kubectl -n "${sensor_namespace}" patch daemonset/collector --type='json' -p='[{"op": "add", "path": "/spec/template/spec/containers/1/ports", "value":[{"containerPort":9091,"name":"cmonitor","protocol":"TCP"}]}]'
kubectl -n "${central_namespace}" patch svc/scanner-v4-indexer -p '{"spec":{"ports":[{"name":"monitoring","port":9090,"protocol":"TCP","targetPort":"monitoring"}]}}'
kubectl -n "${central_namespace}" patch svc/scanner-v4-matcher -p '{"spec":{"ports":[{"name":"monitoring","port":9090,"protocol":"TCP","targetPort":"monitoring"}]}}'

# Modify network policies to allow ingress
kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  labels:
    app.kubernetes.io/name: stackrox
  name: allow-monitoring-central
  namespace: "${central_namespace}"
spec:
  ingress:
  - ports:
    - port: 9090
      protocol: TCP
  podSelector:
    matchExpressions:
    - {key: app, operator: In, values: [central, sensor, scanner-v4-indexer, scanner-v4-matcher]}
  policyTypes:
  - Ingress
---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  labels:
    app.kubernetes.io/name: stackrox
  name: allow-compliance-monitoring
  namespace: "${sensor_namespace}"
spec:
  ingress:
  - ports:
    - port: 9091
      protocol: TCP
  podSelector:
    matchLabels:
      app: collector
  policyTypes:
  - Ingress
EOF
