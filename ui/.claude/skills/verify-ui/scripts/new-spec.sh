#!/usr/bin/env bash
# Create a proof spec for one run outside the git tree and print its path.
#
#   new-spec.sh <name> [run-name]   name is kebab-case, e.g. violations-filter; run-name
#                                   defaults to name. drive.sh keeps attempts under it.

source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

name="${1:?usage: new-spec.sh <name> [run-name]}"
run_id="${2:-${name}}"
spec_dir="${RUNS_DIR}/${run_id}/specs"
spec="${spec_dir}/${name}.cy.js"

mkdir -p "${spec_dir}"
[[ -e "${spec}" ]] && die "${spec} already exists; edit it and rerun drive.sh, or pick another name"
sed -e "s|__PLATFORM__|${PLATFORM_DIR}|g" -e "s|__SKILL__|${SKILL_DIR}|g" -e "s|__NAME__|${name}|g" \
    "${SKILL_DIR}/templates/proof.cy.js" > "${spec}"
echo "${spec}"
