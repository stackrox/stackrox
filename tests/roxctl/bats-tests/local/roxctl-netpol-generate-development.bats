#!/usr/bin/env bats

load "../helpers.bash"

out_dir=""

setup_file() {
    command -v yq >/dev/null || skip "Tests in this file require yq"
    echo "Using yq version: '$(yq4.16 --version)'" >&3
    # as of Aug 2022, we run yq version 4.16.2
    # remove binaries from the previous runs
    rm -f "$(roxctl-development-cmd)" "$(roxctl-release-cmd)"
    echo "Testing roxctl version: '$(roxctl-development version)'" >&3
}

setup() {
  out_dir="$(mktemp -d -u)"
  ofile="$(mktemp)"
  export ROX_ROXCTL_NETPOL_GENERATE='true'
}

teardown() {
  rm -rf "$out_dir"
  rm -f "$ofile"
}

@test "roxctl-development generate netpol should respect ROX_ROXCTL_NETPOL_GENERATE feature-flag at runtime" {
  export ROX_ROXCTL_NETPOL_GENERATE=false
  run roxctl-development generate netpol "$out_dir"
  assert_failure
  assert_line --partial 'unknown command "generate"'
}


@test "roxctl-development generate netpol should return error on empty or non-existing directory" {
  run roxctl-development generate netpol "$out_dir"
  assert_failure
  assert_line --regexp "[eE]rror synthesizing policies from folder: no deployment objects discovered in the repository"

  run roxctl-development generate netpol
  assert_failure
  assert_line --partial "accepts 1 arg(s), received 0"
}

@test "roxctl-development generate netpol generates network policies" {
  assert_file_exist "${test_data}/np-guard/scenario-minimal-service/frontend.yaml"
  assert_file_exist "${test_data}/np-guard/scenario-minimal-service/backend.yaml"
  echo "Writing network policies to ${ofile}" >&3
  run roxctl-development generate netpol "${test_data}/np-guard/scenario-minimal-service"
  assert_success

  echo "$output" > "$ofile"
  assert_file_exist "$ofile"
  yaml_valid "$ofile"

  # There must be at least 2 yaml documents in the output
  # yq version 4.16.2 has problems with handling 'document_index', thus we use 'di'
  run yq e 'di' "${ofile}"
  assert_line '0'
  assert_line '1'

  # Ensure that both yaml docs are of kind 'NetworkPolicy'
  run yq e '.kind | ({"match": ., "doc": di})' "${ofile}"
  assert_line --index 0 'match: NetworkPolicy'
  assert_line --index 1 'doc: 0'
  assert_line --index 2 'match: NetworkPolicy'
  assert_line --index 3 'doc: 1'
}

@test "roxctl-release generate netpol produces no output when all yamls are templated" {
  mkdir -p "$out_dir"
  write_templated_yaml_to_file "$(mktemp "$out_dir/templated-XXXXXX.yaml")"

  echo "Analyzing a corrupted yaml file '$templatedYaml'" >&3
  run roxctl-release generate netpol "$out_dir/"
  # We may actually want to throw an error if all yamls are corrupted
  assert_success
  assert_output ''
}

@test "roxctl-release generate netpol produces <warning/error>? when some yamls are templated" {
  mkdir -p "$out_dir"
  write_templated_yaml_to_file "$(mktemp "$out_dir/templated-XXXXXX.yaml")"

  assert_file_exist "${test_data}/np-guard/scenario-minimal-service/frontend.yaml"
  assert_file_exist "${test_data}/np-guard/scenario-minimal-service/backend.yaml"
  cp "${test_data}/np-guard/scenario-minimal-service/frontend.yaml" "$out_dir/frontend.yaml"
  cp "${test_data}/np-guard/scenario-minimal-service/backend.yaml" "$out_dir/backend.yaml"

  echo "Analyzing a directory where 1/3 of yaml files are templated '$out_dir/'" >&3
  run roxctl-release generate netpol "$out_dir/"
  # We may actually want to show a warning if some yamls are corrupted
  assert_success
  assert_output ''
}

write_templated_yaml_to_file() {
  templatedYaml="${1:-/dev/null}"
  cat >"$templatedYaml" <<-EOF
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: frontend
spec:
  selector:
    matchLabels:
      app: frontend
  template:
    metadata:
      labels:
        app: frontend
    spec:
      containers:
      - name: server
        image: "{{ printf "%s:%s" ._zzz.image.main.repository ._zzz.image.main.tag }}"
        ports:
        - containerPort: 8080
        env:
        - name: PORT
          value: 8080
        resources:
          requests:
            cpu: 100m
            memory: 64Mi
          limits:
            cpu: 200m
            memory: 128Mi
---
apiVersion: v1
kind: Service
metadata:
  name: frontend
spec:
  type: ClusterIP
  selector:
    app: frontend
  ports:
  - name: http
    port: 80
    targetPort: 8080
EOF
}
