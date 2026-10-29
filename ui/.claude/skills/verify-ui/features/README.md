# StackRox UI verification map

Maintained source for proving user-facing UI behavior with verify-ui. Read this index, then use the matching feature file as the recipe for the proof spec.

## Baseline preconditions

- `scripts/doctor.sh` prints `healthy`: backend connected, dev server from this checkout on `https://localhost:3000`, `/v1` proxied to Central.
- Central has a secured cluster reporting data: a roxie deployment of Central plus SecuredCluster, or an infra qa-demo (more demo workloads, violations, images, and CVEs). On a small roxie deployment some views can be empty; each feature's Gotchas names a view that has data.
- Specs run through `scripts/drive.sh`, which authenticates with a per-run Admin API token (`withAuth()`), not the login form.
- Paths below are relative to `ui/apps/platform`. Import them in specs as `<platform>/<path>` (absolute), as the template does.

## Driving conventions

- Start each spec with `withAuth()` (template does this). `drive.sh` records uncaught exceptions and `console.error` calls for every spec.
- Navigate the way a user does when the entry point is part of the proof: left nav (`visitFromLeftNav`, `visitFromLeftNavExpandable` in `cypress/helpers/nav.ts`), then horizontal subnav (`visitFromHorizontalNav`). Use `visit(url)` from `cypress/helpers/visit.js` for deep links.
- Wait with `interactAndWaitForResponses(callback, routeMatcherMap)` from `cypress/helpers/request.js` and leave out `staticResponseMap` so requests hit Central. GraphQL matchers come from `getRouteMatcherMapForGraphQL([opnames])`.
- Repo helpers named `...WithFixture`, `...WithMockedResponses`, and every WorkloadCves `interactAndWaitFor*List` stub Central with fixtures. Do not use them in proofs.
- Selector priority: visible text and accessible names (`h1:contains(...)`, `button:contains(...)`, `[aria-label=...]`), then `td[data-label=...]`, then OUIA (`cypress/selectors/pf6.ts`), and PF class names only as a last resort.
- Demo data drifts. Assert presence and shape (non-empty table, heading equals the clicked name), not exact counts.
- Mutations (create/edit/delete) use a unique name like `verify-ui-<timestamp>` and clean up in an `after()`.

## Proof and skip reporting

- Call `captureState('<NN-step>')` after each action whose result is part of the claim; it saves a screenshot and a `.state.json`.
- Mutation proof needs a second view: reload, or reopen from the list.
- An uncaught exception fails the proof. `console.error` entries in `errors.txt` do not, but explain or flag each one.
- Report the run dir and `summary.txt` result with every claim.
- If an entry point cannot be reached (flag off, no permission, no data), report it as unverified with the failing step. Do not count it as verified through another path.

## Feature entry contract

Each feature file starts with an H1 and one paragraph on user-visible behavior, then exactly four H2s in order: `Sub-features`, `How to get to it (user POV)`, `Driving it with verify-ui`, `Gotchas`. Keep implementation detail out except stable handles, helpers, and observable proof.

## Features

- [Login](./login.md): basic-auth login form, redirect, user menu, logout.
- [Dashboard](./dashboard.md): landing page, summary count tiles, widgets.
- [Violations](./violations.md): workflow views, table, search filter, violation detail tabs.
- [Policies](./policies.md): policy table, policy detail, create-policy wizard.
- [Workload CVEs](./workload-cves.md): vulnerability results views, entity tabs, CVE/image/deployment detail pages.
