#!/usr/bin/env bash
# ROX-36961 / Task 1 — intended workflow shape, implementation pending.
# Publication stays inline in scanner-updater-bundle.yaml.
set -euo pipefail

not_implemented() {
    echo "::error::ROX-36961 skeleton: $1 is not implemented" >&2
    return 1
}

# Read generation, creation time, and content-digest from GCS.
read_staged_candidate() {
    not_implemented read_staged_candidate
}

# The only intentionally simulated external dependency in the finished task.
# TODO: Replace with an authenticated Grafana query for the exact stream/SHA.
query_bundle_health() {
    local stream="$1" sha256="$2"
    echo "Normal"
}

# Delete only the inspected generation on rejection or timeout.
remove_staged_candidate() {
    not_implemented remove_staged_candidate
}

# Decide action=publish/skip, generation, and should_stage=true/false.
# Normal -> publish the inspected generation via the separate workflow step.
# Alerting -> reject/remove; Pending/No Data -> wait or expire after 8h.
# No candidate -> allow staging. Force-stage behavior is still to be defined.
evaluate_staging() {
    local stream="$1"
    # TODO: Read candidate, handle age/force-stage and evaluate health.
    # TODO: Emit action, generation and should_stage to GITHUB_OUTPUT.
    # No fabricated publish approval until the GCS evaluation is implemented.
    not_implemented "evaluate_staging($stream)"
}

# Compute whole-ZIP SHA-256, upload to <stream>-staging with
# content-digest=sha256:<hex> and GCS generation preconditions.
stage_bundle() {
    local stream="$1" zip_path="$2"
    not_implemented "stage_bundle($stream, $zip_path)"
}

main() {
    case "${1:-}" in
        evaluate)
            [[ $# -eq 2 ]] || { echo "Usage: $0 evaluate <stream>" >&2; return 2; }
            evaluate_staging "$2"
            ;;
        stage)
            [[ $# -eq 3 ]] || { echo "Usage: $0 stage <stream> <zip>" >&2; return 2; }
            stage_bundle "$2" "$3"
            ;;
        *)
            echo "Usage: $0 {evaluate <stream>|stage <stream> <zip>}" >&2
            return 2
            ;;
    esac
}
main "$@"
