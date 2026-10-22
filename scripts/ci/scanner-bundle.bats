#!/usr/bin/env bats

load "../test_helpers.bats"

setup() {
    source "${BATS_TEST_DIRNAME}/scanner-bundle.sh"
    export BUNDLE_ROOT="$(cd "${BATS_TEST_DIRNAME}/../.." && pwd)"
    export BUNDLE_FILE="$BUNDLE_ROOT/scanner/image/scanner/bundles/ci-minimal/vulnerabilities.zip"
    export GITHUB_REF=refs/heads/master
    export GCLOUD_LOG="$BATS_TEST_TMPDIR/gcloud.log"
    export GITHUB_STEP_SUMMARY="$BATS_TEST_TMPDIR/summary"
    unset EXISTING_OBJECT STORAGE_FAILURE CORRUPT_OBJECT PUBLIC_CORRUPT HEADER_VALUE HTTP_STATUS
    gcloud() {
        echo "$*" >> "$GCLOUD_LOG"
        if [[ "$*" == 'storage objects list '* ]]; then
            [[ "${STORAGE_FAILURE:-false}" != true ]] || return 1
            printf '%s' "${EXISTING_OBJECT:-}"
        elif [[ "$3" == gs://* ]]; then
            if [[ "${CORRUPT_OBJECT:-false}" == true ]]; then
                printf 'corrupt' > "$4"
            else
                cp "$BUNDLE_FILE" "$4"
            fi
        else
            [[ "$5" == --if-generation-match=0 ]]
        fi
    }
    curl() {
        local output headers conditional=false
        while (($#)); do
            case "$1" in
                --output) output="$2"; shift 2 ;;
                --dump-header) headers="$2"; shift 2 ;;
                --header) conditional=true; shift 2 ;;
                *) shift ;;
            esac
        done
        if [[ "$conditional" == true ]]; then
            printf '%s' "${HTTP_STATUS:-304}"
        else
            printf 'HTTP/2 200\r\n%s\r\n\r\n' "${HEADER_VALUE-Last-Modified: Fri, 16 Oct 2026 00:00:00 GMT}" > "$headers"
            if [[ "${PUBLIC_CORRUPT:-false}" == true ]]; then
                printf 'corrupt' > "$output"
            else
                cp "$BUNDLE_FILE" "$output"
            fi
        fi
    }
    export -f gcloud curl
}

publish() {
    "$BUNDLE_ROOT/.github/workflows/scripts/scanner-publish-ci-bundle.sh"
}

@test "bundle version is SHA-256 of the checked-in bytes" {
    local digest
    digest="$(sha256sum "$BUNDLE_FILE")"
    run scanner_v4_ci_bundle_version
    assert_success
    assert_output "ci-minimal-${digest%% *}"
}

@test "missing bundle fails" {
    run scanner_v4_ci_bundle_version "$BATS_TEST_TMPDIR"
    assert_failure
}

@test "new bundle uses a create-only upload and verifies public content" {
    run publish
    assert_success
    run grep -- '--if-generation-match=0' "$GCLOUD_LOG"
    assert_success
    run cat "$GITHUB_STEP_SUMMARY"
    assert_output --partial 'https://definitions.stackrox.io/v4/vulnerability-bundles/ci-minimal-'
}

@test "existing identical bundle is verified without upload" {
    export EXISTING_OBJECT=existing
    run publish
    assert_success
    run grep -- '--if-generation-match=0' "$GCLOUD_LOG"
    assert_failure
}

@test "storage errors fail without attempting upload" {
    export STORAGE_FAILURE=true
    run publish
    assert_failure
    run grep -- '--if-generation-match=0' "$GCLOUD_LOG"
    assert_failure
}

@test "existing corrupt object fails without overwrite" {
    export EXISTING_OBJECT=existing CORRUPT_OBJECT=true
    run publish
    assert_failure
    run grep -- '--if-generation-match=0' "$GCLOUD_LOG"
    assert_failure
}

@test "public download must match committed bytes" {
    export PUBLIC_CORRUPT=true
    run publish
    assert_failure
}

@test "publication requires a valid Last-Modified header" {
    for header in '' 'Last-Modified: invalid'; do
        export HEADER_VALUE="$header"
        run publish
        assert_failure
    done
}

@test "publication requires conditional HTTP 304" {
    export HTTP_STATUS=200
    run publish
    assert_failure
    assert_output --partial 'Expected HTTP 304'
}

@test "branches other than master cannot publish" {
    export GITHUB_REF=refs/heads/feature
    run publish
    assert_failure
    [[ ! -e "$GCLOUD_LOG" ]]
}
