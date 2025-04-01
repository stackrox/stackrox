#!/usr/bin/env bats

load "../helpers.bash"
out_dir=""
templated_fragment='"{{ printf "%s" ._thing.image }}"'

setup_file() {
    delete-outdated-binaries "$(roxctl-release version)"
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
    assert_output --regexp 'WARN:.*no relevant Kubernetes network policy resources found'
    assert_output --regexp 'ERROR:.*the path.*does not exist'
    assert_output --regexp 'ERROR:.*no relevant Kubernetes workload resources found'
    assert_output --regexp 'ERROR:.*building connectivity map: there were errors during execution'

    run roxctl-release netpol connectivity map
    assert_failure
    assert_line --partial "accepts 1 arg(s), received 0"
}

@test "roxctl-release netpol connectivity map should return error on directory with no files" {
    mkdir -p "$out_dir"
    run roxctl-release netpol connectivity map "$out_dir"
    assert_failure
    assert_output --regexp 'WARN:.*no relevant Kubernetes network policy resources found'
    assert_output --regexp 'ERROR:.*no relevant Kubernetes workload resources found'
    assert_output --regexp 'ERROR:.*building connectivity map: there were errors during execution'
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

@test "roxctl-release netpol connectivity shows all warnings about corrupted files" {
    mkdir -p "$out_dir"
    assert_file_exist "${test_data}/np-guard/mixed/backend.yaml"
    assert_file_exist "${test_data}/np-guard/mixed/frontend.yaml"
    assert_file_exist "${test_data}/np-guard/mixed/netpols.yaml"
    assert_file_exist "${test_data}/np-guard/mixed/empty.yaml"
    cp "${test_data}/np-guard/mixed/backend.yaml" "$out_dir/backend.yaml"
    cp "${test_data}/np-guard/mixed/frontend.yaml" "$out_dir/frontend.yaml"
    cp "${test_data}/np-guard/mixed/netpols.yaml" "$out_dir/netpols.yaml"
    cp "${test_data}/np-guard/mixed/empty.yaml" "$out_dir/empty.yaml"

    run roxctl-release netpol connectivity map "$out_dir/" --remove --output-file=/dev/null
    assert_success
    assert_output --regexp 'WARN:.*unable to decode.*empty.yaml'
    assert_output --regexp "WARN:.*empty.yaml\": Object 'Kind' is missing in"
}

@test "roxctl-release netpol connectivity map parameter --strict" {
    mkdir -p "$out_dir"
    assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/backend.yaml"
    assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/frontend.yaml"
    assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/netpols.yaml"
    cp "${test_data}/np-guard/netpols-analysis-example-minimal/backend.yaml" "$out_dir/backend.yaml"
    cp "${test_data}/np-guard/netpols-analysis-example-minimal/frontend.yaml" "$out_dir/frontend.yaml"
    cp "${test_data}/np-guard/netpols-analysis-example-minimal/netpols.yaml" "$out_dir/netpols.yaml"
    write_yaml_to_file "$templated_fragment" "$(mktemp "$out_dir/templated-01-XXXXXX-file1.yaml")"
    write_yaml_to_file "$templated_fragment" "$(mktemp "$out_dir/templated-02-XXXXXX-file2.yaml")"

    run roxctl-release netpol connectivity map "$out_dir/" --strict
    assert_failure
    assert_output --regexp 'WARN:.*unable to decode.*-file1.yaml'
    assert_output --regexp 'WARN:.*unable to decode.*-file2.yaml'
    assert_output --regexp 'ERROR:.*there were warnings during execution'

    run roxctl-release netpol connectivity map "$out_dir/"
    assert_success
    assert_output --regexp 'WARN:.*unable to decode.*-file1.yaml'
    assert_output --regexp 'WARN:.*unable to decode.*-file2.yaml'
}

# TODO: It is difficult to find any scenario for this command when --fail alone would cause change of behavior
@test "roxctl-release netpol connectivity map parameter --fail" {
    mkdir -p "$out_dir"
    assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/backend.yaml"
    assert_file_exist "${test_data}/np-guard/mixed/empty.yaml"
    cp "${test_data}/np-guard/netpols-analysis-example-minimal/backend.yaml" "$out_dir/backend.yaml"
    cp "${test_data}/np-guard/mixed/empty.yaml" "$out_dir/empty.yaml"
    cp "${test_data}/np-guard/mixed/empty.yaml" "$out_dir/empty2.yaml"

    run roxctl-release netpol connectivity map "$out_dir/"
    assert_success
    assert_output --regexp 'WARN:.*unable to decode.*/empty.yaml'
    assert_output --regexp 'WARN:.*unable to decode.*/empty2.yaml'
    assert_output --regexp 'WARN:.*no relevant Kubernetes network policy resources found'

    run roxctl-release netpol connectivity map "$out_dir/" --fail
    assert_success
    assert_output --regexp 'WARN:.*unable to decode.*/empty.yaml'
    assert_output --regexp 'WARN:.*unable to decode.*/empty2.yaml'
    assert_output --regexp 'WARN:.*no relevant Kubernetes network policy resources found'
}

@test "roxctl-release netpol connectivity map parameter --fail and --strict" {
    mkdir -p "$out_dir"
    assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/backend.yaml"
    assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/frontend.yaml"
    assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/netpols.yaml"
    cp "${test_data}/np-guard/netpols-analysis-example-minimal/backend.yaml" "$out_dir/backend.yaml"
    cp "${test_data}/np-guard/netpols-analysis-example-minimal/frontend.yaml" "$out_dir/frontend.yaml"
    cp "${test_data}/np-guard/netpols-analysis-example-minimal/netpols.yaml" "$out_dir/netpols.yaml"
    write_yaml_to_file "$templated_fragment" "$(mktemp "$out_dir/templated-01-XXXXXX-file1.yaml")"
    write_yaml_to_file "$templated_fragment" "$(mktemp "$out_dir/templated-02-XXXXXX-file2.yaml")"

    run roxctl-release netpol connectivity map "$out_dir/" --fail --strict
    assert_failure
    assert_output --regexp 'ERROR:.*building connectivity map: there were warnings during execution'
    assert_output --regexp 'WARN:.*unable to decode.*-file1.yaml'
    # should fail fast before trying to decode file2 due to warnings when processing file1
    refute_output --regexp 'WARN:.*unable to decode.*-file2.yaml'
}

@test "roxctl-release netpol connectivity map produces no output when all yamls are templated" {
  mkdir -p "$out_dir"
  write_yaml_to_file "$templated_fragment" "$(mktemp "$out_dir/templated-XXXXXX.yaml")"

  echo "Analyzing a corrupted yaml file '$templatedYaml'" >&3
  run roxctl-release netpol connectivity map "$out_dir/"
  assert_failure
  assert_output --regexp 'WARN:.*unable to decode.*templated-.*.yaml'
  assert_output --regexp 'WARN:.*error parsing.*templated-.*.yaml'
  assert_output --regexp 'WARN:.*no relevant Kubernetes network policy resources found'
  assert_output --regexp 'ERROR:.*no relevant Kubernetes workload resources found'
  assert_output --regexp 'ERROR:.*building connectivity map: there were errors during execution'
}

@test "roxctl-release netpol connectivity map produces warnings when some yamls are templated" {
    mkdir -p "$out_dir"
    write_yaml_to_file "$templated_fragment" "$(mktemp "$out_dir/templated-XXXXXX.yaml")"

    assert_file_exist "${test_data}/np-guard/scenario-minimal-service/frontend.yaml"
    assert_file_exist "${test_data}/np-guard/scenario-minimal-service/backend.yaml"
    cp "${test_data}/np-guard/scenario-minimal-service/frontend.yaml" "$out_dir/frontend.yaml"
    cp "${test_data}/np-guard/scenario-minimal-service/backend.yaml" "$out_dir/backend.yaml"

    echo "Analyzing a directory where 1/3 of yaml files are templated '$out_dir/'" >&3
    run roxctl-release netpol connectivity map "$out_dir/" --remove --output-file=/dev/null
    assert_success
    assert_output --regexp 'WARN:.*unable to decode.*templated-.*.yaml'
    assert_output --regexp 'WARN:.*error parsing.*templated-.*.yaml'
    assert_output --regexp 'WARN:.*no relevant Kubernetes network policy resources found'

    refute_output --regexp 'ERROR:.*no relevant Kubernetes workload resources found'
    refute_output --regexp 'ERROR:.*building connectivity map:'

    assert_output --partial 'default/frontend[Deployment] => default/backend[Deployment] : All Connections'
}

@test "roxctl-release netpol connectivity map produces warnings when yamls are not K8s resources" {
    mkdir -p "$out_dir"
    assert_file_exist "${test_data}/np-guard/empty-yamls/empty.yaml"
    assert_file_exist "${test_data}/np-guard/empty-yamls/empty2.yaml"
    cp "${test_data}/np-guard/empty-yamls/empty.yaml" "$out_dir/empty.yaml"
    cp "${test_data}/np-guard/empty-yamls/empty2.yaml" "$out_dir/empty2.yaml"

    run roxctl-release netpol connectivity map "$out_dir/" --remove --output-file=/dev/null
    assert_failure
    assert_output --regexp 'WARN:.*unable to decode.*empty.yaml'
    refute_output --regexp 'WARN:.*error parsing.*empty.yaml'
    assert_output --regexp 'WARN:.*unable to decode.*empty2.yaml'
    refute_output --regexp 'WARN:.*error parsing.*empty2.yaml'

    assert_output --regexp 'WARN:.*no relevant Kubernetes network policy resources found'
    assert_output --regexp 'ERROR:.*no relevant Kubernetes workload resources found'
    assert_output --regexp 'ERROR:.*building connectivity map: there were errors during execution'
}

@test "roxctl-release netpol connectivity map should return error on invalid networkpolicy resource" {
    assert_file_exist "${test_data}/np-guard/bad-netpol-example/resources.yaml"
    run roxctl-release netpol connectivity map "${test_data}/np-guard/bad-netpol-example"
    assert_failure
    assert_line --partial "selector error"
    assert_output --regexp 'ERROR:.*connectivity analysis:.*'
    assert_output --regexp 'ERROR:.*building connectivity map: there were errors during execution'
}

@test "roxctl-release netpol connectivity map should return error on not supported output format" {
    assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/backend.yaml"
    assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/frontend.yaml"
    assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/netpols.yaml"
    run roxctl-release netpol connectivity map "${test_data}/np-guard/netpols-analysis-example-minimal" --output-format=docx
    assert_failure
    assert_output --regexp 'ERROR:.*formatting connectivity list: docx output format is not supported.'
    assert_output --regexp 'ERROR:.*building connectivity map: there were errors during execution'
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
  assert_output --partial 'Workload abc does not exist in the input resources'
  assert_output --partial 'Connectivity map report will be empty.'
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

@test "roxctl-release netpol connectivity map generates connlist with exposure-analysis for acs-security-demo" {
  check_acs_security_demos_files
  run roxctl-release netpol connectivity map "${acs_security_demos_dir}" --exposure
  assert_success
  echo "$output" > "$ofile"
  assert_file_exist "$ofile"
  # normalizing tabs and whitespaces in output so it will be easier to compare with expected
  output=$(normalize_whitespaces "$output")
  expected_output='backend/checkout[Deployment] => backend/notification[Deployment] : TCP 8080
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
{ingress-controller} => frontend/webapp[Deployment] : TCP 8080

Exposure Analysis Result:
Egress Exposure:
backend/checkout[Deployment]            =>      entire-cluster : UDP 5353
backend/recommendation[Deployment]      =>      entire-cluster : UDP 5353
backend/reports[Deployment]             =>      entire-cluster : UDP 5353
frontend/webapp[Deployment]             =>      entire-cluster : UDP 5353
payments/gateway[Deployment]            =>      entire-cluster : UDP 5353

Ingress Exposure:
frontend/asset-cache[Deployment]        <=      entire-cluster : TCP 8080
frontend/webapp[Deployment]             <=      entire-cluster : TCP 8080'
  normalized_expected_output=$(normalize_whitespaces "$expected_output")
  # partial is used to filter WARN and INFO messages
  assert_output --partial "$normalized_expected_output"
}

@test "roxctl-release netpol connectivity map generates connlist with exposure-analysis for acs-security-demo md format" {
  check_acs_security_demos_files
  run roxctl-release netpol connectivity map "${acs_security_demos_dir}" --output-format=md --exposure
  assert_success

  echo "$output" > "$ofile"
  assert_file_exist "$ofile"
  # output lines , skipping connlist and WARN and INFO messages
  assert_output --partial '## Exposure Analysis Result:
### Egress Exposure:
| src | dst | conn |
|-----|-----|------|
| backend/checkout[Deployment] | entire-cluster | UDP 5353 |
| backend/recommendation[Deployment] | entire-cluster | UDP 5353 |
| backend/reports[Deployment] | entire-cluster | UDP 5353 |
| frontend/webapp[Deployment] | entire-cluster | UDP 5353 |
| payments/gateway[Deployment] | entire-cluster | UDP 5353 |

### Ingress Exposure:
| dst | src | conn |
|-----|-----|------|
| frontend/asset-cache[Deployment] | entire-cluster | TCP 8080 |
| frontend/webapp[Deployment] | entire-cluster | TCP 8080 |'
}

@test "roxctl-release netpol connectivity map generates connlist with exposure-analysis for acs-security-demo dot format" {
  check_acs_security_demos_files
  run roxctl-release netpol connectivity map "${acs_security_demos_dir}" --output-format=dot --exposure
  assert_success

  echo "$output" > "$ofile"
  assert_file_exist "$ofile"
  # normalizing tabs and whitespaces in output so it will be easier to compare with expected
  output=$(normalize_whitespaces "$output")
  expected_output='digraph {
        subgraph "cluster_backend" {
                color="black"
                fontcolor="black"
                "backend/catalog[Deployment]" [label="catalog[Deployment]" color="blue" fontcolor="blue"]
                "backend/checkout[Deployment]" [label="checkout[Deployment]" color="blue" fontcolor="blue"]
                "backend/notification[Deployment]" [label="notification[Deployment]" color="blue" fontcolor="blue"]
                "backend/recommendation[Deployment]" [label="recommendation[Deployment]" color="blue" fontcolor="blue"]
                "backend/reports[Deployment]" [label="reports[Deployment]" color="blue" fontcolor="blue"]
                "backend/shipping[Deployment]" [label="shipping[Deployment]" color="blue" fontcolor="blue"]
                label="backend"
        }
        subgraph "cluster_frontend" {
                color="black"
                fontcolor="black"
                "frontend/asset-cache[Deployment]" [label="asset-cache[Deployment]" color="blue" fontcolor="blue"]
                "frontend/webapp[Deployment]" [label="webapp[Deployment]" color="blue" fontcolor="blue"]
                label="frontend"
        }
        subgraph "cluster_payments" {
                color="black"
                fontcolor="black"
                "payments/gateway[Deployment]" [label="gateway[Deployment]" color="blue" fontcolor="blue"]
                "payments/mastercard-processor[Deployment]" [label="mastercard-processor[Deployment]" color="blue" fontcolor="blue"]
                "payments/visa-processor[Deployment]" [label="visa-processor[Deployment]" color="blue" fontcolor="blue"]
                label="payments"
        }
        "entire-cluster" [label="entire-cluster" color="red2" fontcolor="red2" shape=diamond]
        "{ingress-controller}" [label="{ingress-controller}" color="blue" fontcolor="blue"]
        "backend/checkout[Deployment]" -> "backend/notification[Deployment]" [label="TCP 8080" color="gold2" fontcolor="darkgreen" weight=0.5]
        "backend/checkout[Deployment]" -> "backend/recommendation[Deployment]" [label="TCP 8080" color="gold2" fontcolor="darkgreen" weight=0.5]
        "backend/checkout[Deployment]" -> "entire-cluster" [label="UDP 5353" color="darkorange4" fontcolor="darkgreen" weight=0.5 style=dashed]
        "backend/checkout[Deployment]" -> "payments/gateway[Deployment]" [label="TCP 8080" color="gold2" fontcolor="darkgreen" weight=0.5]
        "backend/recommendation[Deployment]" -> "backend/catalog[Deployment]" [label="TCP 8080" color="gold2" fontcolor="darkgreen" weight=1]
        "backend/recommendation[Deployment]" -> "entire-cluster" [label="UDP 5353" color="darkorange4" fontcolor="darkgreen" weight=0.5 style=dashed]
        "backend/reports[Deployment]" -> "backend/catalog[Deployment]" [label="TCP 8080" color="gold2" fontcolor="darkgreen" weight=1]
        "backend/reports[Deployment]" -> "backend/recommendation[Deployment]" [label="TCP 8080" color="gold2" fontcolor="darkgreen" weight=1]
        "backend/reports[Deployment]" -> "entire-cluster" [label="UDP 5353" color="darkorange4" fontcolor="darkgreen" weight=0.5 style=dashed]
        "entire-cluster" -> "frontend/asset-cache[Deployment]" [label="TCP 8080" color="darkorange2" fontcolor="darkgreen" weight=1 style=dashed]
        "entire-cluster" -> "frontend/webapp[Deployment]" [label="TCP 8080" color="darkorange2" fontcolor="darkgreen" weight=1 style=dashed]
        "frontend/webapp[Deployment]" -> "backend/checkout[Deployment]" [label="TCP 8080" color="gold2" fontcolor="darkgreen" weight=1]
        "frontend/webapp[Deployment]" -> "backend/recommendation[Deployment]" [label="TCP 8080" color="gold2" fontcolor="darkgreen" weight=1]
        "frontend/webapp[Deployment]" -> "backend/reports[Deployment]" [label="TCP 8080" color="gold2" fontcolor="darkgreen" weight=1]
        "frontend/webapp[Deployment]" -> "backend/shipping[Deployment]" [label="TCP 8080" color="gold2" fontcolor="darkgreen" weight=1]
        "frontend/webapp[Deployment]" -> "entire-cluster" [label="UDP 5353" color="darkorange4" fontcolor="darkgreen" weight=0.5 style=dashed]
        "payments/gateway[Deployment]" -> "entire-cluster" [label="UDP 5353" color="darkorange4" fontcolor="darkgreen" weight=0.5 style=dashed]
        "payments/gateway[Deployment]" -> "payments/mastercard-processor[Deployment]" [label="TCP 8080" color="gold2" fontcolor="darkgreen" weight=0.5]
        "payments/gateway[Deployment]" -> "payments/visa-processor[Deployment]" [label="TCP 8080" color="gold2" fontcolor="darkgreen" weight=0.5]
        "{ingress-controller}" -> "frontend/asset-cache[Deployment]" [label="TCP 8080" color="gold2" fontcolor="darkgreen" weight=1]
        "{ingress-controller}" -> "frontend/webapp[Deployment]" [label="TCP 8080" color="gold2" fontcolor="darkgreen" weight=1]
}'
  normalized_expected_output=$(normalize_whitespaces "$expected_output")
  # partial is used to filter WARN and INFO messages
  assert_output --partial "$normalized_expected_output"
}

@test "roxctl-release netpol connectivity map generates exposure for acs-security-demo with focus-workload=gateway" {
  check_acs_security_demos_files
  run roxctl-release netpol connectivity map "${acs_security_demos_dir}" --focus-workload=gateway --exposure
  assert_success
  echo "$output" > "$ofile"
  assert_file_exist "$ofile"
  # normalizing tabs and whitespaces in output so it will be easier to compare with expected
  output=$(normalize_whitespaces "$output")
  expected_output='backend/checkout[Deployment] => payments/gateway[Deployment] : TCP 8080
payments/gateway[Deployment] => payments/mastercard-processor[Deployment] : TCP 8080
payments/gateway[Deployment] => payments/visa-processor[Deployment] : TCP 8080

Exposure Analysis Result:
Egress Exposure:
payments/gateway[Deployment]    =>      entire-cluster : UDP 5353'
  normalized_expected_output=$(normalize_whitespaces "$expected_output")
  # partial is used to filter WARN and INFO messages
  assert_output --partial "$normalized_expected_output"
}

@test "roxctl-release netpol connectivity map generates exposure from certain Namespace labels and Pod labels specified" {
  assert_file_exist "${test_data}/np-guard/exposure-example/netpol.yaml"
  assert_file_exist "${test_data}/np-guard/exposure-example/ns_and_deployments.yaml"
  echo "Writing exposure report to ${ofile}" >&3
  run roxctl-release netpol connectivity map "${test_data}/np-guard/exposure-example" --exposure
  assert_success

  echo "$output" > "$ofile"
  assert_file_exist "$ofile"
  # normalizing tabs and whitespaces in output so it will be easier to compare with expected
  output=$(normalize_whitespaces "$output")
  expected_output='hello-world/workload-a[Deployment] => 0.0.0.0-255.255.255.255 : All Connections

Exposure Analysis Result:
Egress Exposure:
hello-world/workload-a[Deployment]      =>      0.0.0.0-255.255.255.255 : All Connections
hello-world/workload-a[Deployment]      =>      entire-cluster : All Connections

Ingress Exposure:
hello-world/workload-a[Deployment]      <=      [namespace with {effect=NoSchedule}]/[pod with {role=monitoring}] : TCP 8050

Workloads not protected by network policies:
hello-world/workload-a[Deployment] is not protected on Egress'
  normalized_expected_output=$(normalize_whitespaces "$expected_output")
  assert_output "$normalized_expected_output"
}

@test "roxctl-release netpol connectivity map generates connlist for input resources with admin network policies" {
  assert_file_exist "${test_data}/np-guard/anp_banp_demo/ns.yaml"
  assert_file_exist "${test_data}/np-guard/anp_banp_demo/policies.yaml"
  assert_file_exist "${test_data}/np-guard/anp_banp_demo/workloads.yaml"

  echo "Writing connlist report to ${ofile}" >&3
  run roxctl-release netpol connectivity map "${test_data}/np-guard/anp_banp_demo"
  assert_success

  echo "$output" > "$ofile"
  assert_file_exist "$ofile"
  # normalizing tabs and whitespaces in output so it will be easier to compare with expected
  output=$(normalize_whitespaces "$output")
  expected_output='0.0.0.0-255.255.255.255 => bar/mybar[Pod] : All Connections
0.0.0.0-255.255.255.255 => baz/mybaz[Pod] : All Connections
0.0.0.0-255.255.255.255 => monitoring/mymonitoring[Pod] : All Connections
bar/mybar[Pod] => 0.0.0.0-255.255.255.255 : All Connections
bar/mybar[Pod] => baz/mybaz[Pod] : All Connections
bar/mybar[Pod] => monitoring/mymonitoring[Pod] : All Connections
baz/mybaz[Pod] => 0.0.0.0-255.255.255.255 : All Connections
baz/mybaz[Pod] => monitoring/mymonitoring[Pod] : All Connections
foo/myfoo[Pod] => 0.0.0.0-255.255.255.255 : All Connections
foo/myfoo[Pod] => baz/mybaz[Pod] : All Connections
foo/myfoo[Pod] => monitoring/mymonitoring[Pod] : All Connections
monitoring/mymonitoring[Pod] => 0.0.0.0-255.255.255.255 : All Connections
monitoring/mymonitoring[Pod] => baz/mybaz[Pod] : All Connections
monitoring/mymonitoring[Pod] => foo/myfoo[Pod] : All Connections'
  normalized_expected_output=$(normalize_whitespaces "$expected_output")
  assert_output "$normalized_expected_output"
}

@test "roxctl-release netpol connectivity map generates explainability report" {
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/backend.yaml"
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/frontend.yaml"
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/netpols.yaml"
  echo "Writing explainability to ${ofile}" >&3
  run roxctl-release netpol connectivity map "${test_data}/np-guard/netpols-analysis-example-minimal" --explain
  assert_success

  echo "$output" > "$ofile"
  assert_file_exist "$ofile"
  # normalizing tabs and whitespaces in output so it will be easier to compare with expected
  output=$(normalize_whitespaces "$output")
  # partial output - explaining connections between pair of the input peers
  partial_expected_output="""Connections between default/backend[Deployment] => default/frontend[Deployment]:

Denied connections:
        Denied TCP:[1-8079,8081-65535], UDP, SCTP due to the following policies and rules:
                Egress (Denied)
                        NetworkPolicy list:
                                - NetworkPolicy 'default/backend-netpol' selects default/backend[Deployment], but default/frontend[Deployment] is not allowed by any Egress rule (no rules defined)
                                - NetworkPolicy 'default/default-deny-in-namespace' selects default/backend[Deployment], but default/frontend[Deployment] is not allowed by any Egress rule (no rules defined)

                Ingress (Denied)
                        NetworkPolicy list:
                                - NetworkPolicy 'default/default-deny-in-namespace' selects default/frontend[Deployment], but default/backend[Deployment] is not allowed by any Ingress rule (no rules defined)
                                - NetworkPolicy 'default/frontend-netpol' selects default/frontend[Deployment], and Ingress rule #1 selects default/backend[Deployment], but the protocols and ports do not match


        Denied TCP:[8080] due to the following policies and rules:
                Egress (Denied)
                        NetworkPolicy list:
                                - NetworkPolicy 'default/backend-netpol' selects default/backend[Deployment], but default/frontend[Deployment] is not allowed by any Egress rule (no rules defined)
                                - NetworkPolicy 'default/default-deny-in-namespace' selects default/backend[Deployment], but default/frontend[Deployment] is not allowed by any Egress rule (no rules defined)

                Ingress (Allowed)
                        NetworkPolicy 'default/frontend-netpol' allows connections by Ingress rule #1"""
  normalized_expected_output=$(normalize_whitespaces "$partial_expected_output")
  assert_output --partial "$normalized_expected_output"
}

@test "roxctl-release netpol connectivity map ignores explain flag for unsupported md format with warning" {
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/backend.yaml"
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/frontend.yaml"
  assert_file_exist "${test_data}/np-guard/netpols-analysis-example-minimal/netpols.yaml"
  echo "Writing explainability to ${ofile}" >&3
  run roxctl-release netpol connectivity map "${test_data}/np-guard/netpols-analysis-example-minimal" --explain --output-format=md
  assert_success

  echo "$output" > "$ofile"
  assert_file_exist "$ofile"
   # normalizing tabs and whitespaces in output so it will be easier to compare with expected
  output=$(normalize_whitespaces "$output")
  # output contains a warn since explain is supported only with txt format
  expected_warn=$(normalize_whitespaces "WARN:   explain flag is supported only with txt output format; ignoring this flag for the required output format md")
  assert_output --partial "$expected_warn"
  # output consists of connectivity map without explanations in md format
  expected_connlist='| src | dst | conn |
|-----|-----|------|
| 0.0.0.0-255.255.255.255 | default/frontend[Deployment] | TCP 8080 |
| default/backend[Deployment] | default/frontend[Deployment] |  |
| default/frontend[Deployment] | 0.0.0.0-255.255.255.255 | UDP 53 |
| default/frontend[Deployment] | default/backend[Deployment] | TCP 9090 |'
normalized_expected_connlist=$(normalize_whitespaces "$expected_connlist")
assert_output --partial "$normalized_expected_connlist"
}

normalize_whitespaces() {
  echo "$1"| sed -e "s/[[:space:]]\+/ /g"
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
