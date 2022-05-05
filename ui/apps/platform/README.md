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

### Feature flags

#### Add a feature flag to frontend code

Given a feature flag environment variable `"ROX_WHATEVER"` in pkg/features/list.go:

1. Add `'ROX_WHATEVER'` to string enumeration type `FeatureFlagEnvVar` in ui/apps/platform/src/types/featureFlag.ts

2. To include frontend code when the flag is enabled, do any of the following:

    * Add `featureFlagDependency: 'ROX_WHATEVER'` property in any of the following:
        * for **integration tile** in ui/apps/platform/src/Containers/Integrations/utils/integrationsList.ts
        * for **integration table column** in ui/apps/platform/src/Containers/Integrations/utils/tableColumnDescriptor.ts
        * for **policy criterion** in ui/apps/platform/src/Containers/Policies/Wizard/Step3/policyCriteriaDescriptors.tsx

    * Use `useFeatureFlags` hook in a React component:
        * Add `import useFeatureFlags from 'hooks/useFeatureFlags';` in component file
        * Add `const { isFeatureFlagEnabled } = useFeatureFlags();` in component function
        * Add `const isWhateverEnabled = isFeatureFlagEnabled('ROX_WHATEVER');` assignment statement
        * And then, do any of the following:

            * Add `if` statement:

                * For **newer** behavior only when feature flag is **enabled**

                    ```tsx
                    if (isWhateverEnabled) {
                        /* Do whatever */
                    }
                    ```

                * For **older** behavior only when feature flag is **disabled** use negative `!isWhateverEnabled` condition

                * For alternative either/or behavior, add `else` block to `if` statement

            * Add conditional rendering:

                * For **newer** behavior only when feature flag is **enabled**

                    ```tsx
                    {isWhateverEnabled && (
                        <Whatever />
                    )}
                    ```

                * For **older** behavior only when feature flag is **disabled** use negative `!isWhateverEnabled` condition

                * For alternative either/or behavior, add **newer** elements and move **older** elements into ternary expression:

                    ```tsx
                    {isWhateverEnabled ? (
                        <Whatever />
                    ) : (
                        <WhateverItHasBeen />
                    )}
                    ```

3. To skip integration tests:

    * Add `import { hasFeatureFlag } from '../…/helpers/features';` in cypress/integration/…/whatever.test.js
    * And then at the beginning of `describe` block do either or both of the following:

        * To skip **older** tests which are not relevant when feature flag is **enabled**

            ```js
            before(function beforeHook() {
                if (hasFeatureFlag('ROX_WHATEVER')) {
                    this.skip();
                }
            });
            ```

        * Skip **newer** tests which are not relevant when feature flag is **disabled**

            ```js
            before(function beforeHook() {
                if (!hasFeatureFlag('ROX_WHATEVER')) {
                    this.skip();
                }
            });
            ```

4. To turn on a feature flag for **local deployment**, do either or both of the following:
    * Before you enter `yarn deploy-local` command in **ui** directory, enter `export ROX_WHATEVER=true` command
    * Before you enter `yarn cypress-open` command in **ui/apps/platform** directory, enter `export CYPRESS_ROX_WHATEVER=true` command

#### Delete a feature flag from frontend code

Given a feature flag environment variable `"ROX_WHATEVER"` in pkg/features/list.go:

1. Delete `'ROX_WHATEVER'` from string enumeration type `FeatureFlagEnvVar` in ui/apps/platform/src/types/featureFlag.ts

2. Do any of the following:

    * Delete `featureFlagDependency: 'ROX_WHATEVER'` property in any of the following:
        * for **integration tile** in ui/apps/platform/src/Containers/Integrations/utils/integrationsList.ts
        * for **integration table column** in ui/apps/platform/src/Containers/Integrations/utils/tableColumnDescriptor.ts
        * for **policy criterion** in ui/apps/platform/src/Containers/Policies/Wizard/Step3/policyCriteriaDescriptors.tsx

    * For `useFeatureFlags` hook in a React component:
        * Delete `import useFeatureFlags from 'hooks/useFeatureFlags';` in component file
        * Delete `const { isFeatureFlagEnabled } = useFeatureFlags();` in component function
        * Delete `const isWhateverEnabled = isFeatureFlagEnabled('ROX_WHATEVER');` assignment statement
        * And then, do any of the following:

            * For `if` statement:

                * For **newer** behavior only when feature flag is **enabled**

                    Replace `if (isWhateverEnabled) {/* Do whatever */}` with `/* Do whatever */`

                * For **older** behavior only when feature flag is **disabled**

                    * Delete `if (!isWhateverEnabled) {/* Do whatever it was */}`

                * For alternative either/or behavior

                    Replace `if (isWhateverEnabled) {/* Do whatever */} else {/* Do whatever it has been */}` with `/* Do whatever */`

            * For conditional rendering:

                * For **newer** behavior only when feature flag is **enabled**

                    Replace `{isWhateverEnabled && (<Whatever />)}` with `<Whatever />`

                * For **older** behavior only when feature flag is **disabled**

                    Delete `{!isWhateverEnabled && (<WhateverItHasBeen />)}`

                * For alternative either/or behavior

                    Replace `{isWhateverEnabled ? (<Whatever />) : (<WhateverItHasBeen />)}` with `<Whatever />`

3. For integration tests:

    * Delete `import { hasFeatureFlag } from '../…/helpers/features';` in cypress/integration/…/whatever.test.js
    * And then at the beginning of `describe` block do either or both of the following:

        * For **older** tests which were not relevant when feature flag is **enabled**

            Delete obsolete `describe` block (or possibly entire test file) which has the following:

            ```js
            before(function beforeHook() {
                if (hasFeatureFlag('ROX_WHATEVER')) {
                    this.skip();
                }
            });
            ```

        * For **newer** tests which were not relevant when feature flag is **disabled**

            To run tests unconditionally, delete the following:

            ```js
            before(function beforeHook() {
                if (!hasFeatureFlag('ROX_WHATEVER')) {
                    this.skip();
                }
            });
            ```
