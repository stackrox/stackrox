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

    if [[ -n "${ROXIE_PR:-}" ]]; then
        install_roxie_from_pr "$install_path"
        return 0
    fi

    if ! exists_with_correct_version "$install_path" "$expected_version"; then
        echo "File $install_path is missing or has an incorrect version."
        echo "Downloading the correct version..."
        local asset_name="roxie-${os}-${arch}"
        local tmp_roxie; tmp_roxie=$(mktemp)
        gh_download_release "$tmp_roxie" "$GITHUB_ROXIE_REPO" "v${expected_version}" "$asset_name"
        mv "$tmp_roxie" "$install_path"
        chmod +x "$install_path"
        echo "roxie ${expected_version} has been installed to $install_path"
    fi
}

# Build roxie from the head of the pull request in ROXIE_PR instead of
# installing the release pinned in ROXIE_VERSION. Lets CI jobs exercise a
# roxie PR before it is merged and released.
install_roxie_from_pr() {
    local install_path="$1"
    local stamp_path="${install_path}.pr-stamp"

    local head_sha
    if ! head_sha="$(curl -fsSL --retry 3 --retry-delay 5 --retry-connrefused \
        "https://api.github.com/repos/${GITHUB_ROXIE_REPO}/pulls/${ROXIE_PR}" \
        | jq -r '.head.sha')"; then
        echo >&2 "Error: Failed to look up PR #${ROXIE_PR} of ${GITHUB_ROXIE_REPO}"
        return 1
    fi
    if [[ -z "$head_sha" || "$head_sha" == "null" ]]; then
        echo >&2 "Error: No head SHA found for PR #${ROXIE_PR} of ${GITHUB_ROXIE_REPO}"
        return 1
    fi

    if [[ -x "$install_path" && "$(cat "$stamp_path" 2>/dev/null || true)" == "$head_sha" ]]; then
        echo "roxie from PR #${ROXIE_PR} (commit ${head_sha:0:12}) is already installed."
        return 0
    fi

    echo "Building roxie from PR #${ROXIE_PR} (commit ${head_sha:0:12})..."
    local tmpdir
    tmpdir="$(mktemp -d)"
    if ! git clone --quiet "https://github.com/${GITHUB_ROXIE_REPO}.git" "${tmpdir}/roxie"; then
        echo >&2 "Error: Failed to clone ${GITHUB_ROXIE_REPO}"
        rm -rf "$tmpdir"
        return 1
    fi
    if ! git -C "${tmpdir}/roxie" fetch --quiet --depth 1 origin "$head_sha" \
        || ! git -C "${tmpdir}/roxie" checkout --quiet --detach "$head_sha"; then
        echo >&2 "Error: Failed to check out ${head_sha} of ${GITHUB_ROXIE_REPO}"
        rm -rf "$tmpdir"
        return 1
    fi
    if ! (cd "${tmpdir}/roxie" && go build -o "$install_path" ./cmd); then
        echo >&2 "Error: Failed to build roxie from PR #${ROXIE_PR}"
        rm -rf "$tmpdir"
        return 1
    fi
    rm -rf "$tmpdir"
    echo "$head_sha" > "$stamp_path"
    echo "roxie from PR #${ROXIE_PR} built and installed to $install_path"
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
    local release_path="download/${version}"
    if [[ "$version" == latest ]]; then
        release_path="latest/download"
    fi

    local auth_headers=()
    if [[ -n "${GITHUB_TOKEN:-}" ]]; then
        auth_headers+=(-H "Authorization: token ${GITHUB_TOKEN}")
    fi

    # The caller runs in a conditional, so explicitly stop before it can install a failed download.
    if ! curl -fsSL --retry 3 --retry-delay 5 --retry-connrefused "${auth_headers[@]+"${auth_headers[@]}"}" \
        -o "$target" "https://github.com/${repo}/releases/${release_path}/${asset_name}"; then
        rm -f "$target"
        exit 1
    fi
}

main "$@"
