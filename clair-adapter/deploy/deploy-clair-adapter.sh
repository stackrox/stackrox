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
SCALE_DOWN_SCANNER_V4="${SCALE_DOWN_SCANNER_V4:-true}"
BUILD_IMAGE="${BUILD_IMAGE:-false}"

# Default VULN_URL: use Central's endpoint if Central is deployed, otherwise fall back to public CDN
if [[ -z "${VULN_URL:-}" ]]; then
    if kubectl get svc central -n "${NAMESPACE}" &>/dev/null; then
        VULN_URL="https://central.${NAMESPACE}.svc/api/extensions/scannerdefinitions?version=dev"
    else
        VULN_URL="https://definitions.stackrox.io/v4/vulnerability-bundles/dev/vulnerabilities.zip"
    fi
fi

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

    echo "--- Patching Scanner V4 services to route to clair-adapter ---"
    for svc in scanner-v4-indexer scanner-v4-matcher; do
        kubectl patch svc "${svc}" -n "${NAMESPACE}" --type=json \
            -p '[{"op":"replace","path":"/spec/selector","value":{"app":"clair-adapter"}}]' 2>/dev/null || true
    done
fi

# Step 3b: Generate combined TLS cert for the adapter
# The adapter needs a cert valid for both scanner-v4-indexer and scanner-v4-matcher
# DNS names so Central can connect to it as either service.
if kubectl get secret central-tls -n "${NAMESPACE}" &>/dev/null; then
    echo "--- Generating clair-adapter TLS certificate ---"
    CERT_TMPDIR="$(mktemp -d)"
    trap 'rm -rf "${CERT_TMPDIR}"' EXIT

    kubectl get secret central-tls -n "${NAMESPACE}" -o jsonpath='{.data.ca\.pem}' | base64 -d > "${CERT_TMPDIR}/ca.pem"
    kubectl get secret central-tls -n "${NAMESPACE}" -o jsonpath='{.data.ca-key\.pem}' | base64 -d > "${CERT_TMPDIR}/ca-key.pem"

    openssl genrsa -out "${CERT_TMPDIR}/key.pem" 2048 2>/dev/null

    cat > "${CERT_TMPDIR}/csr.conf" <<CSREOF
[req]
default_bits = 2048
prompt = no
distinguished_name = dn
req_extensions = v3_req

[dn]
CN = SCANNER_V4_INDEXER_SERVICE: Scanner V4 Indexer
OU = SCANNER_V4_MATCHER_SERVICE

[v3_req]
keyUsage = critical, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth, clientAuth
basicConstraints = critical, CA:FALSE
subjectAltName = @alt_names

[alt_names]
DNS.1 = scanner-v4-indexer.${NAMESPACE}
DNS.2 = scanner-v4-indexer.${NAMESPACE}.svc
DNS.3 = scanner-v4-indexer.${NAMESPACE}.svc.cluster.local
DNS.4 = scanner-v4-matcher.${NAMESPACE}
DNS.5 = scanner-v4-matcher.${NAMESPACE}.svc
DNS.6 = scanner-v4-matcher.${NAMESPACE}.svc.cluster.local
CSREOF

    openssl req -new -key "${CERT_TMPDIR}/key.pem" -out "${CERT_TMPDIR}/csr.pem" -config "${CERT_TMPDIR}/csr.conf" 2>/dev/null
    openssl x509 -req -in "${CERT_TMPDIR}/csr.pem" \
        -CA "${CERT_TMPDIR}/ca.pem" -CAkey "${CERT_TMPDIR}/ca-key.pem" -CAcreateserial \
        -out "${CERT_TMPDIR}/cert.pem" -days 365 \
        -extensions v3_req -extfile "${CERT_TMPDIR}/csr.conf" 2>/dev/null

    kubectl create secret generic clair-adapter-tls -n "${NAMESPACE}" \
        --from-file=ca.pem="${CERT_TMPDIR}/ca.pem" \
        --from-file=cert.pem="${CERT_TMPDIR}/cert.pem" \
        --from-file=key.pem="${CERT_TMPDIR}/key.pem" \
        --dry-run=client -o yaml | kubectl apply -f - 2>/dev/null
    echo "  clair-adapter-tls secret created"
else
    echo "  WARNING: central-tls secret not found, skipping TLS cert generation"
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

# Capture desired config to detect changes
CLAIR_CONFIG_DESIRED="$(cat <<CFGEOF
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
CFGEOF
)"
CLAIR_CONFIG_CURRENT="$(kubectl get configmap clair-config -n "${NAMESPACE}" -o jsonpath='{.data.config\.yaml}' 2>/dev/null)"
CLAIR_CONFIG_CHANGED=false
if [[ "${CLAIR_CONFIG_DESIRED}" != "${CLAIR_CONFIG_CURRENT}" ]]; then
    CLAIR_CONFIG_CHANGED=true
fi

kubectl apply -n "${NAMESPACE}" -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: clair-config
  labels:
    app.kubernetes.io/part-of: clair-adapter
data:
  config.yaml: |
$(echo "${CLAIR_CONFIG_DESIRED}" | sed 's/^/    /')
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
          timeoutSeconds: 5
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8089
          initialDelaySeconds: 120
          periodSeconds: 30
          timeoutSeconds: 10
          failureThreshold: 5
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

if [[ "${CLAIR_CONFIG_CHANGED}" == "true" ]]; then
    echo "  Clair config changed, restarting deployment"
    kubectl rollout restart deploy/clair -n "${NAMESPACE}" 2>/dev/null || true
fi

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
    clair_db_connstring: "host=clair-db.${NAMESPACE}.svc port=5432 user=clair dbname=clair sslmode=disable"
    grpc_listen_addr: "0.0.0.0:8443"
    http_listen_addr: "0.0.0.0:9443"
    updater_listen_addr: "0.0.0.0:9444"
    vulnerabilities_url: "${VULN_URL}"
    certs_dir: "/run/secrets/stackrox.io/certs"
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
            memory: 1Gi
          limits:
            cpu: "2"
            memory: 4Gi
        readinessProbe:
          httpGet:
            path: /healthz/ready
            port: 9443
            scheme: HTTPS
          initialDelaySeconds: 10
          periodSeconds: 10
        livenessProbe:
          httpGet:
            path: /healthz/live
            port: 9443
            scheme: HTTPS
          initialDelaySeconds: 5
          periodSeconds: 15
      volumes:
      - name: config
        configMap:
          name: clair-adapter-config
      - name: tls-volume
        secret:
          secretName: clair-adapter-tls
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

# Step 7: Restart adapter if the local image differs from what the pod is running
BUILT_IMAGE_ID="$(docker inspect --format='{{.Id}}' "${CLAIR_ADAPTER_IMAGE}" 2>/dev/null)"
RUNNING_IMAGE_ID="$(kubectl get pods -n "${NAMESPACE}" -l app=clair-adapter -o jsonpath='{.items[0].status.containerStatuses[0].imageID}' 2>/dev/null)"
if [[ -n "${BUILT_IMAGE_ID}" && "${RUNNING_IMAGE_ID}" != *"${BUILT_IMAGE_ID}"* ]]; then
    echo "--- Restarting clair-adapter (image changed) ---"
    kubectl rollout restart deploy/clair-adapter -n "${NAMESPACE}" 2>/dev/null || true
fi

# Step 8: Wait for deployments
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
