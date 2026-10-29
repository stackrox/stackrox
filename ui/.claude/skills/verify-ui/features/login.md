# Login

A user picks an auth provider on the login page, signs in with username and password, lands on the page they asked for (or the dashboard), and can log out again.

## Sub-features

- `login-redirect` sends an unauthenticated visit to `/main/*` to `/login`.
- `login-basic` signs in with the basic (username/password) provider.
- `login-return` returns to the originally requested page after sign-in.
- `logout` ends the session and returns to `/login`.

## How to get to it (user POV)

- Open any app URL while signed out, e.g. `https://localhost:3000/main/violations`.
- Open `https://localhost:3000/login` directly.
- Signed in: user menu (top right) > `Log out`.

## Driving it with verify-ui

Preconditions:

- `doctor.sh` healthy.
- Do NOT call `withAuth()` in this spec; delete it from the template so the session starts signed out.
- The spec needs the admin password. Pass it only through env: run `CYPRESS_VERIFY_ADMIN_PASSWORD="$(cat ~/.cache/stackrox-verify-ui/state/admin-password)" scripts/drive.sh <spec>` and read it with `cy.env(['VERIFY_ADMIN_PASSWORD']).then(({ VERIFY_ADMIN_PASSWORD }) => ...)`. The repo config sets `allowCypressEnv: false`, so `Cypress.env()` throws, and secrets must never go through `Cypress.expose()`. Never inline it in the spec, which is copied into evidence.

- **Redirect.** `cy.visit('/main/violations')`. Expect `cy.location('pathname').should('eq', '/login')` and `h2:contains("Log in to your account")`. `captureState('01-login-page')`.
- **Pick provider.** The provider select (`.react-select__control`, label "Select an auth provider") defaults to the first provider and is disabled when there is only one. If `input[name="username"]` is not rendered, click `.react-select__control`, then `.react-select__option` for the basic provider. The name comes from the deployment's data and is often `Login with username/password`; confirm it from the screenshot.
- **Sign in.** `cy.get('input[name="username"]').type('admin')`, `cy.get('input[name="password"]').type(password, { log: false })`, then click `button:contains("Log in")` inside `interactAndWaitForResponses` on `{ exchangeToken: { method: 'POST', url: '/v1/authProviders/exchangeToken' } }`. The button stays disabled until both fields are filled.
- **Landed.** Expect pathname `/main/violations` (the requested page) and `h1:contains("violations")`. `captureState('02-after-login')`.
- **User menu.** Click `button[aria-label="User menu"]`. `[data-testid="menu-user-name"]` is visible. `captureState('03-user-menu')`.
- **Logout.** Click `button:contains("Log out")` (fires `POST /sso/session/logout`). Expect pathname `/login`. `captureState('04-logged-out')`.

## Gotchas

- The form's labels are plain divs with no `htmlFor`. `findByLabelText` fails; use `input[name=...]`.
- `cypress/integration/auth.test.js` is `describe.skip` and fully stubbed. Do not copy its selectors without checking the screenshot.
- Typing the password with logging on writes it into `cypress.log` and the video's command log. Always pass `{ log: false }`.
- The video still shows masked password dots. Do not attach login videos from a non-throwaway Central.
- Visiting `/login` while signed in shows a `Go to Dashboard` link instead of the form.
