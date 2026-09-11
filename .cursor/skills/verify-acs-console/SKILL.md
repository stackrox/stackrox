---
name: verify-acs-console
description: "Drive the ACS (StackRox) web console the way a user does — Vite UI plus live Central — and capture proof. Use when a change touches ui/apps/platform, ACS routes (Dashboard, Workload CVEs, Search, Network Graph), or when an agent would otherwise ship Cypress-component screenshots as console evidence."
---

# Verify the ACS console

This skill is for the **standalone ACS / StackRox console**: the React 18 + TypeScript + Vite SPA in `ui/apps/platform`, talking to a running **Central**. Read this cold. Do not invent selectors, ports, or a passing live run.

**Surface.** Browser UI at `https://localhost:3000` (Vite). API via Vite proxy to Central (`UI_START_TARGET`, default `https://localhost:8000`). PatternFly v6 chrome (`pf-v6-c-*`). Login at `/login`. After auth, left nav + masthead.

**Not this skill.** OpenShift Console plugin (`start:ocp-plugin`, `cypress/integration-ocp/`). Vitest unit tests. Cypress **component** tests (`npm run test-component`). Those do not prove the console.

## Isolation (read before Launch)

Full Central + k8s **cannot run two instances easily**. Local deploy uses namespace `stackrox`, Central port-forward **8000→8443**, and Vite **3000**. `scripts/port-forward-ui.sh` exits if 8000 is already bound. A typical Cursor cloud VM has **no kubectl, no cluster, no container engine** — `deploy-local` will not start there.

- **Refuse to double-drive a shared cluster.** If `stackrox` already has Central, or 8000/3000 are owned by someone else's session, do not deploy another stack, do not steal the port-forward, do not click around a user's live session.
- **If Central is not up, Doctor fails.** Report **inconclusive**. Do not substitute Cypress component tests, Vitest, or fixture-stubbed GraphQL as console proof.
- **NOTE — live prove-once on this generator host:** blocked. This environment has Node 22 but no kubectl, no Docker/Podman, no k8s, no `ui/apps/platform/node_modules`, and no Central. Doctor can check deps and repo files only. Do not invent a passing live run.

## Launch

Work from the repo root unless a command says otherwise.

### 0. Preconditions

- Node `>=22.13.0` (see `ui/package.json` / `ui/apps/platform/package.json` `engines`).
- For a **local** Central: a k8s cluster (`kubectl` on `docker-desktop`, minikube, colima, or podman-desktop). Clean git tree **or** `MAIN_IMAGE_TAG` set (UI README: branch needs a CI tag unless you override the image).
- UI deps, once per checkout:

```sh
cd ui/apps/platform && npm ci
```

### 1. Central (k8s) — or point at an existing one

**Local deploy** (from `ui/`; wraps `../deploy/k8s/deploy-local.sh` → `deploy/k8s/deploy.sh`):

```sh
cd ui
# optional: MAIN_IMAGE_TAG=<ci-tag>
npm run deploy-local
```

That deploys ACS into namespace `stackrox` and typically port-forwards Central to **`https://localhost:8000`** (`LOCAL_PORT` default 8000 → pod 8443). Ready when deploy prints `Successfully deployed Central!` / `Access the UI at: https://localhost:8000` and `https://localhost:8000/v1/ping` returns 200.

Admin user is **`admin`**. Password is written to:

`deploy/k8s/central-deploy/password`

Load it with the repo helper (do not echo it into logs):

```sh
# from repo root
source scripts/k8s/export-basic-auth-creds.sh deploy/k8s
# sets ROX_USERNAME=admin and ROX_ADMIN_PASSWORD
```

If port-forward died later:

```sh
# from ui/
npm run forward
# ../scripts/port-forward-ui.sh — refuses if 8000 is already bound
```

**Remote Central** (no local deploy). From `ui/`:

```sh
UI_START_TARGET=https://<central-host>:443 npm run start
```

`UI_START_TARGET` must include the scheme. Prefer a stable IP/LB; `kubectl port-forward` drops on sleep/load (`ui/README.md`). Extra overrides: `UI_CUSTOM_PROXIES` (`ui/apps/platform/README.md`).

Agent-oriented deploy (non-interactive) also exists as `roxie deploy --envrc <file>` / `scripts/roxie.sh` (`deploy/AGENTS.md`). Use `--envrc` so the admin password is not dumped. Teardown for a roxie install is `roxie teardown` (or `scripts/teardown.sh`). **Do not teardown a cluster this run did not start.**

### 2. Vite UI

From `ui/` (forwards to `apps/platform`):

```sh
cd ui
npm run start
# equivalent: CI=true npm --prefix apps/platform run start
# or: cd ui/apps/platform && npm run start   # vite serve
```

Vite listens on **https://localhost:3000** (`vite.config.js` `server.port` / `preview.port`). In development it uses HTTPS (`@vitejs/plugin-basic-ssl`, or `SSL_CRT_FILE` + `SSL_KEY_FILE`). Proxy (`src/setupProxy.js`): `/v1`, `/v2`, `/api`, `/docs`, `/sso` → `UI_START_TARGET` or `https://localhost:8000`.

**Ready.** Vite logs a local HTTPS URL on 3000. Then:

- `https://localhost:3000/login` loads (self-signed cert — accept / `curl -sk`).
- Heading **Log in to your account**, fields **Username** / **Password**, button **Log in** (`LoginPage.jsx`).
- After login, `https://localhost:3000/main/dashboard` shows `h1` **Dashboard** and left-nav `Dashboard` with `pf-m-current`.

Default branding title suffix is `StackRox` unless `ROX_PRODUCT_BRANDING=RHACS_BRANDING` (`productBranding.ts`).

### 3. Teardown (what this run started only)

See [Cleanup](#cleanup). Record PIDs when you start Vite or `kubectl port-forward`.

## Doctor

Read-only. Run first whenever anything looks off. Invoke the helper (do not re-implement):

```sh
# from repo root
.cursor/skills/verify-acs-console/scripts/doctor.sh
```

Optional env: `UI_START_TARGET`, `ROX_ADMIN_PASSWORD`, `ROX_ADMIN_PASSWORD_FILE`, `VERIFY_ACS_UI_PORT` (default 3000).

The script checks: UI tree and Cypress e2e files; Node major; `node_modules`; whether 3000 is listening and `/login` answers; `GET {Central}/v1/ping`; admin basic auth against `/v1/auth/status` or `/v1/featureflags` if a password is available; kubectl/cluster **as a warning** (shared instance).

| Verdict | Meaning |
| --- | --- |
| exit 0 `READY` | Vite + Central + auth — worth driving |
| exit 1 `NOT READY` | Central up but UI/auth/config broken — fix, do not fake proof |
| exit 2 `INCONCLUSIVE` | Central not reachable — **stop**. Say inconclusive. No component-Cypress substitute |

## Drive

**Prefer the existing Cypress e2e harness** against a **live** Central. Playwright is not a product harness here (only a Vitest browser optional dep). If Cypress cannot run, use a real browser / CDP against `https://localhost:3000` with the same routes and ARIA/PatternFly selectors. **No fake coordinates.**

Auth for Cypress: `ui/apps/platform/scripts/cypress.sh` sources `deploy/k8s` passwords, sets `CYPRESS_ROX_AUTH_TOKEN` via `scripts/get-auth-token.sh` (`POST /v1/apitokens/generate` as admin, role `Admin`). Specs call `withAuth()` which sets `localStorage.access_token`. Base URL: `https://localhost:3000` (`cypress.config.js`). Videos on.

```sh
cd ui/apps/platform
# one mapped spec (after Doctor READY)
npm run cypress-spec -- "dashboard/dashboardSummaryCounts.test.js"
# or interactive
npm run cypress-open
```

**Live proof vs fixture-heavy specs.** Many e2e files pass `staticResponseMap` / `cy.intercept` fixtures (Workload CVE list is a common case). A green spec that only asserts mocked GraphQL is **not** live-console proof. For verification:

1. Doctor READY.
2. Drive the user path **without** replacing Central responses (browser/CDP, or a Cypress visit that does not stub the page's data).
3. Assert visible chrome: left nav, `h1`, PatternFly toolbar/table, real cluster/namespace names if the cluster has them.

**Login (browser / CDP)** — `/login`:

- Title text: `Log in to your account`
- Auth provider select (`.react-select__control`) when more than one provider
- Username: redux-form input `name="username"`
- Password: `name="password"`
- Submit: PatternFly `button` **Log in**
- Success: navigate to `/main/dashboard`, `h1:contains("Dashboard")`

**Stable handles (from this repo, PatternFly v6):**

| What | Handle |
| --- | --- |
| Left nav link | `.pf-v6-c-nav > ul.pf-v6-c-nav__list > li > a` |
| Left nav expandable | `ul.pf-v6-c-nav__list li.pf-v6-c-nav__item button` |
| Current nav | `.pf-v6-c-nav__link.pf-m-current` |
| Horizontal subnav | `nav.pf-m-horizontal.pf-m-subnav a` |
| Dashboard | `/main/dashboard` — `h1` Dashboard; summary tiles `main section:first-child a.pf-v6-c-button.pf-m-link` |
| User Workload CVEs | `/main/vulnerabilities/user-workloads` — `h1` User workload vulnerabilities |
| Platform Workload CVEs | `/main/vulnerabilities/platform` — `h1` Platform vulnerabilities |
| Compound filter entity | `[aria-label="compound search filter entity selector toggle"]` |
| Compound filter attribute | `[aria-label="compound search filter attribute selector toggle"]` |
| Autocomplete value | `[aria-label^="Filter results by"]` (combobox) |
| Apply autocomplete | `button[aria-label="Apply autocomplete input to search"]` |
| Filter chips | `.pf-v6-c-toolbar .pf-v6-c-toolbar__group[aria-label="applied search filters"]` |
| Global Search | masthead button **Search** → `/main/search` — `h1` Search, placeholder `Filter resources` |
| Network Graph | `/main/network-graph` — expand **Network** → **Network Graph** |
| NG cluster/namespace | `nav[aria-label="Breadcrumb"]` `[aria-label="Select a cluster"]` / `Select namespaces` |

Feature recipes: [features/README.md](features/README.md). Drive **every** mapped entry the task claims, not one shortcut.

## Evidence

Write artifacts under **`${VERIFY_ACS_EVIDENCE_DIR:-/tmp/verify-acs-console}`** (host path, not git). Do **not** commit screenshots, videos, or binaries.

Must include:

1. **Product chrome** — left nav and/or masthead + branding (StackRox or RHACS logo / document title). A cropped widget with no chrome is not console proof.
2. **Action + resulting state** — e.g. before filter / after chip + table change; not only the final screen.
3. **Side effects** — URL (`/main/...` and query), network (`/v1/ping`, GraphQL, `/v1/networkgraph/cluster/*`), and any cluster mutation (CIDR save, watched image). Prefer read-only drives.
4. **Doctor transcript** — the `doctor.sh` output from this run.

Cypress videos/screenshots (when used) land in `ui/apps/platform/cypress/test-results/artifacts/{videos,screenshots}` (`cypress.sh`). Copy the relevant files into the evidence dir; do not git-add them.

**Component-only Cypress is NOT full console proof.** `npm run test-component` mounts a component with Vite, **SSL off**, mocked APIs, no Central, no left nav. Say that explicitly if that is all you ran.

## Cleanup

Kill **only** what this verification started. Never `killall node` / `killall kubectl`.

- Vite: the PID of `vite serve` / `npm run start` you launched.
- Port-forward: the `kubectl port-forward` PID you launched (`scripts/port-forward-ui.sh` backgrounds one).
- Central/k8s: only if **this run** ran `npm run deploy-local` or `roxie deploy`. Then `scripts/teardown.sh` (roxie teardown, single namespace) or `roxie teardown`. Do not teardown a shared `stackrox` namespace.
- **Keep evidence.** Do not delete `${VERIFY_ACS_EVIDENCE_DIR:-/tmp/verify-acs-console}`.

## Helpers

| Script | Role |
| --- | --- |
| `.cursor/skills/verify-acs-console/scripts/doctor.sh` | Read-only Doctor. Executable. Invoke as shown in [Doctor](#doctor). |

No other helper is required. Cypress wrappers already live in `ui/apps/platform/scripts/cypress.sh` and `get-auth-token.sh`.
