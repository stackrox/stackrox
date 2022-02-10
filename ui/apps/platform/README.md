# StackRox Kubernetes Security Platform Web Application (UI)

Single-page application (SPA) for StackRox Kubernetes Security Platform. This
application was bootstrapped with
[Create React App](https://github.com/facebookincubator/create-react-app).

## Development

Refer to the parent [README.md](../../README.md) for setting up dev env for the
whole parent monorepo.

The documentation below is only specific to this package.

### Testing

#### Unit Tests

Use `yarn test` to run all unit tests and show test coverage. To run tests and
continuously watch for changes use `yarn test-watch`.

#### End-to-end Tests (Cypress)

To bring up [Cypress](https://www.cypress.io/) UI use `yarn cypress-open`. To
run all end-to-end tests in a headless mode use `yarn test-e2e-local`. To run
one test suite specifically in headless mode, use
`yarn cypress-spec <spec-file>`.

#### End-to-end Tests for Demo Automation (Cypress)

To bring up [Cypress](https://www.cypress.io/) UI use `yarn cypress-demo-open`.
To run all end-to-end tests in a headless mode use `yarn test-e2e-demo-local`.
Make sure that `CYPRESS_DEMO_PASSWORD` is set with the Central Password for the
Demo Setup.
