# ACS console verification map

This directory is the maintained source for verifying user-facing behavior of the standalone ACS / StackRox console (`ui/apps/platform`). Read this index, then the feature file for the path you are proving.

## Baseline preconditions

- Doctor is **READY** (`.cursor/skills/verify-acs-console/scripts/doctor.sh` exit 0). If Doctor is **INCONCLUSIVE**, stop and say so.
- Vite at `https://localhost:3000`, Central at `UI_START_TARGET` or `https://localhost:8000`.
- Signed in as `admin` (password from `deploy/k8s/central-deploy/password` or `ROX_ADMIN_PASSWORD`), or Cypress `withAuth()` / `ROX_AUTH_TOKEN`.
- Do not drive a Central this run did not start unless the user explicitly handed you that instance **and** you will not mutate it.
- Evidence goes to `${VERIFY_ACS_EVIDENCE_DIR:-/tmp/verify-acs-console}`. Keep it after cleanup.

## Driving conventions

- Prefer Cypress e2e helpers in `ui/apps/platform/cypress/helpers/` and feature `*.helpers.js` **without** `staticResponseMap` fixtures when the claim is live console proof.
- Prefer ARIA labels, visible text, and `pf-v6-c-*` over DOM position or coordinates.
- Start from Dashboard (`/main/dashboard`) unless a recipe says otherwise.
- Restore filters you applied. Do not create watched images, CIDR blocks, or exception requests unless the task requires a mutation — then clean those up.

## Proof and skip reporting

- Capture action + resulting state, with left-nav / masthead chrome visible.
- Record the feature file and route used.
- An unreachable path (no cluster, empty graph, missing RBAC) is **inconclusive** or a documented empty state — not a pass via a different page.
- Cypress component tests are not a substitute for any entry below.

## Features

- [Dashboard](./dashboard.md) — summary counts, widgets, scope bar.
- [Workload CVE list](./workload-cve-list.md) — User Workloads / Platform vulnerability tables.
- [Search and filter autocomplete](./search-filter-autocomplete.md) — masthead Search and compound CVE filters.
- [Network Graph](./network-graph.md) — cluster/namespace scope and topology chrome.
