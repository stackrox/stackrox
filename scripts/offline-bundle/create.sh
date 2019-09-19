#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

main() {
    # Main uses the version reported by make tag.
    local main_tag="$(make tag)"
    local main_image="stackrox.io/main:${main_tag}"

    # Monitoring uses the same exact version as main.
    local monitoring_tag="$main_tag"
    local monitoring_image="stackrox.io/monitoring:${monitoring_tag}"

    # Scanner uses the version contained in the SCANNER_VERSION file.
    local scanner_tag="$(cat SCANNER_VERSION)"
    local scanner_image="stackrox.io/scanner:${scanner_tag}"

    # Scanner v2 uses the version contained in the SCANNER_V2_VERSION file.
    local scanner_v2_tag="$(cat SCANNER_V2_VERSION)"
    local scanner_v2_image="stackrox.io/scanner-v2:${scanner_v2_tag}"
    local scanner_v2_db_image="stackrox.io/scanner-v2-db:${scanner_v2_tag}"

    # Collector uses the version contained in the COLLECTOR_VERSION file.
    local collector_tag="$(cat COLLECTOR_VERSION)"
    local collector_image="collector.stackrox.io/collector:${collector_tag}"

    docker pull "$main_image"          | cat
    docker pull "$monitoring_image"    | cat
    docker pull "$scanner_image"       | cat
    docker pull "$scanner_v2_image"    | cat
    docker pull "$scanner_v2_db_image" | cat
    docker pull "$collector_image"     | cat

    cd "$(dirname "${BASH_SOURCE[0]}")"
    docker save "$main_image"          -o "image-bundle/main.img"
    docker save "$monitoring_image"    -o "image-bundle/monitoring.img"
    docker save "$scanner_image"       -o "image-bundle/scanner.img"
    docker save "$scanner_v2_image"    -o "image-bundle/scanner_v2.img"
    docker save "$scanner_v2_db_image" -o "image-bundle/scanner_v2_db.img"
    docker save "$collector_image"     -o "image-collector-bundle/collector.img"

    gsutil -m cp -r "gs://sr-roxc/${main_tag}/bin/darwin"  image-bundle/bin/darwin
    gsutil -m cp -r "gs://sr-roxc/${main_tag}/bin/linux"   image-bundle/bin/linux
    gsutil -m cp -r "gs://sr-roxc/${main_tag}/bin/windows" image-bundle/bin/windows
    chmod +x image-bundle/bin/darwin/roxctl image-bundle/bin/linux/roxctl

    tar -czvf image-bundle.tgz           image-bundle
    tar -czvf image-collector-bundle.tgz image-collector-bundle
}

main "$@"
