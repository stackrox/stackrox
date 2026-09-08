---
name: verify-acs-console
description: Verify the ACS (StackRox/RHACS) standalone console UI at ui/apps/platform. Use when an agent needs to launch the Vite dev server, run Cypress component or E2E harnesses, capture proof artifacts, or confirm a user-facing feature still works without inventing ports, scripts, or selectors.
---

# Verify ACS Console UI

StackRox / RHACS standalone console SPA lives in `ui/apps/platform`. Users reach it in production through Central; locally via `npm run start` (Vite on **https://localhost:3000**) proxying API calls to Central at **https://localhost:8000** by default.

**Preferred agent path without a cluster:** Cypress **component** tests (`npm run test-component`). They start their own Vite dev server (SSL disabled via `CYPRESS_COMPONENT_TEST`) and need no Central.

**Full-stack path:** deploy Central (`npm run deploy-local` from `ui/`), then `npm run start` and/or Cypress E2E (`npm run test-e2e-local`). E2E requires auth creds from `scripts/k8s/export-basic-auth-creds.sh` (wired in `scripts/cypress.sh`).

Feature map: [features/README.md](./features/README.md).

## Launch

### Component harness (no Central required)

Component tests bundle and serve via Cypress + Vite automatically. No separate launch step.

```bash
cd ui/apps/platform
npm run test-component -- --spec src/Containers/Dashboard/Widgets/ViolationsByPolicySeverity.cy.jsx
```

**Ready signal:** Cypress exits 0 and prints passing specs.

**Teardown:** none (process exits). Optional: `npm run clean` removes `cypress/test-results` but not evidence copied to `/opt/cursor/artifacts/`.

### Full UI dev server (needs Central or `UI_START_TARGET`)

Documented dev command from `ui/apps/platform/README.md`:

```bash
cd ui/apps/platform
npm run start
```

Or from `ui/` root: `npm run start` (sets `CI=true`).

Proxy target defaults to `https://localhost:8000`. Override with `UI_START_TARGET=https://<central-host>:443 npm run start`.

**Ready signal:** `curl -sk https://localhost:3000` returns HTML (Vite dev server with basic SSL).

**Teardown (helper):**

```bash
.cursor/skills/verify-acs-console/helpers/cleanup.sh
```

**Side-by-side instances:** `vite.config.js` binds **port 3000** only. A second `npm run start` conflicts unless you change `server.port`. Component tests can run while the dev server is down; they use a separate Cypress-managed dev server.

## Doctor

Read-only preflight. Does not start services.

```bash
.cursor/skills/verify-acs-console/helpers/doctor.sh
```

Checks: `ui/apps/platform` layout, `node_modules`, Node `>=22.13.0`, Cypress binary, optional reachability of `https://localhost:3000` and `https://localhost:8000`.

Install deps once:

```bash
cd ui/apps/platform && npm ci
```

## Drive

Harness priority (from `TESTING.md`):

1. **Component** — co-located `*.cy.jsx` / `*.cy.tsx` under `src/`
2. **E2E standalone** — `cypress/integration/**/*.test.{js,ts}` against `https://localhost:3000` + Central
3. **E2E OCP plugin** — `cypress/integration-ocp/` (OpenShift cluster + console; see `TESTING_E2E.md`)

### Component: mapped feature keys

```bash
.cursor/skills/verify-acs-console/helpers/drive-component.sh dashboard
```

| Feature key | Spec | Notable selectors / commands |
|-------------|------|------------------------------|
| `dashboard` | `src/Containers/Dashboard/Widgets/ViolationsByPolicySeverity.cy.jsx` | `cy.findByText(\`${alertCount} policy violations by severity\`)`; severity tiles `cy.get('a').contains(/220\s*Low/)`; links `cy.findByText('View all')` |
| `scope-bar` | `src/Containers/Dashboard/ScopeBar.cy.jsx` | `cy.findByLabelText('Select clusters')`; `cy.findByLabelText('Select namespaces')` |
| `code-viewer` | `src/Components/CodeViewer.cy.jsx` | `cy.get('button[aria-label="Copy code to clipboard"]')` |

Mount pattern (from `TESTING_COMPONENT.md`):

```jsx
cy.intercept('POST', graphqlUrl('alertCountsBySeverity'), ...); // before mount
cy.mount(<ComponentTestProvider><Component /></ComponentTestProvider>);
```

Use `cy.findByRole` / `cy.findByLabelText` before raw `cy.get`. Never `cy.wait(ms)`.

### E2E (Central required)

```bash
cd ui/apps/platform
npm run cypress-spec -- "dashboard/dashboardSummaryCounts.test.js"
```

Base URL: `https://localhost:3000` (`cypress.config.js`). Auth: `CYPRESS_ROX_AUTH_TOKEN` from `scripts/get-auth-token.sh` (`scripts/cypress.sh`).

## Evidence

After a drive run, copy Cypress artifacts to a path that survives cleanup:

```bash
.cursor/skills/verify-acs-console/helpers/capture-evidence.sh
# optional run id:
.cursor/skills/verify-acs-console/helpers/capture-evidence.sh 20260908T120000Z
```

**Action:** run `drive-component.sh` (or E2E spec).

**Resulting state:** JUnit under `cypress/test-results/`, screenshots/videos under `cypress/test-results/artifacts/{screenshots,videos}/`.

**Persistent proof:** `/opt/cursor/artifacts/verify-acs-console/<run-id>/` plus manifest `.cursor/skills/verify-acs-console/.last-evidence-path`.

Confirm after cleanup:

```bash
cat .cursor/skills/verify-acs-console/.last-evidence-path
ls "$(cat .cursor/skills/verify-acs-console/.last-evidence-path)"
```

Do not commit media into the git branch.

## Cleanup

Kill only what this skill started:

```bash
.cursor/skills/verify-acs-console/helpers/cleanup.sh
```

Does **not** delete `/opt/cursor/artifacts/verify-acs-console/`.

## Helpers

All helpers are executable bash scripts under `.cursor/skills/verify-acs-console/helpers/`.

| Helper | Invocation |
|--------|------------|
| Doctor | `.cursor/skills/verify-acs-console/helpers/doctor.sh` |
| Launch UI | `.cursor/skills/verify-acs-console/helpers/launch-ui.sh` |
| Drive component feature | `.cursor/skills/verify-acs-console/helpers/drive-component.sh <feature-key>` |
| Capture evidence | `.cursor/skills/verify-acs-console/helpers/capture-evidence.sh [run-id]` |
| Cleanup | `.cursor/skills/verify-acs-console/helpers/cleanup.sh` |

End-to-end agent proof (component, no cluster):

```bash
cd ui/apps/platform && npm ci
.cursor/skills/verify-acs-console/helpers/doctor.sh
.cursor/skills/verify-acs-console/helpers/drive-component.sh dashboard
.cursor/skills/verify-acs-console/helpers/capture-evidence.sh
.cursor/skills/verify-acs-console/helpers/cleanup.sh
test -d "$(cat .cursor/skills/verify-acs-console/.last-evidence-path)"
```
