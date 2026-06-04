#!/usr/bin/env bash
#
# Deploy Clair + Clair Adapter to a local Kubernetes cluster.
#
# Compatible with the StackRox deploy-local.sh workflow:
#   1. Run ./deploy/deploy-local.sh first (with or without ROX_SCANNER_V4=false)
#   2. Run this script to deploy Clair + Adapter alongside StackRox
#
# Environment variables:
#   NAMESPACE              - Kubernetes namespace (default: stackrox)
#   CLAIR_ADAPTER_IMAGE    - Adapter container image (default: clair-adapter:dev)
#   CLAIR_IMAGE            - Upstream Clair image (default: quay.io/projectquay/clair:4.7.4)
#   CLAIR_DB_IMAGE         - Clair PostgreSQL image (default: docker.io/library/postgres:15)
#   VULN_URL               - Vulnerability definitions URL (default: definitions.stackrox.io)
#   SCALE_DOWN_SCANNER_V4  - Scale down Scanner V4 if running (default: true)
#   BUILD_IMAGE            - Build adapter image before deploying (default: false)
#   CLUSTER_TYPE           - kind, minikube, or docker-desktop (auto-detected)

set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${DIR}/../.." && pwd)"

# Configuration with defaults
NAMESPACE="${NAMESPACE:-stackrox}"
CLAIR_ADAPTER_IMAGE="${CLAIR_ADAPTER_IMAGE:-clair-adapter:dev}"
CLAIR_IMAGE="${CLAIR_IMAGE:-quay.io/projectquay/clair:4.7.4}"
CLAIR_DB_IMAGE="${CLAIR_DB_IMAGE:-docker.io/library/postgres:15}"
VULN_URL="${VULN_URL:-https://definitions.stackrox.io/v4/vulnerability-bundles/dev/vulnerabilities.zip}"
SCALE_DOWN_SCANNER_V4="${SCALE_DOWN_SCANNER_V4:-true}"
BUILD_IMAGE="${BUILD_IMAGE:-false}"

# Auto-detect cluster type
detect_cluster_type() {
    if kubectl config current-context 2>/dev/null | grep -q "kind-"; then
        echo "kind"
    elif kubectl config current-context 2>/dev/null | grep -q "minikube"; then
        echo "minikube"
    else
        echo "docker-desktop"
    fi
}
CLUSTER_TYPE="${CLUSTER_TYPE:-$(detect_cluster_type)}"

echo "=== Clair Adapter Deployment ==="
echo "  Namespace:       ${NAMESPACE}"
echo "  Adapter image:   ${CLAIR_ADAPTER_IMAGE}"
echo "  Clair image:     ${CLAIR_IMAGE}"
echo "  Cluster type:    ${CLUSTER_TYPE}"
echo "  Vuln URL:        ${VULN_URL}"
echo ""

# Step 0: Optionally build the adapter image
if [[ "${BUILD_IMAGE}" == "true" ]]; then
    echo "--- Building clair-adapter image ---"
    docker build -t "${CLAIR_ADAPTER_IMAGE}" -f "${DIR}/../Dockerfile" "${REPO_ROOT}"
fi

# Step 1: Load image into cluster if needed
echo "--- Loading adapter image into cluster ---"
case "${CLUSTER_TYPE}" in
    kind)
        kind load docker-image "${CLAIR_ADAPTER_IMAGE}" 2>/dev/null || echo "  (image may already be loaded or kind not in use)"
        ;;
    minikube)
        minikube image load "${CLAIR_ADAPTER_IMAGE}" 2>/dev/null || echo "  (image may already be loaded)"
        ;;
    docker-desktop)
        echo "  Docker Desktop: image is available locally"
        ;;
esac

# Step 2: Create namespace if needed
kubectl create namespace "${NAMESPACE}" 2>/dev/null || true

# Step 3: Scale down Scanner V4 if running
if [[ "${SCALE_DOWN_SCANNER_V4}" == "true" ]]; then
    echo "--- Scaling down Scanner V4 (if present) ---"
    kubectl scale deploy scanner-v4-indexer scanner-v4-matcher -n "${NAMESPACE}" --replicas=0 2>/dev/null || true
fi

# Step 4: Deploy Clair PostgreSQL
echo "--- Deploying Clair PostgreSQL ---"
kubectl apply -n "${NAMESPACE}" -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: clair-db
  labels:
    app: clair-db
    app.kubernetes.io/component: clair-db
    app.kubernetes.io/part-of: clair-adapter
spec:
  replicas: 1
  strategy:
    type: Recreate
  selector:
    matchLabels:
      app: clair-db
  template:
    metadata:
      labels:
        app: clair-db
    spec:
      containers:
      - name: postgres
        image: ${CLAIR_DB_IMAGE}
        env:
        - name: POSTGRES_USER
          value: clair
        - name: POSTGRES_DB
          value: clair
        - name: POSTGRES_HOST_AUTH_METHOD
          value: trust
        - name: PGDATA
          value: /var/lib/postgresql/data/pgdata
        ports:
        - containerPort: 5432
          protocol: TCP
        volumeMounts:
        - name: data
          mountPath: /var/lib/postgresql/data
        resources:
          requests:
            cpu: 200m
            memory: 512Mi
          limits:
            cpu: "1"
            memory: 1Gi
        readinessProbe:
          exec:
            command: ["pg_isready", "-U", "clair"]
          initialDelaySeconds: 5
          periodSeconds: 10
      volumes:
      - name: data
        emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: clair-db
  labels:
    app: clair-db
    app.kubernetes.io/part-of: clair-adapter
spec:
  selector:
    app: clair-db
  ports:
  - port: 5432
    targetPort: 5432
    protocol: TCP
EOF

# Step 5: Deploy Clair config and service
echo "--- Deploying upstream Clair ---"
kubectl apply -n "${NAMESPACE}" -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: clair-config
  labels:
    app.kubernetes.io/part-of: clair-adapter
data:
  config.yaml: |
    http_listen_addr: 0.0.0.0:8080
    introspection_addr: 0.0.0.0:8089
    log_level: info
    indexer:
      connstring: host=clair-db.${NAMESPACE}.svc port=5432 user=clair dbname=clair sslmode=disable
      scanlock_retry: 10
      layer_scan_concurrency: 5
      migrations: true
    matcher:
      connstring: host=clair-db.${NAMESPACE}.svc port=5432 user=clair dbname=clair sslmode=disable
      migrations: true
    notifier:
      connstring: host=clair-db.${NAMESPACE}.svc port=5432 user=clair dbname=clair sslmode=disable
      migrations: true
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: clair
  labels:
    app: clair
    app.kubernetes.io/component: clair
    app.kubernetes.io/part-of: clair-adapter
spec:
  replicas: 1
  selector:
    matchLabels:
      app: clair
  template:
    metadata:
      labels:
        app: clair
    spec:
      containers:
      - name: clair
        image: ${CLAIR_IMAGE}
        args:
        - "-conf"
        - "/etc/clair/config.yaml"
        - "-mode"
        - "combo"
        ports:
        - containerPort: 8080
          name: http
          protocol: TCP
        - containerPort: 8089
          name: introspection
          protocol: TCP
        volumeMounts:
        - name: config
          mountPath: /etc/clair
        resources:
          requests:
            cpu: 200m
            memory: 1Gi
          limits:
            cpu: "2"
            memory: 4Gi
        readinessProbe:
          httpGet:
            path: /healthz
            port: 8089
          initialDelaySeconds: 30
          periodSeconds: 10
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8089
          initialDelaySeconds: 60
          periodSeconds: 30
      volumes:
      - name: config
        configMap:
          name: clair-config
---
apiVersion: v1
kind: Service
metadata:
  name: clair
  labels:
    app: clair
    app.kubernetes.io/part-of: clair-adapter
spec:
  selector:
    app: clair
  ports:
  - port: 8080
    targetPort: 8080
    name: http
    protocol: TCP
  - port: 8089
    targetPort: 8089
    name: introspection
    protocol: TCP
EOF

# Step 6: Deploy Clair Adapter
echo "--- Deploying Clair Adapter ---"
kubectl apply -n "${NAMESPACE}" -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: clair-adapter-config
  labels:
    app.kubernetes.io/part-of: clair-adapter
data:
  config.yaml: |
    clair_url: "http://clair.${NAMESPACE}.svc:8080"
    grpc_listen_addr: "0.0.0.0:8443"
    http_listen_addr: "0.0.0.0:9443"
    updater_listen_addr: "0.0.0.0:9444"
    vulnerabilities_url: "${VULN_URL}"
    indexer:
      enable: true
    matcher:
      enable: true
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: clair-adapter
  labels:
    app: clair-adapter
    app.kubernetes.io/component: clair-adapter
    app.kubernetes.io/part-of: clair-adapter
spec:
  replicas: 1
  selector:
    matchLabels:
      app: clair-adapter
  template:
    metadata:
      labels:
        app: clair-adapter
    spec:
      containers:
      - name: clair-adapter
        image: ${CLAIR_ADAPTER_IMAGE}
        imagePullPolicy: IfNotPresent
        args:
        - "-config"
        - "/etc/clair-adapter/config.yaml"
        ports:
        - containerPort: 8443
          name: grpc
          protocol: TCP
        - containerPort: 9443
          name: health
          protocol: TCP
        - containerPort: 9444
          name: updater
          protocol: TCP
        volumeMounts:
        - name: config
          mountPath: /etc/clair-adapter
        - name: tls-volume
          mountPath: /run/secrets/stackrox.io/certs/
          readOnly: true
        resources:
          requests:
            cpu: 200m
            memory: 512Mi
          limits:
            cpu: "1"
            memory: 2Gi
        readinessProbe:
          httpGet:
            path: /healthz/ready
            port: 9443
          initialDelaySeconds: 10
          periodSeconds: 10
        livenessProbe:
          httpGet:
            path: /healthz/live
            port: 9443
          initialDelaySeconds: 5
          periodSeconds: 15
      volumes:
      - name: config
        configMap:
          name: clair-adapter-config
      - name: tls-volume
        secret:
          secretName: scanner-v4-matcher-tls
          optional: true
---
apiVersion: v1
kind: Service
metadata:
  name: clair-adapter
  labels:
    app: clair-adapter
    app.kubernetes.io/part-of: clair-adapter
spec:
  selector:
    app: clair-adapter
  ports:
  - port: 8443
    targetPort: 8443
    name: grpc
    protocol: TCP
  - port: 9443
    targetPort: 9443
    name: health
    protocol: TCP
  - port: 9444
    targetPort: 9444
    name: updater
    protocol: TCP
EOF

# Step 7: Wait for deployments
echo ""
echo "--- Waiting for pods to be ready ---"
kubectl rollout status deploy/clair-db -n "${NAMESPACE}" --timeout=120s
echo "  clair-db: ready"
kubectl rollout status deploy/clair -n "${NAMESPACE}" --timeout=180s
echo "  clair: ready"
kubectl rollout status deploy/clair-adapter -n "${NAMESPACE}" --timeout=120s
echo "  clair-adapter: ready"

# Step 8: Print status and next steps
echo ""
echo "=== Deployment Complete ==="
echo ""
kubectl get pods -n "${NAMESPACE}" -l app.kubernetes.io/part-of=clair-adapter -o wide
echo ""
echo "--- Quick Test ---"
echo ""
echo "  # Port-forward gRPC (for grpcurl):"
echo "  kubectl port-forward -n ${NAMESPACE} svc/clair-adapter 8443:8443"
echo ""
echo "  # List services:"
echo "  grpcurl -plaintext localhost:8443 list"
echo ""
echo "  # Index an image:"
echo "  grpcurl -plaintext localhost:8443 scanner.v4.Indexer/CreateIndexReport \\"
echo "    -d '{\"hash_id\": \"/v4/containerimage/sha256:test1\", \"container_image\": {\"url\": \"https://docker.io/library/alpine:3.18\"}}'"
echo ""
echo "  # Get vulnerabilities:"
echo "  grpcurl -plaintext localhost:8443 scanner.v4.Matcher/GetVulnerabilities \\"
echo "    -d '{\"hash_id\": \"/v4/containerimage/sha256:test1\"}'"
echo ""
echo "  # Check adapter health:"
echo "  kubectl port-forward -n ${NAMESPACE} svc/clair-adapter 9443:9443"
echo "  curl http://localhost:9443/healthz/ready"
echo ""
echo "  # View logs:"
echo "  kubectl logs -n ${NAMESPACE} deploy/clair-adapter -f"
echo ""

# Step 9: If StackRox Central is running, show integration instructions
if kubectl get deploy/central -n "${NAMESPACE}" &>/dev/null; then
    echo "--- StackRox Central Detected ---"
    echo ""
    echo "  Scanner V4 has been scaled down. To test with Central:"
    echo ""
    echo "  Note: Full Central integration requires mTLS (not yet implemented)."
    echo "  For now, test via grpcurl directly against the adapter."
    echo ""
fi
