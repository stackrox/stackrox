#!/usr/bin/env bash
set -eou pipefail

json_config_file=$1

DIR="$(cd "$(dirname "$0")" && pwd)"

echo "json_config_file= $json_config_file"

test_dir="$(jq .test_dir "$json_config_file" --raw-output)"
if [ "$test_dir" == "null" ]; then
    echo "test_dir must be defined"
    exit 1
fi
echo "Set test_dir to $test_dir"

load="$(jq .load "$json_config_file")"
if [ "$load" == "null" ]; then
    echo "load must be defined"
    exit 1
fi
load_duration="$(echo $load | jq .load_duration --raw-output)"
echo "Set load_duration to $load_duration"
kubenetbench_load="$(echo $load | jq .kubenetbench_load)"
if [ "$kubenetbench_load" != null ]; then
    num_streams="$(echo "$kubenetbench_load" | jq .num_streams --raw-output)"
    load_test_name="$(echo "$kubenetbench_load" | jq .load_test_name --raw-output)"

    if [ "$num_streams" == "null" ]; then
        echo "If kubenetbench_load is defined num_streams must be defined"
	exit 1
    fi
    echo "Set num_streams to $num_streams"

    if [ "$load_test_name" == "null" ]; then
        echo "If kubenetbench_load is defined load_test_name must be defined"
	exit 1
    fi
    echo "Set load_test_name to $load_test_name"
fi
open_close_ports_load="$(echo $load | jq .open_close_ports_load)"
if [ "$open_close_ports_load" != null ]; then
    num_ports="$(echo "$open_close_ports_load" | jq .num_ports --raw-output)"
    num_per_second="$(echo "$open_close_ports_load" | jq .num_per_second --raw-output)"
    num_pods="$(echo "$open_close_ports_load" | jq .num_pods --raw-output)"
    num_concurrent="$(echo "$open_close_ports_load" | jq .num_concurrent --raw-output)"

    if [ "$num_ports" == "null" ]; then
        echo "If open_close_ports_load is defined num_ports must be defined"
        exit 1
    fi
    if [ "$num_per_second" == "null" ]; then
        echo "If open_close_ports_load is defined num_per_second must be defined"
        exit 1
    fi
    if [ "$num_pods" == "null" ]; then
        echo "If open_close_ports_load is defined num_pods must be defined"
        exit 1
    fi
    if [ "$num_concurrent" == "null" ]; then
        echo "If open_close_ports_load is defined num_concurrent must be defined"
        exit 1
    fi
fi
kube_burner_load="$(echo $load | jq .kube_burner_load)"
if [ "$kube_burner_load" != null ]; then
    kube_burner_config="$(echo "$kube_burner_load" | jq .config --raw-output)"
    kube_burner_path="$(echo "$kube_burner_load" | jq .path --raw-output)"
    kube_burner_uuid="$(echo "$kube_burner_load" | jq .uuid --raw-output)"

    if [ "$kube_burner_config" == "null" ]; then
        echo "If kube_burner_load is defined kube_burner_config must be defined"
	exit 1
    fi
    if [ "$kube_burner_path" == "null" ]; then
        echo "If kube_burner_load is defined kube_burner_path must be defined"
	exit 1
    fi
    if [ "$kube_burner_uuid" == "null" ]; then
        echo "If kube_burner_load is defined kube_burner_uuid must be defined"
	exit 1
    fi
fi

sleep_after_start_rox="$(jq .sleep_after_start_rox "$json_config_file" --raw-output)"
if [ "$sleep_after_start_rox" == "null" ]; then
    echo "sleep_after_start_rox must be defined"
    exit 1
fi
echo "Set sleep_after_start_rox to $sleep_after_start_rox"

query_window="$(jq .query_window "$json_config_file" --raw-output)"
if [ "$query_window" == "null" ]; then
    echo "query_window must be defined"
    exit 1
fi
echo "Set query_window to $query_window"

versions="$(jq .versions "$json_config_file")"
if [ "$versions" == "null" ]; then
    echo "versions must be defined"
    exit 1
fi

teardown_script="$(jq .teardown_script "$json_config_file" --raw-output)"
if [ "$teardown_script" == "null" ]; then
    echo "teardown_script must be defined"
    exit 1
fi
echo "Set teardown_script to $teardown_script"

cluster_name="$(jq .cluster_name "$json_config_file" --raw-output)"
if [ "$cluster_name" == "null" ]; then
    day=$(date +"%Y-%m-%d")
    cluster_name="perf-testing-${day}-$RANDOM"
fi
echo "Set cluster name to $cluster_name"

nrepeat="$(jq .nrepeat "$json_config_file" --raw-output)"
if [ "$nrepeat" == "null" ]; then
    nrepeat=5
fi
echo "Set nrepeat to $nrepeat"

num_worker_nodes="$(jq .num_worker_nodes "$json_config_file" --raw-output)"
if [ "$num_worker_nodes" == "null" ]; then
    num_worker_nodes=3
fi
echo "Set num_worker_nodes to $num_worker_nodes"

artifacts_dir="$(jq .artifacts_dir "$json_config_file" --raw-output)"
if [ "$artifacts_dir" == "null" ]; then
    artifacts_dir="/tmp/artifacts-${cluster_name}"
fi

nversion="$(jq '.versions | length' "$json_config_file")"

"$DIR"/create-infra.sh "$cluster_name" openshift-4-perf-scale 48h "$num_worker_nodes"
"$DIR"/wait-for-cluster.sh "$cluster_name"
infractl artifacts "$cluster_name" --download-dir "$artifacts_dir"
export KUBECONFIG="$artifacts_dir"/kubeconfig

"$DIR"/set-docker-credentials-for-cluster.sh

if [[ $kubenetbench_load != "null" && $num_streams -gt 0 ]]; then
    knb_base_dir="$(mktemp -d)"
    "$DIR/kubenetbench/teardown-kubenetbench.sh" "$artifacts_dir" "$knb_base_dir"
    "$DIR/kubenetbench/initialize-kubenetbench.sh" "$artifacts_dir" "$load_test_name" "$knb_base_dir"
fi

mkdir -p "$test_dir"

for ((n = 0; n < nrepeat; n = n + 1)); do
    for ((i = 0; i < nversion; i = i + 1)); do
	version="$(echo $versions | jq .["$i"])"
        echo "$version"
	collector_image_registry="$(echo $version | jq .collector_image_registry --raw-output)"
	collector_image_tag="$(echo $version | jq .collector_image_tag --raw-output)"
	nick_name="$(echo $version | jq .nick_name --raw-output)"
	patch_script="$(echo $version | jq .patch_script --raw-output)"
	env_var_file="$(echo $version | jq .env_var_file --raw-output)"
	if [[ $open_close_ports_load != "null" && $num_ports -gt 0 ]]; then
            "$DIR"/open-close-ports-load/start-open-close-ports-load.sh "$artifacts_dir" "$num_ports" "$num_per_second" "$num_concurrent" "$num_pods"
        fi
        source "${env_var_file}"
        printf 'yes\n'  | $teardown_script
        "$DIR"/start-acs-test-stack.sh "$cluster_name" "$artifacts_dir" "$collector_image_registry" "$collector_image_tag"
	if [[ "$patch_script" != null ]]; then
            "$patch_script" "$artifacts_dir"
	fi
	"$DIR"/wait-for-pods.sh "$artifacts_dir"
        sleep "$sleep_after_start_rox"
        if [[ $kubenetbench_load != "null" && $num_streams -gt 0 ]]; then
            "$DIR/kubenetbench/generate-network-load.sh" "$artifacts_dir" "$load_test_name" "$num_streams" "$knb_base_dir" "$load_duration"
	elif [[ $kube_burner_load != "null" ]]; then
            "$kube_burner_path" destroy --uuid "$kube_burner_uuid"
            "$DIR"/kube-burner/start-kube-burner.sh "$kube_burner_path" "$kube_burner_config" "$kube_burner_uuid" "$artifacts_dir" "$load_duration" &> "$test_dir/kube-burner-log-${nick_name}_${n}.txt"
	else
            sleep "$load_duration"
	fi

        query_output="$test_dir/results_${nick_name}_${n}.json"
        "$DIR"/query.sh "$query_output" "$artifacts_dir" "$query_window"
	if [[ $open_close_ports_load != "null" && $num_ports -gt 0 ]]; then
            "$DIR"/open-close-ports-load/stop-open-close-ports-load.sh "$artifacts_dir"
        fi

	if [[ "$kube_burner_load" != "null" ]]; then
	    echo "Tearing down kube-burner"
	    "$kube_burner_path" destroy --uuid "$kube_burner_uuid"
	fi

    done
done

printf 'yes\n'  | $teardown_script

if [[ $kubenetbench_load != "null" && $num_streams -gt 0 ]]; then
    "$DIR/kubenetbench/teardown-kubenetbench.sh" "$artifacts_dir" "$knb_base_dir"
fi

for ((i = 0; i < nversion; i = i + 1)); do
    version="$(echo $versions | jq .["$i"])"
    nick_name="$(echo $version | jq .nick_name --raw-output)"
    python3 "$DIR"/get-averages.py "${test_dir}/results_${nick_name}_" "$nrepeat" "${test_dir}/Average_results_${nick_name}.json"
done
