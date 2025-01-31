{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  name = "operator-bundle-helpers";

  # Specify packages to include in the environment
  buildInputs = [
    pkgs.python39
    pkgs.python39Packages.pip-tools
    pkgs.curl
  ];

  shellHook = ''
    recreate_script="recreate.sh"

    cat >"$recreate_script" <<EOF
    #!/usr/bin/env bash
    set -xe
    echo "Downloading pip_find_builddeps.py..."
    curl -fO https://raw.githubusercontent.com/containerbuildsystem/cachito/master/bin/pip_find_builddeps.py
    chmod +x pip_find_builddeps.py
    pip-compile requirements.in --generate-hashes
    ./pip_find_builddeps.py requirements.txt --append --only-write-on-update -o requirements-build.in
    pip-compile requirements-build.in --allow-unsafe --generate-hashes
    EOF

    chmod 755 "$recreate_script"
    exit_code=0
    "./$recreate_script" || exit_code=1
    rm -f "$recreate_script"

    if [[ "$exit_code" -eq 1 ]]; then
        echo >&2 "Failed to recreate requirements."
    else
        echo "Done."
    fi

    exit "$exit_code"
  '';
}
