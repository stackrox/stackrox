#!/usr/bin/env bats
# shellcheck disable=SC1091

# Tests for `make tag` version resolution, which falls back to the VERSION
# file when git describe is unavailable (e.g. shallow clones).

load "../../test_helpers.bats"

@test "make tag produces a non-empty version string" {
    run make --quiet --no-print-directory tag
    assert_success
    assert [ -n "$output" ]
}

@test "make tag output starts with a version number" {
    run make --quiet --no-print-directory tag
    assert_success
    [[ "$output" =~ ^[0-9]+\.[0-9]+\. ]]
}

@test "VERSION file exists" {
    assert [ -f VERSION ]
}

@test "VERSION file contains a valid version tag" {
    run cat VERSION
    assert_success
    # Should match patterns like 4.11.x or 4.12.0
    [[ "$output" =~ ^[0-9]+\.[0-9]+\.[a-z0-9]+$ ]]
}

@test "make tag output starts with VERSION file content" {
    version="$(tr -d '[:space:]' < VERSION)"
    run make --quiet --no-print-directory tag
    assert_success
    [[ "$output" == "$version"* ]]
}

@test "VERSION file matches nearest git tag" {
    # Fetch full history so git describe can find the nearest tag naturally,
    # then verify it matches the VERSION file. This catches drift between
    # the VERSION file and actual git tags (e.g. if start-release.yml creates
    # a tag but fails to update VERSION, or vice versa).
    run git fetch --unshallow origin HEAD
    run git fetch --tags origin
    assert_success "failed to fetch tags — cannot validate VERSION against git history"

    run git describe --tags --abbrev=0 --exclude '*-nightly-*'
    assert_success "git describe failed after fetching full history — no tags reachable from HEAD"

    version="$(tr -d '[:space:]' < VERSION)"
    assert_output "$version"
}
