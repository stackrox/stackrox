# Dashboard

The Dashboard is the post-login home of the ACS console. A signed-in user sees summary count tiles (Cluster, Node, Violation, Deployment, Image, Secret), a scope bar when they can read Cluster and Namespace, and a gallery of widgets (policy violations by severity, images at most risk, deployments at most risk, aging images, policy violations by category, optional compliance by standard).

## Sub-features

- `dashboard-open` shows `h1` Dashboard and current left-nav **Dashboard**.
- `dashboard-summary-counts` shows last-updated text and tile links for resources the user can read.
- `dashboard-widgets` shows `#main-dashboard-widget-gallery` cards with `h2` titles such as Deployments at most risk and Policy violations by category.
- `dashboard-scope` shows the cluster/namespace scope bar when Cluster + Namespace read access exists.

## How to get to it (user POV)

- After **Log in**, the console goes to `/main/dashboard`.
- Click **Dashboard** in the left nav (first link).
- Open `https://localhost:3000/main/dashboard` while authenticated.

## Driving it with Cypress e2e

Preconditions:

- Doctor READY.
- Admin (or Analyst-equivalent) session.
- No `staticResponseMap` on the page’s GraphQL if you claim live Central data.

- **Open from login.** Submit Username `admin` and Password, **Log in**. Result: URL `/main/dashboard`, `h1:contains("Dashboard")`, `.pf-v6-c-nav__link.pf-m-current:contains("Dashboard")`.
- **Open with harness.** From `ui/apps/platform`, `visitMainDashboard()` in `cypress/helpers/main.js` (`visit('/main/dashboard')`). Result: same `h1` and current nav. Title matches `Dashboard | (StackRox|Red Hat Advanced Cluster Security)`.
- **Summary counts.** `main section:first-child a.pf-v6-c-button.pf-m-link:contains("Cluster")` (and Node, Violation, Deployment, Image, Secret as permissions allow). `main section:first-child div:contains("Last updated")`. Tiles are links, not raw numbers only.
- **Widgets.** `#main-dashboard-widget-gallery` exists. Visible `h2` includes **Deployments at most risk** and **Policy violations by category**. Severity widget title is `{n} policy violation(s) by severity`. Images widget is **Images at most risk** or **Active images at most risk**.
- **Proof.** Screenshot of left nav + tiles + at least one widget. Record `/v1` or GraphQL traffic (summary counts / widget ops in `cypress/helpers/main.js`). Copy artifacts to `${VERIFY_ACS_EVIDENCE_DIR:-/tmp/verify-acs-console}`.

Note: `dashboard/dashboardSummaryCounts.test.js` overrides permissions with static `/v1/mypermissions` responses. That spec is good for RBAC chrome, not for “Central returned these counts.” For live counts, visit without that static response.

## Gotchas

- With no Cluster/Namespace read access, the scope bar is absent; that is expected, not a failure.
- Compliance widget only appears if `ROX_DEPRECATED_COMPLIANCE_DASHBOARD` is on and Compliance is readable.
- Empty or zero counts on a fresh cluster are valid. Assert chrome and “Last updated”, not a hardcoded CVE or deployment name.
- A component screenshot of `SummaryCounts` without the page `h1` and nav is not Dashboard proof.
