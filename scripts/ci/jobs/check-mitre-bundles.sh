#!/bin/bash

set -xveou pipefail

#  check-mitre-attack-bundle-up-to-date:
#    executor: custom
#    resource_class: small
#    steps:
#      - run:
#          name: Skipping until ROX-8486 is resolved
#          command: |
#            echo "Skipping until ROX-8486 is resolved"
#            circleci step halt
#
#      - run:
#          name: Determine whether to skip MITRE ATT&CK bundle check
#          command: |
#            if [[ -z "${CIRCLE_TAG}" ]]; then
#              echo "Not a tagged build, skipping MITRE ATT&CK bundle check"
#              circleci step halt
#            fi
#
#      - checkout
#      - restore-go-mod-cache
#      - setup-go-build-env
#
#      - run:

# shellcheck disable=SC2016
echo 'Ensure MITRE ATT&CK bundle at "./pkg/mitre/files/mitre.json" is up-to-date. (If this fails, run `mitreutil fetch` and commit the result.)'

function check_mitre_attach_bundle_up_to_date() {
    make deps
    make mitre
    mitre fetch --domain enterprise --out /tmp/enterprise-mitre.json
    diff pkg/mitre/files/mitre.json /tmp/enterprise-mitre.json > /tmp/mitre-diff || true
    if [[ -s /tmp/mitre-diff ]]; then
        echo 'error: MITRE ATT&CK bundle at 'pkg/mitre/files/mitre.json' is not up-to-date. Check "mitre-diff" for more informtaion.'
        cat /tmp/mitre-diff
        exit 1
    fi
}
#      - ci-artifacts/store:
#          path: /tmp/mitre-diff
#          destination: mitre-diff

check_mitre_attach_bundle_up_to_date
