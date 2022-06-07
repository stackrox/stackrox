#!/bin/env bash

#  check-generated-files-up-to-date:
#    executor: custom
#    resource_class: medium
#    steps:
#      - checkout
#      - restore-go-mod-cache
#      - setup-go-build-env
#      - run:
#          no_output_timeout: 30m
#          name: Ensure that generated files are up to date. (If this fails, run `make proto-generated-srcs && make go-generated-srcs` and commit the result.)
#          command: |
#            git ls-files --others --exclude-standard >/tmp/untracked
#            make proto-generated-srcs
#            # Print the timestamp along with each new line of output, so we can track how long each command takes
#            make go-generated-srcs 2>&1 | while IFS= read -r line; do printf '[%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$line"; done
#            git diff --exit-code HEAD
#            { git ls-files --others --exclude-standard ; cat /tmp/untracked ; } | sort | uniq -u >/tmp/untracked-new
#            if [[ -s /tmp/untracked-new ]]; then
#              echo 'Found new untracked files after running `make proto-generated-srcs` and `make go-generated-srcs`. Did you forget to `git add` generated mocks and protos?'
#              cat /tmp/untracked-new
#              exit 1
#            fi
#
#      - run:
#          name: Ensure that all TODO references to fixed tickets are gone
#          command: |
#            .circleci/check-pr-fixes.sh
#      - run:
#          name: Ensure that there are no TODO references that the developer has marked as blocking a merge
#          command: |
#            # Matches comments of the form TODO(x), where x can be "DO NOT MERGE/don't-merge"/"dont-merge"/similar
#            ./scripts/check-todos.sh 'do\s?n.*merge'
#
#      - run:
#          name: Check operator files are up to date (If this fails, run `make -C operator manifests generate bundle` and commit the result.)
#          no_output_timeout: 30m
#          command: |
#            set -e
#            make -C operator/ generate
#            make -C operator/ manifests
#            echo 'Checking for diffs after making generate and manifests...'
#            git diff --exit-code HEAD
#            make -C operator/ bundle
#            echo 'Checking for diffs after making bundle...'
#            echo 'If this fails, check if the invocation of the normalize-metadata.py script in operator/Makefile'
#            echo 'needs to change due to formatting changes in the generated files.'
#            git diff --exit-code HEAD
