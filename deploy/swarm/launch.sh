#!/usr/bin/env bash

function launch_central {
    local swarm_dir="$1"
    local main_image="$2"
    local local_api_endpoint="$3"
    local rox_disable_registry_auth="$4"

    echo "Generating central config..."
    OLD_DOCKER_HOST="$DOCKER_HOST"
    OLD_DOCKER_CERT_PATH="$DOCKER_CERT_PATH"
    OLD_DOCKER_TLS_VERIFY="$DOCKER_TLS_VERIFY"
    unset DOCKER_HOST DOCKER_CERT_PATH DOCKER_TLS_VERIFY

    set -u

    docker run --rm -e ROX_HTPASSWD_AUTH "$main_image" deploy swarm -i "$main_image" -p 8000 none > "$swarm_dir/central.zip"


    export DOCKER_HOST="$OLD_DOCKER_HOST"
    export DOCKER_CERT_PATH="$OLD_DOCKER_CERT_PATH"
    export DOCKER_TLS_VERIFY="$OLD_DOCKER_TLS_VERIFY"

    local unzip_dir="$swarm_dir/central-deploy/"
    rm -rf "$unzip_dir"
    unzip "$swarm_dir/central.zip" -d "$unzip_dir"
    echo

    echo "Deploying Central..."
    if [ "$rox_disable_registry_auth" = "true" ]; then
        cp "$unzip_dir/central.sh" "$unzip_dir/tmp"
        cat "$unzip_dir/tmp" | sed "s/--with-registry-auth//" > "$unzip_dir/central.sh"
        rm "$unzip_dir/tmp"
    fi

	if [[ -f "${unzip_dir}/password" ]]; then
		export ROX_ADMIN_USER=admin
		export ROX_ADMIN_PASSWORD="$(< "${unzip_dir}/password")"
	fi
    $unzip_dir/central.sh
    echo
    wait_for_central "localhost:8000"
    echo "Successfully launched central"
    echo "Access the UI at: https://localhost:8000"
    setup_auth0 "localhost:8000"
}

function launch_sensor {
    local swarm_dir="$1"
    local main_image="$2"
    local cluster="$3"
    local cluster_api_endpoint="$4"
    local rox_disable_registry_auth="$5"

    local extra_config=""
    if [ "$DOCKER_CERT_PATH" = "" ]; then
        extra_config="\"swarm\": { \"disableSwarmTls\":true } }"
    fi
    # false is for runtime support
    get_cluster_zip localhost:8000 "$cluster" SWARM_CLUSTER "$main_image" "$cluster_api_endpoint" "$swarm_dir" false "$extra_config"

    echo "Deploying Sensor..."
    local unzip_dir="$swarm_dir/sensor-deploy/"
    rm -rf "$unzip_dir"
    unzip "$swarm_dir/sensor-deploy.zip" -d "$unzip_dir"

    if [ "$rox_disable_registry_auth" = "true" ]; then
        cp "$unzip_dir/sensor.sh" "$unzip_dir/tmp"
        cat "$unzip_dir/tmp" | sed "s/--with-registry-auth//" > "$unzip_dir/sensor.sh"
        rm "$unzip_dir/tmp"
    fi

    $unzip_dir/sensor.sh
    echo

    echo "Successfully deployed!"
}
