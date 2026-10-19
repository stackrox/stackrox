#!/usr/bin/env bash

# mock_executable creates an executable script at target_path with the given body.
mock_executable() {
    local target_path="$1"
    local script_body="$2"

    mkdir -p "$(dirname "$target_path")"
    printf '#!/usr/bin/env bash\n%s\n' "$script_body" > "$target_path"
    chmod +x "$target_path"
}

# given_binary_version makes `postgres -V` report the given major version.
given_binary_version() {
    local dir="$1" major="$2"
    mock_executable "${dir}/postgres" "echo 'postgres (PostgreSQL) ${major}.2'"
}

# given_data_version writes PG_VERSION in the given directory.
given_data_version() {
    local dir="$1" version="$2"
    echo "$version" > "${dir}/PG_VERSION"
}

# given_prev_binaries mimics the postgres executable under a version-specific directory.
given_prev_binaries() {
    local dir="$1" major="$2"
    mock_executable "${dir}/postgresql-${major}/bin/postgres" "echo 'PostgreSQL ${major} server'"
}

# given_available_space mocks du and df to simulate disk space conditions.
given_available_space() {
    local bin_dir="$1" data_size_kb="$2" available_kb="$3"
    mock_executable "${bin_dir}/du" "echo '${data_size_kb}  \$1'"
    mock_executable "${bin_dir}/df" "echo 'Filesystem    1024-blocks       Used  Available'; echo '/dev/sda1 2000000 1900000 ${available_kb}'"
}
