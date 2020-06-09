#!/bin/bash

kubectl -n stackrox patch svc/sensor -p '{"spec":{"ports":[{"name":"monitoring","port":9090,"protocol":"TCP","targetPort":9090}]}}'
kubectl -n stackrox patch svc/central -p '{"spec":{"ports":[{"name":"monitoring","port":9090,"protocol":"TCP","targetPort":9090}]}}'

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
EOF
