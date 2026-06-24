#!/usr/bin/env bash

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
GITHUB_ROXIE_REPO="stackrox/roxie"
VERSION_FILE="${ROOT}/ROXIE_VERSION"

main() {
    local os; os=$(host_os)
    local arch; arch=$(host_arch)
    local install_path="${ROOT}/bin/${os}_${arch}/roxie"

    if ! ensure_roxie_installed "$os" "$arch" "$install_path"; then
        echo >&2 "Error: Failed to ensure roxie is installed"
        exit 1
    fi
    echo
    "$install_path" "$@"
}

host_os() {
    case "$(uname -s)" in
        Linux*)
            echo "linux"
            ;;
        Darwin*)
            echo "darwin"
            ;;
        *)
            echo >&2 "Error: Unsupported operating system"
            exit 1
            ;;
    esac
}

host_arch() {
    case "$(uname -m)" in
        x86_64)
            echo "amd64"
            ;;
        arm64)
            echo "arm64"
            ;;
        aarch64)
            echo "arm64"
            ;;
        *)
            echo >&2 "Error: Unsupported architecture"
            exit 1
            ;;
    esac

}

ensure_roxie_installed() {
    local os="$1"
    local arch="$2"
    local install_path="$3"

    local expected_version; expected_version=$(cat "$VERSION_FILE")
    if [[ -z "$expected_version" ]]; then
        echo >&2 "Error: No version found in $VERSION_FILE"
        return 1
    fi
    mkdir -p "$(dirname "$install_path")"

    if ! exists_with_correct_version "$install_path" "$expected_version"; then
        echo "File $install_path is missing or has an incorrect version."
        echo "Downloading the correct version..."
        local asset_name="roxie-${os}-${arch}"
        local tmp_roxie; tmp_roxie=$(mktemp)
        gh_download_release "$tmp_roxie" "$GITHUB_ROXIE_REPO" "v${expected_version}" "$asset_name"
        if [[ "$os" == "darwin" ]]; then
            xattr -d com.apple.quarantine "$tmp_roxie"
        fi
        mv "$tmp_roxie" "$install_path"
        chmod +x "$install_path"
        echo "roxie ${expected_version} has been installed to $install_path"
    fi
}

exists_with_correct_version() {
    local filepath="${1:-}"
    local expected_version="${2:-}"

    if [[ ! -e "$filepath" ]]; then
        echo "File '$filepath' does not exist"
        return 1
    fi

    local version; version="$("$filepath" version | awk '{ print $3 }')"
    if [[ "$version" != "v${expected_version}" ]]; then
        echo >&2 "Version mismatch for '$filepath'"
        echo >&2 "expected: v${expected_version}"
        echo >&2 "     got: $version"
        return 1
    fi

    return 0
}

gh_download_release() {
    local target=${1:-}
    local repo=${2:-}
    local version=${3:-}
    local asset_name=${4:-}
    local asset_id

    if [[ $version != 'latest' ]]; then
        version="tags/${version}"
    fi

    asset_id="$(curl -fsS --retry 3 --retry-delay 5 --retry-connrefused \
        -H "Accept: application/vnd.github+json" \
        -H "X-GitHub-Api-Version: 2026-03-10" \
        "https://api.github.com/repos/${repo}/releases/${version}" \
        | jq -r --arg asset_name "${asset_name}" '.assets[]|select(.name==$asset_name)|.id')"
    curl -fsSL --retry 3 --retry-delay 5 --retry-connrefused \
        -H 'Accept: application/octet-stream' \
        -o "$target" \
        "https://api.github.com/repos/${repo}/releases/assets/${asset_id}"
}

main "$@"
