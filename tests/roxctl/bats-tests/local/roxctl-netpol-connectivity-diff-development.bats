#!/usr/bin/env bats

load "../helpers.bash"
out_dir=""
templated_fragment='"{{ printf "%s" ._thing.image }}"'

setup_file() {
    delete-outdated-binaries "$(roxctl-development version)"
    echo "Testing roxctl version: '$(roxctl-development version)'" >&3
    export diff_tests_dir="${BATS_TEST_DIRNAME}/../../../../roxctl/netpol/connectivity/diff/testdata/"
}

setup() {
    out_dir="$(mktemp -d -u)"
    ofile="$(mktemp)"
}

teardown() {
    rm -rf "$out_dir"
    rm -f "$ofile"
}

@test "roxctl-development netpol connectivity diff illegal args" {
    run roxctl-development netpol connectivity diff "dir1" "dir2"
    assert_failure
    assert_line --partial "accepts 0 arg(s), received 2"
}

@test "roxctl-development netpol connectivity diff no input directories" {
    run roxctl-development netpol connectivity diff
    assert_failure
    assert_line --regexp "ERROR:.*--dir1 is required"
}

@test "roxctl-development netpol connectivity diff only one input directory" {
    run roxctl-development netpol connectivity diff --dir1="dir1"
    assert_failure
    assert_line --regexp "ERROR:.*--dir2 is required"
}

@test "roxctl-development netpol connectivity diff non existing dirs" {
    run roxctl-development netpol connectivity diff --dir1="$out_dir" --dir2="$out_dir"
    assert_failure
    assert_line --regexp "INFO:.*No connections diff"
    assert_line --regexp "ERROR:.*the path \"${out_dir}\" does not exist"
    assert_line --regexp "ERROR:.*the path \"${out_dir}\" does not exist"
    assert_line --regexp "ERROR:.*at dir1: no relevant Kubernetes workload resources found"
    assert_line --regexp "ERROR:.*at dir2: no relevant Kubernetes workload resources found"
    assert_line --regexp "ERROR:.*analyzing connectivity diff: there were errors during execution"
}

@test "roxctl-development netpol connectivity diff analyses only dir1 when run with --fail" {
    mkdir -p "$out_dir"
    write_yaml_to_file "$templated_fragment" "$(mktemp "$out_dir/templated-01-XXXXXX-file1.yaml")"
    write_yaml_to_file "$templated_fragment" "$(mktemp "$out_dir/templated-02-XXXXXX-file2.yaml")"

    run roxctl-development netpol connectivity diff --dir1="$out_dir/" --dir2="$out_dir/" --remove --output-file=/dev/null --fail
    assert_failure
    assert_line --regexp "INFO:.*No connections diff"

    assert_output --regexp 'WARN:.*unable to decode.*templated-01-.*-file1.yaml'
    assert_output --regexp 'WARN:.*unable to decode.*templated-02-.*-file2.yaml'
    assert_output --regexp 'WARN:.*error parsing.*templated-01-.*-file1.yaml'
    assert_output --regexp 'WARN:.*error parsing.*templated-02-.*-file2.yaml'

    assert_line --regexp "WARN:.*at dir1: no relevant Kubernetes network policy resources found"
    refute_output --regexp "WARN:.*at dir2: no relevant Kubernetes network policy resources found"

    assert_line --regexp "ERROR:.*at dir1: no relevant Kubernetes workload resources found"
    refute_output --regexp "ERROR:.*at dir2: no relevant Kubernetes workload resources found"
    assert_line --regexp "ERROR:.*analyzing connectivity diff: there were errors during execution"
}

@test "roxctl-development netpol connectivity diff analyses dir1 and dir2 when run without --fail" {
    mkdir -p "$out_dir"
    write_yaml_to_file "$templated_fragment" "$(mktemp "$out_dir/templated-01-XXXXXX-file1.yaml")"
    write_yaml_to_file "$templated_fragment" "$(mktemp "$out_dir/templated-02-XXXXXX-file2.yaml")"

    run roxctl-development netpol connectivity diff --dir1="$out_dir/" --dir2="$out_dir/" --remove --output-file=/dev/null
    assert_failure
    assert_line --regexp "INFO:.*No connections diff"

    assert_output --regexp 'WARN:.*unable to decode.*templated-01-.*-file1.yaml'
    assert_output --regexp 'WARN:.*unable to decode.*templated-02-.*-file2.yaml'
    assert_output --regexp 'WARN:.*error parsing.*templated-01-.*-file1.yaml'
    assert_output --regexp 'WARN:.*error parsing.*templated-02-.*-file2.yaml'

    assert_line --regexp "WARN:.*at dir1: no relevant Kubernetes network policy resources found"
    assert_line --regexp "WARN:.*at dir2: no relevant Kubernetes network policy resources found"

    assert_line --regexp "ERROR:.*at dir1: no relevant Kubernetes workload resources found"
    assert_line --regexp "ERROR:.*at dir2: no relevant Kubernetes workload resources found"
    assert_line --regexp "ERROR:.*analyzing connectivity diff: there were errors during execution"
}

@test "roxctl-development netpol connectivity diff produces no output when all yamls are templated" {
    mkdir -p "$out_dir"
    write_yaml_to_file "$templated_fragment" "$(mktemp "$out_dir/templated-XXXXXX.yaml")"

    echo "Analyzing a corrupted yaml file '$templatedYaml'" >&3
    run roxctl-development netpol connectivity diff --dir1="$out_dir/" --dir2="$out_dir/"
    assert_failure
    assert_line --regexp "INFO:.*No connections diff"

    assert_output --regexp 'WARN:.*unable to decode.*templated-.*.yaml'
    assert_output --regexp 'WARN:.*error parsing.*templated-.*.yaml'

    assert_line --regexp "WARN:.*at dir1: no relevant Kubernetes network policy resources found"
    assert_line --regexp "WARN:.*at dir2: no relevant Kubernetes network policy resources found"

    assert_line --regexp "ERROR:.*at dir1: no relevant Kubernetes workload resources found"
    assert_line --regexp "ERROR:.*at dir2: no relevant Kubernetes workload resources found"
    assert_line --regexp "ERROR:.*analyzing connectivity diff: there were errors during execution"
}

@test "roxctl-development netpol connectivity diff produces warnings when some yamls are templated" {
    mkdir -p "$out_dir"
    write_yaml_to_file "$templated_fragment" "$(mktemp "$out_dir/templated-XXXXXX.yaml")"

    assert_file_exist "${test_data}/np-guard/scenario-minimal-service/frontend.yaml"
    assert_file_exist "${test_data}/np-guard/scenario-minimal-service/backend.yaml"
    cp "${test_data}/np-guard/scenario-minimal-service/frontend.yaml" "$out_dir/frontend.yaml"
    cp "${test_data}/np-guard/scenario-minimal-service/backend.yaml" "$out_dir/backend.yaml"

    echo "Analyzing a directory where 1/3 of yaml files are templated '$out_dir/'" >&3
    run roxctl-development netpol connectivity diff --dir1="$out_dir/" --dir2="$out_dir/" --remove --output-file=/dev/null --fail
    assert_success
    assert_line --regexp "INFO:.*No connections diff"
    assert_output --regexp 'WARN:.*unable to decode.*templated-.*.yaml'
    assert_output --regexp 'WARN:.*error parsing.*templated-.*.yaml'
    assert_line --regexp "WARN:.*at dir1: no relevant Kubernetes network policy resources found"
    assert_line --regexp "WARN:.*at dir2: no relevant Kubernetes network policy resources found"
}

@test "roxctl-development netpol connectivity diff produces errors when all yamls are not K8s resources" {
    mkdir -p "$out_dir"
    assert_file_exist "${test_data}/np-guard/empty-yamls/empty.yaml"
    assert_file_exist "${test_data}/np-guard/empty-yamls/empty2.yaml"
    cp "${test_data}/np-guard/empty-yamls/empty.yaml" "$out_dir/empty.yaml"
    cp "${test_data}/np-guard/empty-yamls/empty2.yaml" "$out_dir/empty2.yaml"

    run roxctl-development netpol connectivity diff --dir1="$out_dir/" --dir2="$out_dir/" --remove --output-file=/dev/null
    assert_failure
    assert_line --regexp "INFO:.*No connections diff"
    assert_output --regexp 'WARN:.*unable to decode.*/empty.yaml'
    assert_output --regexp 'WARN:.*unable to decode.*/empty2.yaml'
    assert_line --regexp "WARN:.*at dir1: no relevant Kubernetes network policy resources found"
    assert_line --regexp "WARN:.*at dir2: no relevant Kubernetes network policy resources found"
    assert_line --regexp "ERROR:.*at dir1: no relevant Kubernetes workload resources found"
    assert_line --regexp "ERROR:.*at dir2: no relevant Kubernetes workload resources found"
    assert_line --regexp "ERROR:.*analyzing connectivity diff: there were errors during execution"
}

@test "roxctl-development netpol connectivity diff ignores without --strict invalid yamls and continues" {
    dir1="${diff_tests_dir}/acs-zeroday-with-invalid-doc/"
    assert_file_exist "${dir1}/deployment.yaml"
    assert_file_exist "${dir1}/namespace.yaml"
    assert_file_exist "${dir1}/route.yaml"
    # without strict it ignores the invalid yaml and continue
    run roxctl-development netpol connectivity diff --dir1="${dir1}" --dir2="${dir1}" --remove --output-file=/dev/null
    assert_success

    assert_line --regexp "INFO:.*No connections diff"
    assert_output --regexp 'WARN:.*unable to decode.*/deployment.yaml'
    assert_line --regexp "WARN:.*at dir1: no relevant Kubernetes network policy resources found"
    assert_line --regexp "WARN:.*at dir2: no relevant Kubernetes network policy resources found"
}

@test "roxctl-development netpol connectivity diff treats warning as error with --strict when some yamls are not valid" {
    dir1="${diff_tests_dir}/acs-zeroday-with-invalid-doc/"
    assert_file_exist "${dir1}/deployment.yaml"
    assert_file_exist "${dir1}/namespace.yaml"
    assert_file_exist "${dir1}/route.yaml"
    # running with strict, a warning on invalid yaml doc is treated as error
    run roxctl-development netpol connectivity diff --dir1="${dir1}" --dir2="${dir1}" --remove --output-file=/dev/null --strict
    assert_failure
    assert_line --regexp "INFO:.*No connections diff"
    assert_output --regexp 'WARN:.*unable to decode.*/deployment.yaml'
    assert_line --regexp "WARN:.*at dir1: no relevant Kubernetes network policy resources found"
    assert_line --regexp "WARN:.*at dir2: no relevant Kubernetes network policy resources found"
    assert_line --regexp "ERROR:.*analyzing connectivity diff: there were warnings during execution"
}

@test "roxctl-development netpol connectivity diff generates conns diff report between resources from two directories" {
    dir1="${diff_tests_dir}/acs-security-demos/"
    dir2="${diff_tests_dir}/acs-security-demos-new-version/"
    # assert files exist in dir1
    check_acs_security_demos_files ${dir1}
    # assert files exist in dir2
    check_acs_security_demos_new_version_files ${dir2}
    echo "Writing diff report to ${ofile}" >&3
    run roxctl-development netpol connectivity diff --dir1="${dir1}" --dir2="${dir2}"
    assert_success

    echo "$output" > "$ofile"
    assert_file_exist "$ofile"
    # partial is used to filter WARN and INFO messages
    assert_line --regexp "INFO:.*Found connections diffs"
    assert_output --partial 'Connectivity diff:
diff-type: added, source: payments/gateway[Deployment], destination: payments/visa-processor-v2[Deployment], dir1: No Connections, dir2: TCP 8080, workloads-diff-info: workload payments/visa-processor-v2[Deployment] added
diff-type: added, source: {ingress-controller}, destination: frontend/blog[Deployment], dir1: No Connections, dir2: TCP 8080, workloads-diff-info: workload frontend/blog[Deployment] added
diff-type: added, source: {ingress-controller}, destination: zeroday/zeroday[Deployment], dir1: No Connections, dir2: TCP 8080, workloads-diff-info: workload zeroday/zeroday[Deployment] added'
}

@test "roxctl-development netpol connectivity diff generates conns diff report between resources from two directories txt output" {
    dir1="${diff_tests_dir}/acs-security-demos/"
    dir2="${diff_tests_dir}/acs-security-demos-new-version/"
    # assert files exist in dir1
    check_acs_security_demos_files ${dir1}
    # assert files exist in dir2
    check_acs_security_demos_new_version_files ${dir2}
    echo "Writing diff report to ${ofile}" >&3
    run roxctl-development netpol connectivity diff --dir1="${dir1}" --dir2="${dir2}" --output-format=txt
    assert_success

    echo "$output" > "$ofile"
    assert_file_exist "$ofile"
    # partial is used to filter WARN and INFO messages
    assert_line --regexp "INFO:.*Found connections diffs"
    assert_output --partial 'Connectivity diff:
diff-type: added, source: payments/gateway[Deployment], destination: payments/visa-processor-v2[Deployment], dir1: No Connections, dir2: TCP 8080, workloads-diff-info: workload payments/visa-processor-v2[Deployment] added
diff-type: added, source: {ingress-controller}, destination: frontend/blog[Deployment], dir1: No Connections, dir2: TCP 8080, workloads-diff-info: workload frontend/blog[Deployment] added
diff-type: added, source: {ingress-controller}, destination: zeroday/zeroday[Deployment], dir1: No Connections, dir2: TCP 8080, workloads-diff-info: workload zeroday/zeroday[Deployment] added'
}

@test "roxctl-development netpol connectivity diff generates conns diff report between resources from two directories md output" {
    dir1="${diff_tests_dir}/acs-security-demos/"
    dir2="${diff_tests_dir}/acs-security-demos-new-version/"
    # assert files exist in dir1
    check_acs_security_demos_files ${dir1}
    # assert files exist in dir2
    check_acs_security_demos_new_version_files ${dir2}
    echo "Writing diff report to ${ofile}" >&3
    run roxctl-development netpol connectivity diff --dir1="${dir1}" --dir2="${dir2}" --output-format=md
    assert_success

    echo "$output" > "$ofile"
    assert_file_exist "$ofile"
    # partial is used to filter WARN and INFO messages
    assert_line --regexp "INFO:.*Found connections diffs"
    assert_output --partial '| diff-type | source | destination | dir1 | dir2 | workloads-diff-info |
|-----------|--------|-------------|------|------|---------------------|
| added | payments/gateway[Deployment] | payments/visa-processor-v2[Deployment] | No Connections | TCP 8080 | workload payments/visa-processor-v2[Deployment] added |
| added | {ingress-controller} | frontend/blog[Deployment] | No Connections | TCP 8080 | workload frontend/blog[Deployment] added |
| added | {ingress-controller} | zeroday/zeroday[Deployment] | No Connections | TCP 8080 | workload zeroday/zeroday[Deployment] added |'
}

@test "roxctl-development netpol connectivity diff generates conns diff report between resources from two directories dot output" {
    dir1="${diff_tests_dir}/acs-security-demos/"
    dir2="${diff_tests_dir}/acs-security-demos-new-version/"
    # assert files exist in dir1
    check_acs_security_demos_files ${dir1}
    # assert files exist in dir2
    check_acs_security_demos_new_version_files ${dir2}
    echo "Writing diff report to ${ofile}" >&3
    run roxctl-development netpol connectivity diff --dir1="${dir1}" --dir2="${dir2}" --output-format=dot
    assert_success

    echo "$output" > "$ofile"
    assert_file_exist "$ofile"
    # partial is used to filter WARN and INFO messages
    assert_line --regexp "INFO:.*Found connections diffs"
    assert_output --partial 'digraph {'
    assert_output --partial '"backend/checkout[Deployment]" -> "backend/notification[Deployment]" [label="TCP 8080" color="grey" fontcolor="grey" weight=0.5]'
}

@test "roxctl-development netpol connectivity diff generates conns diff report between resources from two directories csv output" {
    dir1="${diff_tests_dir}/acs-security-demos/"
    dir2="${diff_tests_dir}/acs-security-demos-new-version/"
    # assert files exist in dir1
    check_acs_security_demos_files ${dir1}
    # assert files exist in dir2
    check_acs_security_demos_new_version_files ${dir2}
    echo "Writing diff report to ${ofile}" >&3
    run roxctl-development netpol connectivity diff --dir1="${dir1}" --dir2="${dir2}" --output-format=csv
    assert_success

    echo "$output" > "$ofile"
    assert_file_exist "$ofile"
    assert_line --regexp "INFO:.*Found connections diffs"
    # partial is used to filter WARN and INFO messages
    assert_output --partial 'diff-type,source,destination,dir1,dir2,workloads-diff-info
added,payments/gateway[Deployment],payments/visa-processor-v2[Deployment],No Connections,TCP 8080,workload payments/visa-processor-v2[Deployment] added
added,{ingress-controller},frontend/blog[Deployment],No Connections,TCP 8080,workload frontend/blog[Deployment] added
added,{ingress-controller},zeroday/zeroday[Deployment],No Connections,TCP 8080,workload zeroday/zeroday[Deployment] added'
}

@test "roxctl-development netpol connectivity diff should return error on not supported output format" {
    dir1="${diff_tests_dir}/acs-security-demos/"
    dir2="${diff_tests_dir}/acs-security-demos-new-version/"
    # assert files exist in dir1
    check_acs_security_demos_files ${dir1}
    # assert files exist in dir2
    check_acs_security_demos_new_version_files ${dir2}
    echo "Writing diff report to ${ofile}" >&3
    run roxctl-development netpol connectivity diff --dir1="${dir1}" --dir2="${dir2}" --output-format=png
    assert_failure
    assert_line --regexp "INFO:.*Found connections diffs"
    assert_line --regexp "ERROR:.*formatting connectivity diff: png output format is not supported."
    assert_line --regexp "ERROR:.*analyzing connectivity diff: there were errors during execution"
}

@test "roxctl-development netpol connectivity diff generates conns diff report between resources from two directories to output file" {
    dir1="${diff_tests_dir}/acs-security-demos/"
    dir2="${diff_tests_dir}/acs-security-demos-new-version/"
    # assert files exist in dir1
    check_acs_security_demos_files ${dir1}
    # assert files exist in dir2
    check_acs_security_demos_new_version_files ${dir2}

    echo "Writing diff report to ${ofile}" >&3
    run roxctl-development netpol connectivity diff --dir1="${dir1}" --dir2="${dir2}" --output-file="$out_dir/out.txt"
    assert_success

    assert_file_exist "$out_dir/out.txt"
    # partial is used to filter WARN and INFO messages
    assert_line --regexp "INFO:.*Found connections diffs"
    assert_output --partial 'Connectivity diff:
diff-type: added, source: payments/gateway[Deployment], destination: payments/visa-processor-v2[Deployment], dir1: No Connections, dir2: TCP 8080, workloads-diff-info: workload payments/visa-processor-v2[Deployment] added
diff-type: added, source: {ingress-controller}, destination: frontend/blog[Deployment], dir1: No Connections, dir2: TCP 8080, workloads-diff-info: workload frontend/blog[Deployment] added
diff-type: added, source: {ingress-controller}, destination: zeroday/zeroday[Deployment], dir1: No Connections, dir2: TCP 8080, workloads-diff-info: workload zeroday/zeroday[Deployment] added'
}

@test "roxctl-development netpol connectivity diff generates conns diff report between resources from another two directories txt output" {
    dir1="${diff_tests_dir}/netpol-analysis-example-minimal/"
    dir2="${diff_tests_dir}/netpol-diff-example-minimal/"
    # assert files exist in dir1
    assert_file_exist "${dir1}/backend.yaml"
    assert_file_exist "${dir1}/frontend.yaml"
    assert_file_exist "${dir1}/netpols.yaml"
    # assert files exist in dir2
    assert_file_exist "${dir2}/backend.yaml"
    assert_file_exist "${dir2}/frontend.yaml"
    assert_file_exist "${dir2}/netpols.yaml"
    echo "Writing diff report to ${ofile}" >&3
    run roxctl-development netpol connectivity diff --dir1="${dir1}" --dir2="${dir2}" --output-format=txt
    assert_success

    echo "$output" > "$ofile"
    assert_file_exist "$ofile"
    # partial is used to filter WARN and INFO messages
    assert_line --regexp "INFO:.*Found connections diffs"
    assert_output --partial 'Connectivity diff:
diff-type: changed, source: default/frontend[Deployment], destination: default/backend[Deployment], dir1: TCP 9090, dir2: TCP 9090,UDP 53
diff-type: added, source: 0.0.0.0-255.255.255.255, destination: default/backend[Deployment], dir1: No Connections, dir2: TCP 9090'
}


@test "roxctl-development netpol connectivity diff generates conns diff report between resources from another two directories md output" {
    dir1="${diff_tests_dir}/netpol-analysis-example-minimal/"
    dir2="${diff_tests_dir}/netpol-diff-example-minimal/"
    # assert files exist in dir1
    assert_file_exist "${dir1}/backend.yaml"
    assert_file_exist "${dir1}/frontend.yaml"
    assert_file_exist "${dir1}/netpols.yaml"
    # assert files exist in dir2
    assert_file_exist "${dir2}/backend.yaml"
    assert_file_exist "${dir2}/frontend.yaml"
    assert_file_exist "${dir2}/netpols.yaml"
    echo "Writing diff report to ${ofile}" >&3
    run roxctl-development netpol connectivity diff --dir1="${dir1}" --dir2="${dir2}" --output-format=md
    assert_success

    echo "$output" > "$ofile"
    assert_file_exist "$ofile"
    assert_line --regexp "INFO:.*Found connections diffs"
  # partial is used to filter WARN and INFO messages
    assert_output --partial '| diff-type | source | destination | dir1 | dir2 | workloads-diff-info |
|-----------|--------|-------------|------|------|---------------------|
| changed | default/frontend[Deployment] | default/backend[Deployment] | TCP 9090 | TCP 9090,UDP 53 |  |
| added | 0.0.0.0-255.255.255.255 | default/backend[Deployment] | No Connections | TCP 9090 |  |'
}

@test "roxctl-development netpol connectivity diff generates conns diff report between resources from another two directories csv output" {
    dir1="${diff_tests_dir}/netpol-analysis-example-minimal/"
    dir2="${diff_tests_dir}/netpol-diff-example-minimal/"
    # assert files exist in dir1
    assert_file_exist "${dir1}/backend.yaml"
    assert_file_exist "${dir1}/frontend.yaml"
    assert_file_exist "${dir1}/netpols.yaml"
    # assert files exist in dir2
    assert_file_exist "${dir2}/backend.yaml"
    assert_file_exist "${dir2}/frontend.yaml"
    assert_file_exist "${dir2}/netpols.yaml"
    echo "Writing diff report to ${ofile}" >&3
    run roxctl-development netpol connectivity diff --dir1="${dir1}" --dir2="${dir2}" --output-format=csv
    assert_success

    echo "$output" > "$ofile"
    assert_file_exist "$ofile"
    # partial is used to filter WARN and INFO messages
    assert_line --regexp "INFO:.*Found connections diffs"
    assert_output --partial 'diff-type,source,destination,dir1,dir2,workloads-diff-info
changed,default/frontend[Deployment],default/backend[Deployment],TCP 9090,"TCP 9090,UDP 53",
added,0.0.0.0-255.255.255.255,default/backend[Deployment],No Connections,TCP 9090,'
}

@test "roxctl-development netpol connectivity diff empty diff report for two paths with same directory " {
    dir1="${diff_tests_dir}/netpol-analysis-example-minimal/"
    # assert files exist in dir1
    assert_file_exist "${dir1}/backend.yaml"
    assert_file_exist "${dir1}/frontend.yaml"
    assert_file_exist "${dir1}/netpols.yaml"
    echo "Writing diff report to ${ofile}" >&3
    run roxctl-development netpol connectivity diff --dir1="${dir1}" --dir2="${dir1}"
    assert_success

    echo "$output" > "$ofile"
    assert_file_exist "$ofile"
    # partial is used to filter WARN messages
    assert_line --regexp "INFO:.*No connections diff"
}

@test "roxctl-development netpol connectivity diff generates conns diff report between resources from directories with admin network policies txt output" {
    dir1="${test_data}/np-guard/anp_banp_demo"
    dir2="${test_data}/np-guard/anp_banp_diff_demo"
    # assert files exist in dir1
    assert_file_exist "${dir1}/ns.yaml"
    assert_file_exist "${dir1}/workloads.yaml"
    assert_file_exist "${dir1}/policies.yaml"
    # assert files exist in dir2
    assert_file_exist "${dir2}/ns.yaml"
    assert_file_exist "${dir2}/workloads.yaml"
    assert_file_exist "${dir2}/policies.yaml"
    echo "Writing diff report to ${ofile}" >&3
    run roxctl-development netpol connectivity diff --dir1="${dir1}" --dir2="${dir2}" --output-format=txt
    assert_success

    echo "$output" > "$ofile"
    assert_file_exist "$ofile"
    # partial is used to filter WARN and INFO messages
    assert_line --regexp "INFO:.*Found connections diffs"
    assert_output --partial 'Connectivity diff:
diff-type: added, source: baz/mybaz[Pod], destination: bar/mybar[Pod], dir1: No Connections, dir2: All Connections
diff-type: added, source: foo/myfoo[Pod], destination: bar/mybar[Pod], dir1: No Connections, dir2: All Connections
diff-type: removed, source: monitoring/mymonitoring[Pod], destination: foo/myfoo[Pod], dir1: All Connections, dir2: No Connections'
}

check_acs_security_demos_files() {
  dir="${1}"
  assert_file_exist "${dir}/backend/catalog/deployment.yaml"
  assert_file_exist "${dir}/backend/checkout/configmap.yaml"
  assert_file_exist "${dir}/backend/checkout/deployment.yaml"
  assert_file_exist "${dir}/backend/notification/deployment.yaml"
  assert_file_exist "${dir}/backend/recommendation/configmap.yaml"
  assert_file_exist "${dir}/backend/recommendation/deployment.yaml"
  assert_file_exist "${dir}/backend/reports/configmap.yaml"
  assert_file_exist "${dir}/backend/reports/deployment.yaml"
  assert_file_exist "${dir}/backend/shipping/deployment.yaml"
  assert_file_exist "${dir}/frontend/asset-cache/deployment.yaml"
  assert_file_exist "${dir}/frontend/asset-cache/route.yaml"
  assert_file_exist "${dir}/frontend/webapp/configmap.yaml"
  assert_file_exist "${dir}/frontend/webapp/deployment.yaml"
  assert_file_exist "${dir}/frontend/webapp/route.yaml"
  assert_file_exist "${dir}/payments/gateway/deployment.yaml"
  assert_file_exist "${dir}/payments/mastercard-processor/deployment.yaml"
  assert_file_exist "${dir}/payments/visa-processor/deployment.yaml"
  assert_file_exist "${dir}/acs_netpols.yaml"
}

check_acs_security_demos_new_version_files() {
  dir="${1}"
  assert_file_exist "${dir}/backend/catalog/deployment.yaml"
  assert_file_exist "${dir}/backend/checkout/configmap.yaml"
  assert_file_exist "${dir}/backend/checkout/deployment.yaml"
  assert_file_exist "${dir}/backend/notification/deployment.yaml"
  assert_file_exist "${dir}/backend/recommendation/configmap.yaml"
  assert_file_exist "${dir}/backend/recommendation/deployment.yaml"
  assert_file_exist "${dir}/backend/reports/configmap.yaml"
  assert_file_exist "${dir}/backend/reports/deployment.yaml"
  assert_file_exist "${dir}/backend/namespace.yaml"
  assert_file_exist "${dir}/backend/shipping/deployment.yaml"
  assert_file_exist "${dir}/frontend/asset-cache/deployment.yaml"
  assert_file_exist "${dir}/frontend/asset-cache/route.yaml"
  assert_file_exist "${dir}/frontend/blog/deployment.yaml"
  assert_file_exist "${dir}/frontend/blog/route.yaml"
  assert_file_exist "${dir}/frontend/namespace.yaml"
  assert_file_exist "${dir}/frontend/webapp/configmap.yaml"
  assert_file_exist "${dir}/frontend/webapp/deployment.yaml"
  assert_file_exist "${dir}/frontend/webapp/route.yaml"
  assert_file_exist "${dir}/payments/gateway/deployment.yaml"
  assert_file_exist "${dir}/payments/mastercard-processor/deployment.yaml"
  assert_file_exist "${dir}/payments/visa-processor/deployment.yaml"
  assert_file_exist "${dir}/payments/visa-processor-v2/deployment.yaml"
  assert_file_exist "${dir}/payments/namespace.yaml"
  assert_file_exist "${dir}/zeroday/deployment.yaml"
  assert_file_exist "${dir}/zeroday/namespace.yaml"
  assert_file_exist "${dir}/zeroday/route.yaml"
  assert_file_exist "${dir}/acs_netpols.yaml"
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
