#!/bin/bash

kubectl -n stackrox patch svc/sensor -p '{"spec":{"ports":[{"name":"monitoring","port":9090,"protocol":"TCP","targetPort":9090}]}}'
kubectl -n stackrox patch svc/central -p '{"spec":{"ports":[{"name":"monitoring","port":9090,"protocol":"TCP","targetPort":9090}]}}'
kubectl -n stackrox patch daemonset/collector --type='json' -p='[{"op": "add", "path": "/spec/template/spec/containers/1/ports", "value":[{"containerPort":9092,"name":"cmonitor","protocol":"TCP"}]}]'
kubectl -n stackrox patch daemonset/collector --type='json' -p='[{"op": "add", "path": "/spec/template/spec/containers/2/ports", "value":[{"containerPort":9094,"name":"cmonitor","protocol":"TCP"}]}]'

# Modify network policies to allow ingress
kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  labels:
    app.kubernetes.io/name: stackrox
  name: allow-monitoring
  namespace: stackrox
spec:
  ingress:
  - ports:
    - port: 9090
      protocol: TCP
  podSelector:
    matchExpressions:
    - {key: app, operator: In, values: [central, sensor]}
  policyTypes:
  - Ingress
---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  labels:
    app.kubernetes.io/name: stackrox
  name: allow-compliance-monitoring
  namespace: stackrox
spec:
  ingress:
  - ports:
    - port: 9092
      protocol: TCP
  podSelector:
    matchLabels:
      app: collector
  policyTypes:
  - Ingress
---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  labels:
    app.kubernetes.io/name: stackrox
  name: allow-node-inventory-monitoring
  namespace: stackrox
spec:
  ingress:
  - ports:
    - port: 9094
      protocol: TCP
  podSelector:
    matchLabels:
      app: collector
  policyTypes:
  - Ingress
EOF
