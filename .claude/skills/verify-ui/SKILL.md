---
name: verify-ui
description: Prove a StackRox UI change works in the running app before reporting it done. Use after editing files under ui/apps/platform, before opening a PR, or when asked to verify, check, or prove a UI change.
---

# Verify a UI change

Lint, tsc, and unit tests pass on code that still breaks the page. The proof is opening the
page in the running app and keeping evidence. Do not report UI work done until this loop
passes, or until you say which step you could not run and why.

All commands run from `ui/apps/platform`.

## The loop

1. **Doctor.** `scripts/verify.sh doctor`. It checks the dev server, Central through the
   proxy, auth, and prints enabled feature flags. Fix what it reports before going on.
   If the dev server is not running, start it yourself as a background process with
   `BROWSER=none npm run start` (it never exits), wait until it answers, and run `doctor`
   again. If Central is down, stop and give the user the ways to get one that `doctor`
   prints. Do not deploy Central or create a cluster unless the user asked you to: it
   uses shared quota, takes a long time, needs their credentials, and which Central to
   use is their call. Run `doctor` again
   after any failed `open` or `prove`, because a half-broken environment gives misleading
   failures.
2. **Static.** `scripts/verify.sh static`. It runs tsc, then eslint and `vitest related` on
   files changed since `origin/master`. Pass file paths to check only those.
3. **Open.** `scripts/verify.sh open <route>` for every route the change touches, for
   example `scripts/verify.sh open /main/violations`. Find routes in `src/routePaths.ts`.
   Use a URL with query params or an entity id when the change is on a detail page or
   behind a filter.
4. **Prove.** `scripts/verify.sh prove <feature>` runs the e2e specs for the feature, where
   the feature is a directory under `cypress/integration`. Run it without an argument to
   list features.
5. **Report.** In the PR description or the final message, list what you ran, the result,
   and the paths to `report.json` and the screenshot.

## Reading `open` results

`report.json` is written to `verify-evidence/<timestamp>-<route>/` with a full-page
screenshot next to it. The folder is gitignored and `npm run clean` does not delete it.

- **Failures** fail the run: API responses of 400 or higher, uncaught exceptions, requests
  still pending after 30 seconds, a spinner or skeleton still visible after 30 seconds,
  "Cannot find the page", or a redirect to login.
- The screenshot is taken once no request of any kind has been in flight for a second and
  no spinner or skeleton is visible.
- **Warnings** do not fail the run: console errors and axe accessibility violations. Some
  pages already have them. Treat any warning that comes from code you touched as a failure.
  If you are not sure, run `open` on the same route on `origin/master` and compare.
- Look at the screenshot. A page can pass every check and still show the wrong thing. A
  green run means the page works, not that it does what was asked.

## Evidence rules

- Capture the action and the state it produced, not only the final screen. For a change
  to a form, a green `open` on the page is not proof the save works: check the saved
  item after reloading.
- A success toast is not proof. Reopen the item.
- Never add test-only code paths or setters to the product to make verification pass.
- Component tests with mocked APIs are useful but weaker evidence than `open`. Say which
  one you have.

## When something fails

Decide which kind of failure it is before fixing anything:

- **Product bug**: the change or the app is wrong. Fix the code, or report it. Never work
  around it in the harness or this skill.
- **Harness gap**: the script or spec cannot express the check, or reports a false failure.
  Fix `scripts/verify.sh` or `cypress/verify/`, and say so in the PR.
- **Doc drift**: this skill or `ui/AGENTS.md` is wrong about how things work. Update it in
  the same PR.

## Environment

- The dev server is `npm run start` (https://localhost:3000). It proxies `/v1`, `/v2`,
  `/api`, `/docs`, and `/sso` to `UI_START_TARGET`, default https://localhost:8000. To use a
  remote Central, start the dev server with `UI_START_TARGET=https://<central>`.
- Auth: set `ROX_AUTH_TOKEN` to reuse a token. Otherwise the script mints one from
  `ROX_USERNAME` and `ROX_ADMIN_PASSWORD`, or from `deploy/k8s/central-deploy/password`
  after a local deploy. Each mint creates a new API token in Central, so export one token
  and reuse it for repeated runs.
- Set `UI_BASE_URL` if the dev server is not on https://localhost:3000.
- Only one agent should drive a shared Central at a time. Two agents changing policies or
  exceptions on the same instance corrupt each other's results.

## Gotchas

- Routes behind a feature flag render "Cannot find the page" when the flag is off. Check
  the flags `doctor` prints before treating that as a bug.
- Routes need read access to the resources in `routeRequirementsMap` in
  `src/routePaths.ts`. The minted token has the Admin role.
- Pages with no data may show an empty state that hides the change. Say so rather than
  claiming the change was verified.
