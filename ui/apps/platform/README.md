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

### Feature flags

#### Add a feature flag to frontend code

Given a feature flag environment variable `"ROX_WHATEVER"` in pkg/features/list.go:

```go
	// Whatever enables something wherever.
	Whatever = registerFeature("Enable Whatever wherever", "ROX_WHATEVER", false)
```

1. Add `'ROX_WHATEVER'` to string enumeration type `FeatureFlagEnvVar` in ui/apps/platform/src/types/featureFlag.ts

2. To include frontend code when the feature flag is enabled, do any of the following:

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

4. To turn on a feature flag for continuous integration in **branch** and **master** builds:

    * Add `ci_export ROX_WHATEVER "${ROX_WHATEVER:-true}"` to `export_test_environment` function in tests/e2e/lib.sh

    The value of feature flags for **demo** and **release** builds is in pkg/features/list.go 

5. To turn on a feature flag for **local deployment**, do either or both of the following:

    * Before you enter `yarn deploy-local` command in **ui** directory, enter `export ROX_WHATEVER=true` command
    * Before you enter `yarn cypress-open` command in **ui/apps/platform** directory, enter `export CYPRESS_ROX_WHATEVER=true` command

#### Delete a feature flag from frontend code

Given a feature flag environment variable `"ROX_WHATEVER"` in pkg/features/list.go:

```go
	// Whatever enables something wherever.
	Whatever = registerFeature("Enable Whatever wherever", "ROX_WHATEVER", false)
```

1. Delete `'ROX_WHATEVER'` from string enumeration type `FeatureFlagEnvVar` in ui/apps/platform/src/types/featureFlag.ts

2. In frontend code, do any of the following:

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

3. In integration tests:

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

4. For continuous integration:

    * Delete `ci_export ROX_WHATEVER "${ROX_WHATEVER:-true}"` from `export_test_environment` function in tests/e2e/lib.sh

### API

#### Frontend request and response types

Given a change to a backend data structure:

1. Create or edit a corresponding file with camel case name in the ui/apps/platform/src/types folder:

    * whateverService.proto.ts for request or response in proto/api/v1/whatever_service.proto
    * whatever.proto.ts for storage/whatever.proto
    * whatEver.proto.ts for storage/what_ever.proto

2. For type and property names:

    * If a backend type is declared within the scope of a parent type or has a generic name, you might prefix the frontend type to prevent name collisions or increase specificity, for example: `ContainerSecurityContext` or `PolicySeverity`
    * If a backend property has underscore case like `service_id` the frontend property has camelcase like `serviceId`

3. For property types, follow existing examples of the correspondence between backend proto files and frontend TypeScript files

#### Frontend REST services

Given a change to a backend whatever_service.proto file in the proto/api/v1 folder:

1. Create or edit a corresponding file in the ui/apps/platform/src/services folder:

    * Classic naming convention is title case and plural: WhateversService.ts
    * Consistent naming convention is camel case and singular: whateverService.ts

2. For request and response types:

    * Import from type files (described in the previous section)
    * For function arguments and return types, follow existing examples of the correspondence between backend services and frontend functions
