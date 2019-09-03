#!/usr/bin/env bash

set -euo pipefail
set -x

cluster_name="${1:-remote}"

ROX_ADMIN_PASSWORD="${ROX_ADMIN_PASSWORD:-$(< deploy/k8s/central-deploy/password)}"
MAIN_IMAGE_TAG="${MAIN_IMAGE_TAG:-$(make tag)}"

ROX_CLUSTER_ID="$(curl -sk -u admin:"${ROX_ADMIN_PASSWORD}" "https://localhost:8000/v1/clusters?query=cluster:${cluster_name}" | jq -r '.clusters[0].id')"
if [[ -z "$ROX_CLUSTER_ID" ]]; then
	echo >&2 "No such cluster: ${cluster_name}"
	exit 1
fi

ROX_UPGRADE_PROCESS_ID="$(uuidgen)"

kubectl -n stackrox create -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sensor-upgrader
  namespace: stackrox
  labels:
    app: sensor-upgrader
spec:
  replicas: 1
  selector:
    matchLabels:
      app: sensor-upgrader
  template:
    metadata:
      labels:
        app: sensor-upgrader
      namespace: stackrox
    spec:
      containers:
      - name: upgrader
        image: stackrox/main:${MAIN_IMAGE_TAG}
        imagePullPolicy: IfNotPresent
        command: ["sh", "-c", "sensor-upgrader -workflow roll-forward ; sleep 5 ; sensor-upgrader -workflow cleanup ;"]
        env:
        - name: ROX_CLUSTER_ID
          value: "${ROX_CLUSTER_ID}"
        - name: ROX_UPGRADE_PROCESS_ID
          value: "${ROX_UPGRADE_PROCESS_ID}"
        - name: ROX_CENTRAL_ENDPOINT
          value: "central.stackrox:443"
        - name: ROX_UPGRADER_OWNER
          value: "Deployment:apps/v1:stackrox/sensor-upgrader"
        volumeMounts:
        - mountPath: /run/secrets/stackrox.io/certs/
          name: certs
          readOnly: true
      serviceAccountName: sensor
      imagePullSecrets:
      - name: stackrox
      volumes:
      - name: certs
        secret:
          defaultMode: 420
          items:
          - key: sensor-cert.pem
            path: cert.pem
          - key: sensor-key.pem
            path: key.pem
          - key: ca.pem
            path: ca.pem
          secretName: sensor-tls
EOF
