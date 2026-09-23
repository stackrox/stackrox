#!/usr/bin/env bats

setup() {
    policy="${BATS_TEST_DIRNAME}/../../../deploy/common/ci-resource-policy.sh"
}

@test "resource policy covers init containers and sidecars and preserves storage" {
    cat > "$BATS_TEST_TMPDIR/input.yaml" <<'YAML'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: central
spec:
  template:
    spec:
      containers:
      - name: central
        image: central:test
        resources:
          requests: {cpu: 1500m, memory: 4Gi}
          limits: {cpu: '4', memory: 8Gi}
      - name: sidecar
        image: sidecar:test
        resources:
          requests: {cpu: 10m, memory: 16Mi}
          limits: {cpu: 250m, memory: 32Mi}
      initContainers:
      - name: init-db
        image: db:test
        resources:
          requests: {cpu: 500m, memory: 1Gi}
          limits: {cpu: '8', memory: 16Gi}
---
apiVersion: v1
kind: PersistentVolumeClaim
spec:
  resources:
    requests: {storage: 100Gi}
YAML
    "$policy" < "$BATS_TEST_TMPDIR/input.yaml" > "$BATS_TEST_TMPDIR/output.yaml"
    yq eval -N -j '.' "$BATS_TEST_TMPDIR/output.yaml" | jq -se '
      .[0].spec.template.spec as $spec |
      ($spec.containers[0].resources.limits.memory == "1Gi") and
      ($spec.containers[1].resources.limits.memory == "16Mi") and
      ($spec.initContainers[0].resources.limits.memory == "1Gi") and
      (all(($spec.containers[], $spec.initContainers[]);
        .resources.requests.memory == .resources.limits.memory and
        (.resources.limits | has("cpu") | not))) and
      (.[1].spec.resources.requests.storage == "100Gi")'
    "$policy" < "$BATS_TEST_TMPDIR/output.yaml" > "$BATS_TEST_TMPDIR/repeated.yaml"
    cmp "$BATS_TEST_TMPDIR/output.yaml" "$BATS_TEST_TMPDIR/repeated.yaml"
}

@test "invalid YAML fails rather than producing an empty install" {
    run bash -c 'printf "spec: [invalid" | "$1"' _ "$policy"
    [ "$status" -ne 0 ]
}

@test "operator policy preserves secrets and existing overlays and is idempotent" {
    cat > "$BATS_TEST_TMPDIR/input.yaml" <<'YAML'
kind: Central
spec:
  overlays:
  - kind: Deployment
    name: central
    patches:
    - path: metadata.labels.test
      value: retained
---
kind: Secret
data:
  password: unchanged
YAML
    "$policy" --operator < "$BATS_TEST_TMPDIR/input.yaml" > "$BATS_TEST_TMPDIR/output.yaml"
    yq eval -N -j '.' "$BATS_TEST_TMPDIR/output.yaml" | jq -se '
      (.[0].spec.overlays | length > 1) and
      (.[0].spec.overlays[0].patches[0].value == "retained") and
      (.[1].data.password == "unchanged")'
    "$policy" --operator < "$BATS_TEST_TMPDIR/output.yaml" > "$BATS_TEST_TMPDIR/repeated.yaml"
    cmp "$BATS_TEST_TMPDIR/output.yaml" "$BATS_TEST_TMPDIR/repeated.yaml"
}

@test "roxie policy covers both CRs" {
    printf '%s\n' 'central: {spec: {central: {resources: {requests: {memory: 1Gi}}}}}' 'securedCluster: {}' |
      "$policy" --operator > "$BATS_TEST_TMPDIR/output.yaml"
    yq eval -N -j '.' "$BATS_TEST_TMPDIR/output.yaml" | jq -e '
      (.central.spec.overlays | length > 0) and
      (.securedCluster.spec.overlays | length > 0) and
      (.central.spec.central.resources.requests.memory == "1Gi")'
}

@test "directory mode ignores non-manifest files" {
    mkdir "$BATS_TEST_TMPDIR/manifests"
    printf '%s\n' '{"kind":"Pod","spec":{"containers":[{"name":"central","image":"test","resources":{"requests":{"memory":"4Gi","cpu":"1"},"limits":{"memory":"8Gi","cpu":"4"}}}]}}' > "$BATS_TEST_TMPDIR/manifests/pod.yaml"
    printf '%s\n' 'unchanged' > "$BATS_TEST_TMPDIR/manifests/cert.pem"
    "$policy" --directory "$BATS_TEST_TMPDIR/manifests"
    yq eval -N -j '.' "$BATS_TEST_TMPDIR/manifests/pod.yaml" | jq -e '.spec.containers[0].resources.limits == {"memory":"1Gi"}'
    [ "$(cat "$BATS_TEST_TMPDIR/manifests/cert.pem")" = unchanged ]
}
