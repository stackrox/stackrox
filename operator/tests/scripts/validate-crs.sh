#!/usr/bin/env bash

set -euo pipefail
FAILED=0
CRS_VALIDATED=0

die() {
    echo "$@" >&2
    exit 1
}

if (( $# < 2 )); then
    die "Usage: $0 <directory with kuttl tests> <CR 1> <CR 2> ..."
fi

TESTDIR="$1"
shift

CRDS=("$@")
echo "INFO: Validating all Kubernetes object definitions for the following CRDs: ${CRDS[*]}"
echo

# Extract kind names for the CRDs.
CRD_KINDS=$(
    for crd in "${CRDS[@]}"; do
        if kubectl get crd "$crd" -o jsonpath='{.spec.names.kind}'; then
            echo
        else
            die "Failed to lookup kind name for CRD $crd. Make sure CRDs are applied (make install) before validation is attempted."
        fi
    done
    )

# Locate CRs for the kinds extract above.
CRS=$(
    for kind in $CRD_KINDS; do
        find "$TESTDIR" -type f -name "*.yaml" \! -name "*-assert.yaml" \! -name "*-errors.yaml" \! -name "*.envsubst.yaml" -exec grep -Eiq "^kind: *${kind} *$" {} \; -print
    done | sort -u
    )

# Validate CRs.
for cr in $CRS; do
    if grep -q '^apiVersion: kuttl.dev/' "${cr}"; then
        # TODO(ROX-19283): make it possible to validate these files somehow.
        echo "Skipping ${cr} since it contains kuttl CR(s)."
        continue
    fi
    echo -n "Validating custom resource $cr with kubectl... "
    if output=$(kubectl apply --dry-run=client --validate=true -f "$cr" 2>&1); then
        echo PASSED
    else
        FAILED=1
        echo FAILED
        echo "kubectl: $output"
    fi
    echo
    CRS_VALIDATED=$((CRS_VALIDATED + 1))
done

if (( FAILED )); then
    die "ERROR: Some custom resources did not pass validation (see above)."
fi

if (( CRS_VALIDATED == 0 )); then
    die "ERROR: No CRs validated, this does not seem correct. CRs were expected in directory \"$TESTDIR\"."
fi
