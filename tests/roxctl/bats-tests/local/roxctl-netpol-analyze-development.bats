#!/usr/bin/env bats

load "../helpers.bash"
out_dir=""
templated_fragment='"{{ printf "%s" ._thing.image }}"'

setup_file() {
    [[ -n "$NO_BATS_ROXCTL_REBUILD" ]] || rm -f "${tmp_roxctl}"/roxctl*
    echo "Testing roxctl version: '$(roxctl-development version)'" >&3
}

setup() {
  out_dir="$(mktemp -d -u)"
  ofile="$(mktemp)"
}

teardown() {
  rm -rf "$out_dir"
  rm -f "$ofile"
}


@test "roxctl-development analyze netpol should return error on empty or non-existing directory" {
  run roxctl-development analyze netpol "$out_dir" 
  assert_failure
  assert_line --partial "error in connectivity analysis"
  assert_line --partial "no such file or directory"

  run roxctl-development analyze netpol
  assert_failure
  assert_line --partial "accepts 1 arg(s), received 0"
}

@test "roxctl-development analyze netpol generates connlist output" {
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/backend.yaml"
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/frontend.yaml"
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/netpols.yaml"
  echo "Writing connlist to ${ofile}" >&3
  run roxctl-development analyze netpol "${test_data}/np-guard/netpols-analysis-example-minimal"
  assert_success

  
  echo "$output" > "$ofile"
  assert_file_exist "$ofile"
  assert_output --partial 'default/frontend[Deployment] => default/backend[Deployment] : TCP 9090'

}


@test "roxctl-development analyze netpol stops on first error when run with --fail" {
  mkdir -p "$out_dir"
  write_yaml_to_file "$templated_fragment" "$(mktemp "$out_dir/templated-01-XXXXXX-file1.yaml")"
  write_yaml_to_file "$templated_fragment" "$(mktemp "$out_dir/templated-02-XXXXXX-file2.yaml")"

  run roxctl-development analyze netpol "$out_dir/" --remove --output-file=/dev/null --fail
  assert_failure
  assert_output --partial 'YAML document is malformed'
  assert_output --partial 'file1.yaml'
  refute_output --partial 'file2.yaml'
}


@test "roxctl-development analyze netpol produces no output when all yamls are templated" {
  mkdir -p "$out_dir"
  write_yaml_to_file "$templated_fragment" "$(mktemp "$out_dir/templated-XXXXXX.yaml")"

  echo "Analyzing a corrupted yaml file '$templatedYaml'" >&3
  run roxctl-development analyze netpol "$out_dir/"
  assert_failure
  assert_output --partial 'YAML document is malformed'
  assert_output --partial 'no relevant Kubernetes resources found'
}

@test "roxctl-development analyze netpol produces errors when some yamls are templated" {
  mkdir -p "$out_dir"
  write_yaml_to_file "$templated_fragment" "$(mktemp "$out_dir/templated-XXXXXX.yaml")"

  assert_file_exist "${test_data}/np-guard/scenario-minimal-service/frontend.yaml"
  assert_file_exist "${test_data}/np-guard/scenario-minimal-service/backend.yaml"
  cp "${test_data}/np-guard/scenario-minimal-service/frontend.yaml" "$out_dir/frontend.yaml"
  cp "${test_data}/np-guard/scenario-minimal-service/backend.yaml" "$out_dir/backend.yaml"

  echo "Analyzing a directory where 1/3 of yaml files are templated '$out_dir/'" >&3
  run roxctl-development analyze netpol "$out_dir/" --remove --output-file=/dev/null
  assert_failure
  assert_output --partial 'YAML document is malformed'
  refute_output --partial 'no relevant Kubernetes resources found'
}


@test "roxctl-development analyze netpol produces errors when yamls are not K8s resources" {
  mkdir -p "$out_dir"
  assert_file_exist "${test_data}/np-guard/empty-yamls/empty.yaml"
  assert_file_exist "${test_data}/np-guard/empty-yamls/empty2.yaml"
  cp "${test_data}/np-guard/empty-yamls/empty.yaml" "$out_dir/empty.yaml"
  cp "${test_data}/np-guard/empty-yamls/empty2.yaml" "$out_dir/empty2.yaml"

  run roxctl-development analyze netpol "$out_dir/" --remove --output-file=/dev/null
  assert_failure
  assert_output --partial 'Yaml document is not a K8s resource'
  assert_output --partial 'no relevant Kubernetes resources found'
  assert_output --partial 'ERROR:'
  assert_output --partial 'there were errors during execution'
}

@test "roxctl-development analyze netpol should return error on invalid networkpolicy resource" {
  assert_file_exist "${test_data}/np-guard/bad-netpol-example/resources.yaml"  
  run roxctl-development analyze netpol "${test_data}/np-guard/bad-netpol-example"    
  assert_failure
  assert_line --partial "error in connectivity analysis"
  assert_line --partial "selector error"
}


write_yaml_to_file() {
  image="${1}"
  templatedYaml="${2:-/dev/null}"
  cat >"$templatedYaml" <<-EOF
  cat $templatedYaml >&3 
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
        image: $image
        ports:
        - containerPort: 8080
        env:
        - name: PORT
          value: "8080"
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
