#! /bin/bash

source "scripts/ci/lib.sh"

GOVULNCHECK_BIN="$(make  which-govulncheck --silent)"
find bin/linux_amd64 -type f | while read -r file;
do
    echo "Analyzing binary $file"
    $GOVULNCHECK_BIN -mode=binary --json "$file" > vulns.json
    cat vulns.json >> all_vulns.json
    go run govulncheck/main.go vulns.json > filtered_vulns.json
    cat filtered_vulns.json >> all_vulns.json
    jq '.data[] | select(.id) | [.id, .summary, .details] | @tsv' -r filtered_vulns.json | sort -u | while IFS=$'\t' read -r -a vulns
    do
      id="${vulns[0]}"
      summary="${vulns[1]}"
      details="${vulns[2]}"
      echo "$id" "$summary"
      save_junit_failure "$id" "$summary" "$details"
    done
    if [[ $(jq '.data | length' < filtered_vulns.json) == 0 ]]; then
      save_junit_success "$(basename "$file")" "go scan"
    fi
done

go run govulncheck/main.go all_vulns.json > results.json
if [[ $(jq '.data | length' < results.json) != 0 ]]; then
    jq '.data | length' < results.json
    echo "Found vulnerabilities. If they are false positives, add them to govulncheck-allowlist.json" >&2
    cat results.json
    exit 1
fi
