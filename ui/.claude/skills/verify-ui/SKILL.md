---
name: verify-ui
description: Prove StackRox UI behavior in the real app by driving the Vite dev server from this checkout with Cypress against a live Central (infra qa-demo), capturing screenshots, video, and page-state evidence. Use when asked to verify, prove, demo, or check a UI change in the running app, reproduce a UI bug end to end, or when "it type-checks" is not enough evidence for a ui/ change. Not for writing checked-in e2e tests (use cypress-e2e-test).
---

# verify-ui

Drive the StackRox web UI the way a user does and keep proof. The stack is:

```
Cypress (electron, headless) -> https://localhost:3000 (Vite, this checkout)
                                    | proxies /v1 /v2 /api /docs /sso
                                    v
                               Central (UI_START_TARGET, infra qa-demo)
```

All scripts live in `ui/.claude/skills/verify-ui/scripts/`. Run them from any directory. State and evidence live in `~/.cache/stackrox-verify-ui/` (override with `VERIFY_UI_HOME`), outside the git tree.

## 0. Prerequisites (check once)

- Node 22.13 or newer (`engines.node` in `ui/apps/platform/package.json`).
- `node_modules` installed in `ui/apps/platform/` (`cd ui/apps/platform && npm ci`), which includes Cypress 15 and its Electron. `ui/` itself has no lockfile, so `npm ci` there fails.
- `infractl` on PATH and logged in (`infractl whoami` prints your service account).
- `jq`, `curl`, `docker` (only for the image-exists check in `backend.sh create`).
- No gcloud/kubectl needed: the qa-demo artifacts carry the Central URL and admin password.

## 1. Backend: get a Central

Reuse one if `scripts/backend.sh status` already shows a READY cluster. Otherwise:

```bash
S=ui/.claude/skills/verify-ui/scripts
name=$($S/backend.sh create)        # qa-demo running quay.io/rhacs-eng/main:<make tag>; ~30-45 min
$S/backend.sh wait "$name"          # run in background; blocks until READY
$S/backend.sh connect "$name"       # stores URL + admin password (0600) in the state dir
```

- `create` fails fast if the image for local HEAD is not on quay (unpushed commit). Then either check out a pushed commit, or pass an older pushed image's cluster via `connect`. UI-only branches are fine on a slightly older Central; doctor warns on version drift.
- Any other Central works: `backend.sh connect-url https://<host>:<port> <password-file>`.
- Infra clusters are shared GCP cost and internet-facing (public LB, admin password). Default lifespan 8h (`VERIFY_UI_LIFESPAN`). Delete when done.
- `create` passes `--no-slack` (nothing posted to #infra_notifications) and uses zone `us-central1-a` (`VERIFY_UI_GCP_ZONE`). If it FAILS with "does not have enough resources", delete it and retry in another zone.

## 2. Launch

```bash
$S/launch.sh       # vite serve --port 3000 --strictPort, proxied to the connected Central
```

Ready when it prints `dev server ready at https://localhost:3000`. Logs: `~/.cache/stackrox-verify-ui/state/vite.log`. Vite hot-reloads, so keep it running across edits.

Isolation: one dev server per port. `launch.sh` refuses a port it did not claim (do not kill the user's own `npm run start`; set `VERIFY_UI_PORT=3100` instead). Parallel runs on one Central are safe for read-only flows; flows that mutate shared objects (policies, exceptions, integrations) must use unique names.

## 3. Doctor (run first, and whenever anything looks off)

```bash
$S/doctor.sh
```

Read-only. Checks: backend connected, infra READY + lifespan left, Central accepts admin basic auth, Central version vs `make tag`, dev server pid owns the port and runs from this checkout, app shell served, `/v1` proxied. Ends with `healthy` or the first `FAIL`.

## 4. Drive

Write a throwaway proof spec, then run it:

```bash
spec=$($S/new-spec.sh violations-filter)   # ~/.cache/stackrox-verify-ui/runs/<run-id>/specs/violations-filter.cy.js
# edit the it() body using the feature map recipe
$S/drive.sh "$spec"                         # prints PASS/FAIL summary, last line is the run dir
```

`drive.sh` mints a per-run Admin API token (`verify-ui-<run-id>`) and puts it in localStorage through `cypress/helpers/basicAuth.js`. It also exports Central feature flags as `CYPRESS_ROX_*` (same as `scripts/cypress.sh`) and runs Cypress headless with retries off and video on.

Spec rules:
- Start from the matching file in `features/`. It lists entry points, helpers, selectors, and what proves the feature works.
- Reuse repo helpers by absolute import (`<platform>/cypress/helpers/...`, `<platform>/cypress/integration/<area>/*.helpers`). The template already imports `visit`, `withAuth`, and the evidence helpers.
- Do not use repo helpers that take a fixture (`...WithFixture`, a `staticResponseMap`) for the behavior under proof. Those stub Central and prove nothing about the real app.
- Prefer accessible names and text (`button:contains("Create policy")`, `cy.get('h1:contains("Violations")')`) over CSS classes and positions.
- Wait on responses (`interactAndWaitForResponses`) or visible state, never `cy.wait(ms)`.
- Call `captureState('<NN-step>')` after each action whose result is part of the proof.
- An existing checked-in spec can also be driven as is: `$S/drive.sh cypress/integration/<area>/<file>.test.js`. That proves the suite passes, not that a specific change behaves; pair it with a proof spec when the change is new behavior.

Iterate: when a selector fails, read the failing step in `evidence/cypress.log` and the last screenshot, fix the spec, and rerun `drive.sh` on the same path. Each rerun replaces that run's evidence.

## 5. Evidence

Each run leaves `~/.cache/stackrox-verify-ui/runs/<run-id>/evidence/`:

| File | Shows |
|---|---|
| `summary.txt` | PASS/FAIL, branch @ sha (+ uncommitted flag), Central URL + version, time |
| `<spec>.cy.js` | the exact steps that ran |
| `screenshots/<spec>/<NN-step>.png` | the full app (1440x850, scrolled to top) after each captured step; failures auto-captured by Cypress |
| `<NN-step>.state.json` | URL, title, headings, alerts, table row count at that step |
| `videos/<spec>.mp4` | the whole run |
| `console-errors.json` | app `console.error` calls during the run (should be `[]` or explained) |
| `cypress.log` | full runner output |

Proof standards:
- Exercise the real user path (nav, buttons, typing), not `cy.request` setters or stubbed responses.
- Capture the action and the result, not just the final screen.
- For mutations, prove persistence from a second view: reload, or reopen from the list.
- A claim of "verified" cites the run dir, `summary.txt` result, and the screenshots that show the behavior. An unreachable path is reported as unverified with the failing command, never as verified through a different path.
- Privacy: screenshots and video show cluster names, IPs, and anything on screen (including freshly generated tokens). Review before attaching to a PR or Jira; never commit them.

## 6. Cleanup

```bash
$S/cleanup.sh              # stop the dev server we started, revoke verify-ui-* tokens
$S/cleanup.sh --backend    # also infractl delete the connected cluster
```

Kills only the recorded pid, never by process name. Evidence under `runs/` survives cleanup; prune old runs by hand.

## Gotchas

- Dev server is HTTPS with a self-signed cert (`@vitejs/plugin-basic-ssl`). Use `https://localhost:3000`; `http://` gets no response.
- The login page defaults to an SSO provider that is not the demo's. Proof specs skip login via the API token; to prove login itself, see `features/login.md`.
- Feature-flagged UI: check `CYPRESS_ROX_*` values in `cypress.log` context or `curl` `/v1/featureflags`. A qa-demo uses the image's default flags; you cannot flip them without kubectl access to the cluster.
- Central data on qa-demo is live and changes over time (violations, CVEs). Assert on shape and presence, not on exact counts, unless the spec created the data.
- `retries=0` on purpose: a flaky proof is not a proof. Fix the wait, not the retry count.
- `drive.sh` runs with `templates/cypress.config.js` (repo config plus a 2000x1100 window). Headless Electron defaults to 1280x720, which crops the 1440x850 viewport.
- `captureState` screenshots the app frame through the DevTools protocol, not `cy.screenshot`, which hides the command log and makes the video flash. It captures the whole window and crops on a canvas: a `clip` or `scale` makes Chrome briefly resize the page, which also flashes in the video. Electron/Chrome only.

## Maintaining

The feature map in `features/` is the source of truth for how to reach and prove each feature. When a recipe drifts from the app, fix the map in the same change. pstack's `/maintain-verification-skill` audits it end to end.
