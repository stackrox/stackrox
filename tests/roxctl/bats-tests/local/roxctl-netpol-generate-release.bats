#!/usr/bin/env bats

load "../helpers.bash"

out_dir=""
templated_fragment='"{{ printf "%s" ._thing.image }}"'

setup_file() {
    command -v yq >/dev/null || skip "Tests in this file require yq"
    echo "Using yq version: '$(yq --version)'" >&3
    # as of Aug 2022, we run yq version 4.16.2
    # remove binaries from the previous runs
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

@test "roxctl-release netpol generate should not show deprecation info" {
    run roxctl-release netpol generate
    refute_line --partial "is deprecated"
}

@test "roxctl-release netpol generate should return error on empty or non-existing directory" {
    run roxctl-release netpol generate "$out_dir"
    assert_failure
    assert_output --regexp 'ERROR:.*the path.*does not exist'
    assert_output --regexp 'ERROR:.*error generating network policies: could not find any Kubernetes workload resources'
    assert_output --regexp 'ERROR:.*generating netpols: there were errors during execution'

    run roxctl-release netpol generate
    assert_failure
    assert_line --partial "accepts 1 arg(s), received 0"
}

@test "roxctl-release netpol generate generates network policies" {
    assert_file_exist "${test_data}/np-guard/scenario-minimal-service/frontend.yaml"
    assert_file_exist "${test_data}/np-guard/scenario-minimal-service/backend.yaml"
    echo "Writing network policies to ${ofile}" >&3
    run roxctl-release netpol generate "${test_data}/np-guard/scenario-minimal-service"
    assert_success

    echo "$output" > "$ofile"
    assert_file_exist "$ofile"
    yaml_valid "$ofile"

    # There must be at least 3 yaml documents in the output
    # yq version 4.16.2 has problems with handling 'document_index', thus we use 'di'
    run yq e 'di' "${ofile}"
    assert_line '0'
    assert_line '1'
    assert_line '2'

    # Ensure that all yaml docs are of kind 'NetworkPolicy'
    run yq e '.kind | ({"match": ., "doc": di})' "${ofile}"
    # Github actions run yq v3
    assert_line --index 0 'match: NetworkPolicy'
    assert_line --index 1 'doc: 0'
    assert_line --index 2 'match: NetworkPolicy'
    assert_line --index 3 'doc: 1'
    assert_line --index 4 'match: NetworkPolicy'
    assert_line --index 5 'doc: 2'

    # yq v4 assertions
    #    assert_line --index 0 'match: NetworkPolicy'
    #    assert_line --index 1 'doc: 0'
    #    assert_line --index 2 '---'
    #    assert_line --index 3 'match: NetworkPolicy'
    #    assert_line --index 4 'doc: 1'
    #    assert_line --index 5 '---'
    #    assert_line --index 6 'match: NetworkPolicy'
    #    assert_line --index 7 'doc: 2'

    # Ensure that all NetworkPolicies have the generated-by-stackrox label
    run yq e '.metadata.labels | ({"match": ."network-policy-buildtime-generator.stackrox.io/generated", "doc": di})' "${ofile}"
    assert_line --index 0 'match: "true"'
    assert_line --index 1 'doc: 0'
    assert_line --index 2 'match: "true"'
    assert_line --index 3 'doc: 1'
    assert_line --index 4 'match: "true"'
    assert_line --index 5 'doc: 2'

    # yq v4 assertions
    #    assert_line --index 0 'match: "true"'
    #    assert_line --index 1 'doc: 0'
    #    assert_line --index 2 '---'
    #    assert_line --index 3 'match: "true"'
    #    assert_line --index 4 'doc: 1'
    #    assert_line --index 5 '---'
    #    assert_line --index 6 'match: "true"'
    #    assert_line --index 7 'doc: 2'
}

@test "roxctl-release netpol generate generates network policies with custom dns port" {
    assert_file_exist "${test_data}/np-guard/scenario-minimal-service/frontend.yaml"
    assert_file_exist "${test_data}/np-guard/scenario-minimal-service/backend.yaml"
    echo "Writing network policies to ${ofile}" >&3
    dns_port="5353"
    run roxctl-release netpol generate "${test_data}/np-guard/scenario-minimal-service" --dnsport ${dns_port}
    assert_success

    echo "$output" > "$ofile"
    assert_file_exist "$ofile"
    yaml_valid "$ofile"

    # There must be at least 3 yaml documents in the output
    # yq version 4.16.2 has problems with handling 'document_index', thus we use 'di'
    run yq e 'di' "${ofile}"
    assert_line '0'
    assert_line '1'
    assert_line '2'

    # Ensure that all yaml docs are of kind 'NetworkPolicy'
    run yq e '.kind | ({"match": ., "doc": di})' "${ofile}"
    assert_line --index 0 'match: NetworkPolicy'
    assert_line --index 1 'doc: 0'
    assert_line --index 2 'match: NetworkPolicy'
    assert_line --index 3 'doc: 1'
    assert_line --index 4 'match: NetworkPolicy'
    assert_line --index 5 'doc: 2'

    # Ensure that dns ports are properly set
    run yq e '.spec.egress[1].ports[0].port | ({"match": ., "doc": di})' "${ofile}"
    assert_line --index 0 'match: null'
    assert_line --index 1 'doc: 0'
    assert_line --index 2 'match: '${dns_port}
    assert_line --index 3 'doc: 1'
    assert_line --index 4 'match: null'
    assert_line --index 5 'doc: 2'
}

@test "roxctl-release netpol generate generates network policies with custom dns named port" {
    assert_file_exist "${test_data}/np-guard/scenario-minimal-service/frontend.yaml"
    assert_file_exist "${test_data}/np-guard/scenario-minimal-service/backend.yaml"
    echo "Writing network policies to ${ofile}" >&3
    dns_port="dns"
    run roxctl-release netpol generate "${test_data}/np-guard/scenario-minimal-service" --dnsport ${dns_port}
    assert_success

    echo "$output" > "$ofile"
    assert_file_exist "$ofile"
    yaml_valid "$ofile"

    # Ensure that dns ports are properly set
    run yq e '.spec.egress[1].ports[0].port | ({"match": ., "doc": di})' "${ofile}"
    assert_line --index 0 'match: null'
    assert_line --index 1 'doc: 0'
    assert_line --index 2 'match: '${dns_port}
    assert_line --index 3 'doc: 1'
    assert_line --index 4 'match: null'
    assert_line --index 5 'doc: 2'
}

@test "roxctl-release netpol generate fails with dns port set to 0" {
    assert_file_exist "${test_data}/np-guard/scenario-minimal-service/frontend.yaml"
    assert_file_exist "${test_data}/np-guard/scenario-minimal-service/backend.yaml"
    run roxctl-release netpol generate "${test_data}/np-guard/scenario-minimal-service" --dnsport 0
    assert_failure
    assert_output --regexp 'ERROR:.*illegal port number'
}

@test "roxctl-release netpol generate fails with dns port set to empty string" {
    assert_file_exist "${test_data}/np-guard/scenario-minimal-service/frontend.yaml"
    assert_file_exist "${test_data}/np-guard/scenario-minimal-service/backend.yaml"
    run roxctl-release netpol generate "${test_data}/np-guard/scenario-minimal-service" --dnsport ""
    assert_failure
    assert_output --regexp 'ERROR:.*illegal port name'
}

@test "roxctl-release netpol generate produces no output when all yamls are templated" {
    mkdir -p "$out_dir"
    write_yaml_to_file "$templated_fragment" "$(mktemp "$out_dir/templated-XXXXXX.yaml")"

    echo "Analyzing a corrupted yaml file '$templatedYaml'" >&3
    run roxctl-release netpol generate "$out_dir/"
    assert_failure
    assert_output --regexp 'WARN:.*error parsing .*templated-.*.yaml'
    assert_output --regexp 'ERROR:.*error generating network policies: could not find any Kubernetes workload resources'
    assert_output --regexp 'ERROR:.*generating netpols: there were errors during execution'
}

@test "roxctl-release netpol generate produces warnings when some yamls are templated" {
    mkdir -p "$out_dir"
    write_yaml_to_file "$templated_fragment" "$(mktemp "$out_dir/templated-XXXXXX.yaml")"

    assert_file_exist "${test_data}/np-guard/scenario-minimal-service/frontend.yaml"
    assert_file_exist "${test_data}/np-guard/scenario-minimal-service/backend.yaml"
    cp "${test_data}/np-guard/scenario-minimal-service/frontend.yaml" "$out_dir/frontend.yaml"
    cp "${test_data}/np-guard/scenario-minimal-service/backend.yaml" "$out_dir/backend.yaml"

    echo "Analyzing a directory where 1/3 of yaml files are templated '$out_dir/'" >&3
    run roxctl-release netpol generate "$out_dir/" --remove --output-file=/dev/null
    assert_success
    assert_output --regexp 'WARN:.*error parsing .*templated-.*.yaml'
}

@test "roxctl-release netpol generate parameter --strict" {
    mkdir -p "$out_dir"
    assert_file_exist "${test_data}/np-guard/empty-yamls/empty.yaml"
    assert_file_exist "${test_data}/np-guard/empty-yamls/empty2.yaml"
    cp "${test_data}/np-guard/empty-yamls/empty.yaml" "$out_dir/empty.yaml"
    cp "${test_data}/np-guard/empty-yamls/empty2.yaml" "$out_dir/empty2.yaml"

    run roxctl-release netpol generate "$out_dir/" --remove --output-file=/dev/null
    assert_failure
    assert_output --regexp 'WARN:.*unable to decode.*empty.yaml'
    assert_output --regexp 'WARN:.*unable to decode.*empty2.yaml'
    assert_output --regexp 'ERROR:.*error generating network policies: could not find any Kubernetes workload resources'
    assert_output --regexp 'ERROR:.*generating netpols: there were errors during execution'

    run roxctl-release netpol generate "$out_dir/" --remove --output-file=/dev/null --strict
    assert_failure
    assert_output --regexp 'WARN:.*unable to decode.*empty.yaml'
    assert_output --regexp 'WARN:.*unable to decode.*empty2.yaml'
    assert_output --regexp 'ERROR:.*error generating network policies: could not find any Kubernetes workload resources'
    assert_output --regexp 'ERROR:.*generating netpols: there were errors during execution'
}

@test "roxctl-release netpol generate parameter --fail" {
    mkdir -p "$out_dir"
    write_yaml_to_file "$templated_fragment" "$(mktemp "$out_dir/templated-01-XXXXXX-file1.yaml")"
    write_yaml_to_file "$templated_fragment" "$(mktemp "$out_dir/templated-02-XXXXXX-file2.yaml")"

    run roxctl-release netpol generate "$out_dir/" --remove --output-file=/dev/null --fail
    assert_failure
    assert_output --regexp 'WARN:.*error parsing.*-file1.yaml'
    assert_output --regexp 'WARN:.*error parsing.*-file2.yaml'
    assert_output --regexp 'ERROR:.*error generating network policies: could not find any Kubernetes workload resources'
    assert_output --regexp 'ERROR:.*generating netpols: there were errors during execution'
}

@test "roxctl-release netpol generate parameter --fail and --strict" {
    mkdir -p "$out_dir"
    assert_file_exist "${test_data}/np-guard/scenario-minimal-service/frontend.yaml"
    assert_file_exist "${test_data}/np-guard/scenario-minimal-service/backend.yaml"
    cp "${test_data}/np-guard/scenario-minimal-service/frontend.yaml" "$out_dir/frontend.yaml"
    cp "${test_data}/np-guard/scenario-minimal-service/backend.yaml" "$out_dir/backend.yaml"
    write_yaml_to_file "$templated_fragment" "$(mktemp "$out_dir/templated-01-XXXXXX-file1.yaml")"
    write_yaml_to_file "$templated_fragment" "$(mktemp "$out_dir/templated-02-XXXXXX-file2.yaml")"

    run roxctl-release netpol generate "$out_dir/" --fail --strict
    assert_failure
    assert_output --regexp 'ERROR:.*generating netpols: there were warnings during execution'
    assert_output --regexp 'WARN:.*error parsing.*-file1.yaml'
    # should fail fast before trying to decode file2 due to warnings when processing file1
    refute_output --regexp 'WARN:.*error parsing.*-file2.yaml'
}

write_yaml_to_file() {
  image="${1}"
  templatedYaml="${2:-/dev/null}"
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
