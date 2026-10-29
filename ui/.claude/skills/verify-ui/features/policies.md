# Policies

Policy Management lists system and custom policies, opens a policy's detail page (overview, behavior, criteria), and creates, edits, clones, enables, disables, or deletes policies through a multi-step wizard.

## Sub-features

- `policies-nav` opens the table from Platform Configuration.
- `policies-detail` opens a policy and shows its overview and behavior.
- `policies-create` walks the wizard (Details, Lifecycle, Rules, Resources, [Filters], Actions, Review) and saves.
- `policies-status` enables or disables a policy from the row actions.
- `policies-delete` deletes a custom policy.

## How to get to it (user POV)

- Left nav > `Platform Configuration` > `Policy Management` (`/main/policy-management/policies`).
- `Create policy` button on the table page (`?action=create`).
- Row kebab or the detail page `Actions` menu for edit, clone, status, and delete.

## Driving it with verify-ui

Preconditions:

- `doctor.sh` healthy.
- Import from `cypress/integration/policies/Policies.helpers.ts` (`visitPoliciesFromLeftNav`, `deletePolicyIfExists`) and `cypress/integration/policies/policyWizard.helpers.ts` (`startPolicyWizard`, `selectCategory`, `clickNext`, `clickSave`, `assertStepHeading`, `verifyPolicyInTable`). `getInputByLabel` comes from `cypress/helpers/formHelpers.js`.
- Unique name: `const name = \`verify-ui-${Date.now()}\``, and `after(() => deletePolicyIfExists(name))`.

- **Open from nav.** `visitPoliciesFromLeftNav()`. Expect `h1:contains("Policy management")` and `[data-testid="policies-table"] tbody tr` length > 0. `captureState('01-policies')`.
- **Open detail.** Click `td[data-label="Policy"] a:contains("Latest tag")` (a default system policy) inside `interactAndWaitForResponses(cb, { policy: { method: 'GET', url: '/v1/policies/*' } })`. Expect `h1:contains("Latest tag")`, `h2:contains("Policy overview")`, and `h2:contains("Policy behavior")`. `captureState('02-policy-detail')`.
- **Start wizard.** Back on the table, `startPolicyWizard()` (clicks `button:contains("Create policy")`). Expect `h1:contains("Create policy")`. `captureState('03-wizard')`.
- **Details step.** `getInputByLabel('Name').type(name)`, `cy.contains('label', 'Low').click()`, `selectCategory('Privileges')` (Categories is required; Next stays disabled without it), then `clickNext()`. `assertStepHeading('Lifecycle')`.
- **Lifecycle step.** `cy.contains('label', 'Build').click()`, then `clickNext()`. `assertStepHeading('Rules')`. Add one rule criterion with the drag helpers (see `policyCreateWorkflow.test.ts`), or use a lifecycle whose rule step allows advancing.
- **Finish.** `clickNext()` through Resources, Filters (if shown), and Actions until `assertStepHeading('Review policy')`. Then `clickSave()`. `captureState('04-saved')`.
- **Persisted.** `verifyPolicyInTable(name)` (reloads the table and finds the row). `captureState('05-in-table')`.

## Gotchas

- Creating writes to the real Central. Always use a unique name and `deletePolicyIfExists` in `after()`.
- The Filters step appears only with `ROX_EVALUATION_FILTER` and `ROX_INIT_CONTAINER_SUPPORT` on. drive.sh mirrors Central flags into Cypress env, so `hasFeatureFlag` matches the app.
- Rule criteria use drag and drop via `DndSimulatorDataTransfer`. Copy the exact calls from `policyCreateWorkflow.test.ts`.
- Externally managed (CRD) policies show a warning alert and cannot be edited.
- `visitPolicies` sets `requestTimeout` to 10000; large policy lists can still be slow on a cold Central.
