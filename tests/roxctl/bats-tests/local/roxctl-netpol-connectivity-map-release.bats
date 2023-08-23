#!/usr/bin/env bats

load "../helpers.bash"
out_dir=""
templated_fragment='"{{ printf "%s" ._thing.image }}"'

setup_file() {
    [[ -n "$NO_BATS_ROXCTL_REBUILD" ]] || rm -f "${tmp_roxctl}"/roxctl*
    echo "Testing roxctl version: '$(roxctl-release version)'" >&3
}

setup() {
  out_dir="$(mktemp -d -u)"
  ofile="$(mktemp)"
}

teardown() {
  rm -rf "$out_dir"
  rm -f "$ofile"
}


@test "roxctl-release netpol connectivity map should return error on empty or non-existing directory" {
  run roxctl-release netpol connectivity map "$out_dir"
  assert_failure
  assert_line --partial "error in connectivity analysis"
  assert_line --partial "no such file or directory"

  run roxctl-release netpol connectivity map
  assert_failure
  assert_line --partial "accepts 1 arg(s), received 0"
}

@test "roxctl-release netpol connectivity map generates connlist output" {
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/backend.yaml"
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/frontend.yaml"
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/netpols.yaml"
  echo "Writing connlist to ${ofile}" >&3
  run roxctl-release netpol connectivity map "${test_data}/np-guard/netpols-analysis-example-minimal"
  assert_success

  echo "$output" > "$ofile"
  assert_file_exist "$ofile"
  assert_output --partial 'default/frontend[Deployment] => default/backend[Deployment] : TCP 9090'
}

@test "roxctl-release netpol connectivity map stops on first error when run with --fail" {
  mkdir -p "$out_dir"
  write_yaml_to_file "$templated_fragment" "$(mktemp "$out_dir/templated-01-XXXXXX-file1.yaml")"
  write_yaml_to_file "$templated_fragment" "$(mktemp "$out_dir/templated-02-XXXXXX-file2.yaml")"

  run roxctl-release netpol connectivity map "$out_dir/" --remove --output-file=/dev/null --fail
  assert_failure
  assert_line --index 0 --partial 'This is a Technology Preview feature'
  assert_line --index 1 --partial 'YAML document is malformed'  # expect only one line with this error
  assert_line --index 2 --partial 'there were errors during execution'  # last line
}

@test "roxctl-release netpol connectivity map produces no output when all yamls are templated" {
  mkdir -p "$out_dir"
  write_yaml_to_file "$templated_fragment" "$(mktemp "$out_dir/templated-XXXXXX.yaml")"

  echo "Analyzing a corrupted yaml file '$templatedYaml'" >&3
  run roxctl-release netpol connectivity map "$out_dir/"
  assert_failure
  assert_output --partial 'YAML document is malformed'
  assert_output --partial 'no relevant Kubernetes resources found'
}

@test "roxctl-release netpol connectivity map produces errors when some yamls are templated" {
  mkdir -p "$out_dir"
  write_yaml_to_file "$templated_fragment" "$(mktemp "$out_dir/templated-XXXXXX.yaml")"

  assert_file_exist "${test_data}/np-guard/scenario-minimal-service/frontend.yaml"
  assert_file_exist "${test_data}/np-guard/scenario-minimal-service/backend.yaml"
  cp "${test_data}/np-guard/scenario-minimal-service/frontend.yaml" "$out_dir/frontend.yaml"
  cp "${test_data}/np-guard/scenario-minimal-service/backend.yaml" "$out_dir/backend.yaml"

  echo "Analyzing a directory where 1/3 of yaml files are templated '$out_dir/'" >&3
  run roxctl-release netpol connectivity map "$out_dir/" --remove --output-file=/dev/null
  assert_failure
  assert_output --partial 'YAML document is malformed'
  refute_output --partial 'no relevant Kubernetes resources found'
}

@test "roxctl-release netpol connectivity map produces errors when yamls are not K8s resources" {
  mkdir -p "$out_dir"
  assert_file_exist "${test_data}/np-guard/empty-yamls/empty.yaml"
  assert_file_exist "${test_data}/np-guard/empty-yamls/empty2.yaml"
  cp "${test_data}/np-guard/empty-yamls/empty.yaml" "$out_dir/empty.yaml"
  cp "${test_data}/np-guard/empty-yamls/empty2.yaml" "$out_dir/empty2.yaml"

  run roxctl-release netpol connectivity map "$out_dir/" --remove --output-file=/dev/null
  assert_failure
  assert_output --partial 'Yaml document is not a K8s resource'
  assert_output --partial 'no relevant Kubernetes resources found'
  assert_output --partial 'ERROR:'
  assert_output --partial 'there were errors during execution'
}

@test "roxctl-release netpol connectivity map should return error on invalid networkpolicy resource" {
  assert_file_exist "${test_data}/np-guard/bad-netpol-example/resources.yaml"
  run roxctl-release netpol connectivity map "${test_data}/np-guard/bad-netpol-example"
  assert_failure
  assert_line --partial "error in connectivity analysis"
  assert_line --partial "selector error"
}

@test "roxctl-release netpol connectivity map should return error on not supported output format" {
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/backend.yaml"
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/frontend.yaml"
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/netpols.yaml"
  run roxctl-release netpol connectivity map "${test_data}/np-guard/netpols-analysis-example-minimal" --output-format=docx
  assert_failure
  assert_line --partial "error in formatting connectivity list"
  assert_line --partial "docx output format is not supported."
}

@test "roxctl-release netpol connectivity map generates txt connlist output" {
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/backend.yaml"
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/frontend.yaml"
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/netpols.yaml"
  echo "Writing connlist to ${ofile}" >&3
  run roxctl-release netpol connectivity map "${test_data}/np-guard/netpols-analysis-example-minimal" --output-format=txt
  assert_success

  echo "$output" > "$ofile"
  assert_file_exist "$ofile"
  assert_output --partial 'default/frontend[Deployment] => default/backend[Deployment] : TCP 9090'
}

@test "roxctl-release netpol connectivity map generates json connlist output" {
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/backend.yaml"
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/frontend.yaml"
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/netpols.yaml"
  echo "Writing connlist to ${ofile}" >&3
  run roxctl-release netpol connectivity map "${test_data}/np-guard/netpols-analysis-example-minimal" --output-format=json
  assert_success

  echo "$output" > "$ofile"
  assert_file_exist "$ofile"
  assert_output --partial '{
    "src": "0.0.0.0-255.255.255.255",
    "dst": "default/frontend[Deployment]",
    "conn": "TCP 8080"
  },'
}

@test "roxctl-release netpol connectivity map generates md connlist output" {
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/backend.yaml"
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/frontend.yaml"
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/netpols.yaml"
  echo "Writing connlist to ${ofile}" >&3
  run roxctl-release netpol connectivity map "${test_data}/np-guard/netpols-analysis-example-minimal" --output-format=md
  assert_success

  echo "$output" > "$ofile"
  assert_file_exist "$ofile"
  assert_output --partial '0.0.0.0-255.255.255.255 | default/frontend[Deployment] | TCP 8080 |'
}

@test "roxctl-release netpol connectivity map generates dot connlist output" {
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/backend.yaml"
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/frontend.yaml"
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/netpols.yaml"
  echo "Writing connlist to ${ofile}" >&3
  run roxctl-release netpol connectivity map "${test_data}/np-guard/netpols-analysis-example-minimal" --output-format=dot
  assert_success

  echo "$output" > "$ofile"
  assert_file_exist "$ofile"
  assert_output --partial '"0.0.0.0-255.255.255.255" [label="0.0.0.0-255.255.255.255" color="red2" fontcolor="red2"]'
}

@test "roxctl-release netpol connectivity map generates csv connlist output" {
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/backend.yaml"
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/frontend.yaml"
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/netpols.yaml"
  echo "Writing connlist to ${ofile}" >&3
  run roxctl-release netpol connectivity map "${test_data}/np-guard/netpols-analysis-example-minimal" --output-format=csv
  assert_success

  echo "$output" > "$ofile"
  assert_file_exist "$ofile"
  assert_output --partial '0.0.0.0-255.255.255.255,default/frontend[Deployment],TCP 8080'
}

@test "roxctl-release netpol connectivity map skips unsupported openshift resources" {
  assert_file_exist "${test_data}/np-guard/irrelevant-oc-resource-example/irrelevant_oc_resource.yaml"
  echo "Writing connlist to ${ofile}" >&3
  run roxctl-release netpol connectivity map "${test_data}/np-guard/irrelevant-oc-resource-example"
  assert_failure # no workload nor network-policy resources

  echo "$output" > "$ofile"
  assert_file_exist "$ofile"
  assert_output --partial 'skipping object with type: SecurityContextConstraints'
}

@test "roxctl-release netpol connectivity map generates focused to workload connlist output" {
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/backend.yaml"
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/frontend.yaml"
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/netpols.yaml"
  echo "Writing connlist to ${ofile}" >&3
  run roxctl-release netpol connectivity map "${test_data}/np-guard/netpols-analysis-example-minimal" --focus-workload=backend
  assert_success

  echo "$output" > "$ofile"
  assert_file_exist "$ofile"
  # partial here is used to filter the WARN and INFO messages
  assert_output --partial 'default/frontend[Deployment] => default/backend[Deployment] : TCP 9090'
  refute_output --partial 'default/frontend[Deployment] => 0.0.0.0-255.255.255.255 : UDP 53'
}

@test "roxctl-release netpol connectivity map generates connlist to specific txt output file" {
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/backend.yaml"
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/frontend.yaml"
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/netpols.yaml"
  run roxctl-release netpol connectivity map "${test_data}/np-guard/netpols-analysis-example-minimal" --output-file="$out_dir/out.txt"
  assert_success

  assert_file_exist "$out_dir/out.txt"
  assert_output --partial 'default/frontend[Deployment] => default/backend[Deployment] : TCP 9090'
}

@test "roxctl-release netpol connectivity map generates empty connlist netpols blocks ingress connections from Routes" {
  frontend_sec_dir="${BATS_TEST_DIRNAME}/../../../../roxctl/netpol/connectivity/map/testdata/frontend-security"
  assert_file_exist "${frontend_sec_dir}/asset-cache-deployment.yaml"
  assert_file_exist "${frontend_sec_dir}/asset-cache-route.yaml"
  assert_file_exist "${frontend_sec_dir}/frontend-netpols.yaml"
  assert_file_exist "${frontend_sec_dir}/webapp-deployment.yaml"
  assert_file_exist "${frontend_sec_dir}/webapp-route.yaml"
  run roxctl-release netpol connectivity map "${frontend_sec_dir}"
  assert_success
  # netpols deny connections between the existing deployments; and blocks ingress from external ips or ingress-controller
  # the output contains only WARN and INFO messages
  echo "$output" > "$ofile"
  assert_file_exist "$ofile"
  assert_output --partial 'Route resource frontend/asset-cache specified workload frontend/asset-cache[Deployment] as a backend, but network policies are blocking ingress connections from an arbitrary in-cluster source to this workload.'
}

# following const is used as the directory path of the next tests
acs_security_demos_dir="${BATS_TEST_DIRNAME}/../../../../roxctl/netpol/connectivity/map/testdata/acs-security-demos"
@test "roxctl-release netpol connectivity map generates connlist for acs-security-demo" {
  check_acs_security_demos_files
  run roxctl-release netpol connectivity map "${acs_security_demos_dir}"
  assert_success

  echo "$output" > "$ofile"
  assert_file_exist "$ofile"
  # partial is used to filter WARN and INFO messages
  assert_output --partial 'backend/checkout[Deployment] => backend/notification[Deployment] : TCP 8080
backend/checkout[Deployment] => backend/recommendation[Deployment] : TCP 8080
backend/checkout[Deployment] => payments/gateway[Deployment] : TCP 8080
backend/recommendation[Deployment] => backend/catalog[Deployment] : TCP 8080
backend/reports[Deployment] => backend/catalog[Deployment] : TCP 8080
backend/reports[Deployment] => backend/recommendation[Deployment] : TCP 8080
frontend/webapp[Deployment] => backend/checkout[Deployment] : TCP 8080
frontend/webapp[Deployment] => backend/recommendation[Deployment] : TCP 8080
frontend/webapp[Deployment] => backend/reports[Deployment] : TCP 8080
frontend/webapp[Deployment] => backend/shipping[Deployment] : TCP 8080
payments/gateway[Deployment] => payments/mastercard-processor[Deployment] : TCP 8080
payments/gateway[Deployment] => payments/visa-processor[Deployment] : TCP 8080
{ingress-controller} => frontend/asset-cache[Deployment] : TCP 8080
{ingress-controller} => frontend/webapp[Deployment] : TCP 8080'
}

@test "roxctl-release netpol connectivity map generates connlist for acs-security-demo md format" {
  check_acs_security_demos_files
  run roxctl-release netpol connectivity map "${acs_security_demos_dir}" --output-format=md
  assert_success

  echo "$output" > "$ofile"
  assert_file_exist "$ofile"
  # output lines , skipping WARN and INFO messages
  assert_output --partial '| src | dst | conn |
|-----|-----|------|
| backend/checkout[Deployment] | backend/notification[Deployment] | TCP 8080 |
| backend/checkout[Deployment] | backend/recommendation[Deployment] | TCP 8080 |
| backend/checkout[Deployment] | payments/gateway[Deployment] | TCP 8080 |
| backend/recommendation[Deployment] | backend/catalog[Deployment] | TCP 8080 |
| backend/reports[Deployment] | backend/catalog[Deployment] | TCP 8080 |
| backend/reports[Deployment] | backend/recommendation[Deployment] | TCP 8080 |
| frontend/webapp[Deployment] | backend/checkout[Deployment] | TCP 8080 |
| frontend/webapp[Deployment] | backend/recommendation[Deployment] | TCP 8080 |
| frontend/webapp[Deployment] | backend/reports[Deployment] | TCP 8080 |
| frontend/webapp[Deployment] | backend/shipping[Deployment] | TCP 8080 |
| payments/gateway[Deployment] | payments/mastercard-processor[Deployment] | TCP 8080 |
| payments/gateway[Deployment] | payments/visa-processor[Deployment] | TCP 8080 |
| {ingress-controller} | frontend/asset-cache[Deployment] | TCP 8080 |
| {ingress-controller} | frontend/webapp[Deployment] | TCP 8080 |'
}

@test "roxctl-release netpol connectivity map generates connlist for acs-security-demo with focus-workload=gateway" {
  check_acs_security_demos_files
  run roxctl-release netpol connectivity map "${acs_security_demos_dir}" --focus-workload=gateway
  assert_success
  echo "$output" > "$ofile"
  assert_file_exist "$ofile"
  assert_output --partial 'backend/checkout[Deployment] => payments/gateway[Deployment] : TCP 8080
payments/gateway[Deployment] => payments/mastercard-processor[Deployment] : TCP 8080
payments/gateway[Deployment] => payments/visa-processor[Deployment] : TCP 8080'
  refute_output --partial 'frontend/webapp[Deployment] => backend/shipping[Deployment] : TCP 8080'
  refute_output --partial '{ingress-controller} => frontend/asset-cache[Deployment] : TCP 8080'
}

@test "roxctl-release netpol connectivity map generates connlist for acs-security-demo with focus-workload=payments/gateway" {
  check_acs_security_demos_files
  run roxctl-release netpol connectivity map "${acs_security_demos_dir}" --focus-workload=payments/gateway
  assert_success
  echo "$output" > "$ofile"
  assert_file_exist "$ofile"
  assert_output --partial 'backend/checkout[Deployment] => payments/gateway[Deployment] : TCP 8080
payments/gateway[Deployment] => payments/mastercard-processor[Deployment] : TCP 8080
payments/gateway[Deployment] => payments/visa-processor[Deployment] : TCP 8080'
  refute_output --partial 'frontend/webapp[Deployment] => backend/shipping[Deployment] : TCP 8080'
  refute_output --partial '{ingress-controller} => frontend/asset-cache[Deployment] : TCP 8080'
}

@test "roxctl-release netpol connectivity map generates connlist for acs-security-demo with focus-workload that does not exist" {
  check_acs_security_demos_files
  run roxctl-release netpol connectivity map "${acs_security_demos_dir}" --focus-workload=abc
  assert_success
  echo "$output" > "$ofile"
  assert_file_exist "$ofile"
  assert_output --partial 'Workload abc does not exist in the input resources. Connectivity map report will be empty.'
}

@test "roxctl-release netpol connectivity map generates connlist for acs-security-demo with focus-workload=ingress-controller" {
  check_acs_security_demos_files
  run roxctl-release netpol connectivity map "${acs_security_demos_dir}" --focus-workload=ingress-controller
  assert_success
  echo "$output" > "$ofile"
  assert_file_exist "$ofile"
  assert_output --partial '{ingress-controller} => frontend/asset-cache[Deployment] : TCP 8080
{ingress-controller} => frontend/webapp[Deployment] : TCP 8080'
  refute_output --partial 'frontend/webapp[Deployment] => backend/shipping[Deployment] : TCP 8080'
}

check_acs_security_demos_files() {
  assert_file_exist "${acs_security_demos_dir}/backend/catalog/deployment.yaml"
  assert_file_exist "${acs_security_demos_dir}/backend/checkout/configmap.yaml"
  assert_file_exist "${acs_security_demos_dir}/backend/checkout/deployment.yaml"
  assert_file_exist "${acs_security_demos_dir}/backend/notification/deployment.yaml"
  assert_file_exist "${acs_security_demos_dir}/backend/recommendation/configmap.yaml"
  assert_file_exist "${acs_security_demos_dir}/backend/recommendation/deployment.yaml"
  assert_file_exist "${acs_security_demos_dir}/backend/reports/configmap.yaml"
  assert_file_exist "${acs_security_demos_dir}/backend/reports/deployment.yaml"
  assert_file_exist "${acs_security_demos_dir}/backend/shipping/deployment.yaml"
  assert_file_exist "${acs_security_demos_dir}/frontend/asset-cache/deployment.yaml"
  assert_file_exist "${acs_security_demos_dir}/frontend/asset-cache/route.yaml"
  assert_file_exist "${acs_security_demos_dir}/frontend/webapp/configmap.yaml"
  assert_file_exist "${acs_security_demos_dir}/frontend/webapp/deployment.yaml"
  assert_file_exist "${acs_security_demos_dir}/frontend/webapp/route.yaml"
  assert_file_exist "${acs_security_demos_dir}/payments/gateway/deployment.yaml"
  assert_file_exist "${acs_security_demos_dir}/payments/mastercard-processor/deployment.yaml"
  assert_file_exist "${acs_security_demos_dir}/payments/visa-processor/deployment.yaml"
  assert_file_exist "${acs_security_demos_dir}/acs_netpols.yaml"
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
