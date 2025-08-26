# StackRox Kubernetes Security Platform Web Application (UI)

Single-page application (SPA) for StackRox Kubernetes Security Platform. This
application was bootstrapped with
[Create React App](https://github.com/facebookincubator/create-react-app).

## Development

Refer to the parent [README.md](../../README.md) for setting up dev env for the
whole parent monorepo.

The documentation below is only specific to this package.

### Running the development server

To start the local development server in watch mode, run `npm run start`.

The behavior of `npm run start` can be changed with the following environment variables:

#### UI_START_TARGET

`UI_START_TARGET` will set the default endpoint that API requests are forwarded to from
the UI. By default the UI will forward API requests to `https://localhost:8000`.

```sh
UI_START_TARGET=https://8.8.8.8:443 npm run start
```

will start the development server
and forward all API requests to `https://8.8.8.8:443`. _Note that the protocol (https) is required to
be set for this option._

#### UI_CUSTOM_PROXIES

`UI_CUSTOM_PROXIES` will override the endpoint that API requests are forwarded to for specific
endpoints that you define. The value of `UI_CUSTOM_PROXIES` is a comma separated list of URL and
remote endpoint pairs. ('url1,endpoint1,url2,endpoint2...') This value can be combined with
`UI_START_TARGET` and will take precedence over the latter for the endpoints that are defined.

```sh
UI_CUSTOM_PROXIES='/v1/newapi,https://localhost:3030,/sso,https://localhost:9000' npm run start
```

will forward any requests from `/v1/newapi` to `https://localhost:3030` and from `/sso` to `https://localhost:9000`. All
other requests will be forwarded to the default location of `https://localhost:8000`.

```sh
UI_START_TARGET='https://8.8.8.8:443' UI_CUSTOM_PROXIES='/v1/newapi,https://localhost:3030' npm run start
```

will forward any request from `/v1/newapi` to `https://localhost:3030` and all other requests will be forwarded to
the value of `UI_START_TARGET`: `https://8.8.8.8:443`.

### Linting

If **stackrox/ui** is your workspace root folder, you can create or edit stackrox/ui/.vscode/settings.json file to add the following properties:

```json
{
    "eslint.workingDirectories": ["apps/platform"]
}
```

### Running as an OpenShift Console plugin

A subset of the code can also be embedded in the OpenShift Console UI using webpack federated modules. The build tooling for
this is completely separate from the build tooling for the standalone version, but both versions share a large amount of application code.

**Prerequisites**

For additional reference, see the [console-plugin-template](https://github.com/openshift/console-plugin-template?tab=readme-ov-file#development) repository.

You need

1. A running OpenShift cluster and kubeconfig available in order to run the plugin.
2. `podman` or `docker`
3. `oc`

**Architecture**

A plugin development environment has a handful of network components that can be configured in a variety of ways to talk to one another:

1. A running OpenShift installation
2. A local OpenShift console container
3. A local development server for the plugin
4. An API server for the `central` component

> When using the plugin via the console web UI, all requests for data to `central` will first be proxied through the console backend. The console _may_ inject authorization
> headers into each request, overwriting any `Authorization` header that is initially sent in the request. The token injected into this header is the current OpenShift user's opaque
> access token, and must be handled correctly by the backend for this to succeed. **As of today, `central` is unable to do this and all requests will fail.** In other words, the default
> behavior of the commands in the **Running the plugin** section below will not work without additional configuration. After the backend is developed, we should be able to revert
> to the bare commands and removed the need for overrides.

**Running the plugin**

First, we need to generate an API token against the `central` instance that is intended to be used for development. If `central` was installed through the local `deploy` scripts and the current admin password is available in `../../../deploy/k8s/central-deploy/password` this can be done with the included script:

```sh
UI_API_TOKEN_NAME=ocp-console-dev UI_API_TOKEN_ROLE=Analyst ./scripts/get-auth-token.sh
```

Otherwise, generate a new API token via the UI and save it somewhere secure.

Next, start the webpack dev server to make the plugin configuration files and js bundles available. Pass the token obtained above
to the command:

```sh
# In a new terminal
ACS_CONSOLE_DEV_TOKEN=<your-token> npm run start:ocp-plugin
```

This will run a webpack development server on http://localhost:9001 serving the plugin files and ensure the UI passes the token in the `Authorization` header of API requests.

Next, start a local development version of the console in another terminal:

```sh
# With kubectx pointing to your openshift cluster, login via web browser
oc login --web

# Run the following script to start a local instance of the OCP console as a bridge for plugin development. Disable the console
# token injection to ensure our API token from above is passed in all requests.
ACS_INJECT_OCP_AUTH_TOKEN=false ./scripts/start-ocp-console.sh

# By default, requests will be proxied to the detected central running in the cluster. If you wish to use an alternative central, instead run:
ACS_INJECT_OCP_AUTH_TOKEN=false ACS_API_SERVICE_URL=<central-service-url> ./scripts/start-ocp-console.sh
```

This will start the console on http://localhost:9000 with user authentication disabled; you will be logged in automatically as kubeadmin using the token retrieved via `oc login --web` above. With token injection disabled, all requests will include the central API token specified when running the plugin server. Visit http://localhost:9000 in
your browser to develop and test the plugin.

_Note: At this time https is not supported for local plugin development._

### Testing

#### Unit Tests

Use `npm run test` to run all unit tests and show test coverage. To run tests and
continuously watch for changes use `npm run test-watch`.

#### End-to-end Tests (Cypress)

To bring up [Cypress](https://www.cypress.io/) UI use `npm run cypress-open`. To
run all end-to-end tests in a headless mode use `npm run test-e2e-local`. To run
one test suite specifically in headless mode, use
`npm run cypress-spec <spec-file>`.

#### End-to-end Tests (Cypress targeting console plugin)

To run Cypress against the OCP console for dynamic plugin tests, there are two scenarios that are supported.

1. Running against a locally deployed version of the development console with bridge authentication off

```sh
# If necessary, export the target URL
export OPENSHIFT_CONSOLE_ENDPOINT=<url-to-web-console-ui>
# Set ORCHESTRATOR_FLAVOR, which is typically only available in CI
export ORCHESTRATOR_FLAVOR='openshift'
# Runs Cypress OCP tests ignoring authentication
OCP_BRIDGE_AUTH_DISABLED=true npm run cypress-open:ocp
```

2. Running against a deployed version of the console with username/password credentials

```sh
# If necessary, export the target URL
export OPENSHIFT_CONSOLE_ENDPOINT=<url-to-web-console-ui>
# Set ORCHESTRATOR_FLAVOR, which is typically only available in CI
export ORCHESTRATOR_FLAVOR='openshift'
# export credentials
export OPENSHIFT_CONSOLE_USERNAME='kubeadmin'
export OPENSHIFT_CONSOLE_PASSWORD=<password>

# Runs Cypress OCP tests with a session initialization step
npm run cypress-open:ocp
```

### Feature flags

#### Add a feature flag to frontend code

Given a feature flag environment variable `"ROX_WHATEVER"` in pkg/features/list.go:

```go
	// Whatever enables something wherever.
	Whatever = registerFeature("Enable Whatever wherever", "ROX_WHATEVER", false)
```

1. Add `'ROX_WHATEVER'` to string enumeration type `FeatureFlagEnvVar` in ui/apps/platform/src/types/featureFlag.ts

2. To include frontend code when the feature flag is enabled, do any of the following:
    - Add `featureFlagDependency: 'ROX_WHATEVER'` property in any of the following:
        - for **integration tile** in ui/apps/platform/src/Containers/Integrations/utils/integrationsList.ts
        - for **integration table column** in ui/apps/platform/src/Containers/Integrations/utils/tableColumnDescriptor.ts
        - for **policy criterion** in ui/apps/platform/src/Containers/Policies/Wizard/Step3/policyCriteriaDescriptors.tsx

    - Use `useFeatureFlags` hook in a React component:
        - Add `import useFeatureFlags from 'hooks/useFeatureFlags';` in component file
        - Add `const { isFeatureFlagEnabled } = useFeatureFlags();` in component function
        - Add `const isWhateverEnabled = isFeatureFlagEnabled('ROX_WHATEVER');` assignment statement
        - And then, do any of the following:
            - Add `if` statement:
                - For **newer** behavior only when feature flag is **enabled**

                    ```tsx
                    if (isWhateverEnabled) {
                        /* Do whatever */
                    }
                    ```

                - For **older** behavior only when feature flag is **disabled** use negative `!isWhateverEnabled` condition

                - For alternative either/or behavior, add `else` block to `if` statement

            - Add conditional rendering:
                - For **newer** behavior only when feature flag is **enabled**

                    ```tsx
                    {
                        isWhateverEnabled && <Whatever />;
                    }
                    ```

                - For **older** behavior only when feature flag is **disabled** use negative `!isWhateverEnabled` condition

                - For alternative either/or behavior, add **newer** elements and move **older** elements into ternary expression:

                    ```tsx
                    {
                        isWhateverEnabled ? <Whatever /> : <WhateverItHasBeen />;
                    }
                    ```

3. To skip integration tests:
    - Add `import { hasFeatureFlag } from '../…/helpers/features';` in cypress/integration/…/whatever.test.js
    - And then at the beginning of `describe` block do either or both of the following:
        - To skip **older** tests which are not relevant when feature flag is **enabled**

            ```js
            before(function beforeHook() {
                if (hasFeatureFlag('ROX_WHATEVER')) {
                    this.skip();
                }
            });
            ```

        - Skip **newer** tests which are not relevant when feature flag is **disabled**

            ```js
            before(function beforeHook() {
                if (!hasFeatureFlag('ROX_WHATEVER')) {
                    this.skip();
                }
            });
            ```

4. To turn on a feature flag for continuous integration in **branch** and **master** builds:
    - Add `ci_export ROX_WHATEVER "${ROX_WHATEVER:-true}"` to `export_test_environment` function in tests/e2e/lib.sh
    - Add code below to `deploy_central_via_operator` function in tests/e2e/lib.sh

        ```
        customize_envVars+=$'\n      - name: ROX_WHATEVER'
        customize_envVars+=$'\n        value: "true"'
        ```

    The value of feature flags for **demo** and **release** builds is in pkg/features/list.go

5. To turn on a feature flag for **local deployment**, do either or both of the following:
    - Before you enter `npm run deploy-local` command in **ui** directory, enter `export ROX_WHATEVER=true` command
    - Before you enter `npm run cypress-open` command in **ui/apps/platform** directory, enter `export CYPRESS_ROX_WHATEVER=true` command

#### Delete a feature flag from frontend code

Given a feature flag environment variable `"ROX_WHATEVER"` in pkg/features/list.go:

```go
	// Whatever enables something wherever.
	Whatever = registerFeature("Enable Whatever wherever", "ROX_WHATEVER", false)
```

1. Delete `'ROX_WHATEVER'` from string enumeration type `FeatureFlagEnvVar` in ui/apps/platform/src/types/featureFlag.ts

2. In frontend code, do any of the following:
    - Delete `featureFlagDependency: 'ROX_WHATEVER'` property in any of the following:
        - for **integration tile** in ui/apps/platform/src/Containers/Integrations/utils/integrationsList.ts
        - for **integration table column** in ui/apps/platform/src/Containers/Integrations/utils/tableColumnDescriptor.ts
        - for **policy criterion** in ui/apps/platform/src/Containers/Policies/Wizard/Step3/policyCriteriaDescriptors.tsx

    - For `useFeatureFlags` hook in a React component:
        - Delete `import useFeatureFlags from 'hooks/useFeatureFlags';` in component file
        - Delete `const { isFeatureFlagEnabled } = useFeatureFlags();` in component function
        - Delete `const isWhateverEnabled = isFeatureFlagEnabled('ROX_WHATEVER');` assignment statement
        - And then, do any of the following:
            - For `if` statement:
                - For **newer** behavior only when feature flag is **enabled**

                    Replace `if (isWhateverEnabled) {/* Do whatever */}` with `/* Do whatever */`

                - For **older** behavior only when feature flag is **disabled**
                    - Delete `if (!isWhateverEnabled) {/* Do whatever it was */}`

                - For alternative either/or behavior

                    Replace `if (isWhateverEnabled) {/* Do whatever */} else {/* Do whatever it has been */}` with `/* Do whatever */`

            - For conditional rendering:
                - For **newer** behavior only when feature flag is **enabled**

                    Replace `{isWhateverEnabled && (<Whatever />)}` with `<Whatever />`

                - For **older** behavior only when feature flag is **disabled**

                    Delete `{!isWhateverEnabled && (<WhateverItHasBeen />)}`

                - For alternative either/or behavior

                    Replace `{isWhateverEnabled ? (<Whatever />) : (<WhateverItHasBeen />)}` with `<Whatever />`

3. In integration tests:
    - Delete `import { hasFeatureFlag } from '../…/helpers/features';` in cypress/integration/…/whatever.test.js
    - And then at the beginning of `describe` block do either or both of the following:
        - For **older** tests which were not relevant when feature flag is **enabled**

            Delete obsolete `describe` block (or possibly entire test file) which has the following:

            ```js
            before(function beforeHook() {
                if (hasFeatureFlag('ROX_WHATEVER')) {
                    this.skip();
                }
            });
            ```

        - For **newer** tests which were not relevant when feature flag is **disabled**

            To run tests unconditionally, delete the following:

            ```js
            before(function beforeHook() {
                if (!hasFeatureFlag('ROX_WHATEVER')) {
                    this.skip();
                }
            });
            ```

4. For continuous integration:
    - Delete `ci_export ROX_WHATEVER "${ROX_WHATEVER:-true}"` from `export_test_environment` function in tests/e2e/lib.sh
    - Delete code below from `deploy_central_via_operator` function in tests/e2e/lib.sh

        ```
        customize_envVars+=$'\n      - name: ROX_WHATEVER'
        customize_envVars+=$'\n        value: "true"'
        ```

### Routes

#### Add a route

Read and obey comments to add strings or properties **in alphabetical order to minimize merge conflicts**.

1. Edit ui/apps/platform/src/routePaths.ts file.
    - Add a path **without** params for link from sidebar navigation and, if needed, path **with** param for the `Route` element.
        - Use a **plural** noun for something like **clusters**.
        - Use a **singular** noun for something like **compliance**.

        ```ts
        export const whateversBasePath = `${mainPath}/whatevers`;
        export const whateversPathWithParam = `${whateversBasePath}/:id?`;
        ```

    - Add a string to `RouteKey` type.

        ```ts
        | 'whatevers'
        ```

    - Add a property to `routeRequirementsMap` object.

        Specify a feature flag during development of a new route.

        Specify minimum resource requirements. Component files might have conditional rendering for additional resources.

        ```ts
        'whatevers': {
            featureFlagDependency: ['ROX_WHATEVERS'],
            resourceAccessRequirements: everyResource(['Whichever']),
        },
        ```

2. Edit ui/apps/platform/src/Containers/MainPage/Body.tsx file.
    - Import the path for the `Route` element.

        ```ts
        whateversPathWithParam,
        ```

    - Add a property to `routeComponentMap` object.

        Specify the path to the root component of the asynchronously-loaded bundle file for the route (see step 4).

        ```ts
        'whatevers': {
            component: asyncComponent(
                () => import('Containers/Whatevers/WhateversRoute')
            ),
            path: whateversPathWithParam,
        },
        ```

3. Edit ui/apps/platform/src/Containers/MainPage/Sidebar/NavigationSidebar.tsx file, **if** the route has a link.
    - Import a path **without params**.

    ```ts
    whateversBasePath,
    ```

    - Add a child item for the link in the `navDescriptions` array.

    ```ts
    {
        type: 'child',
        content: 'Whatevers',
        path: whateversBasePath,
        routeKey: 'whatevers',
    },
    ```

4. Add a folder and root component file (see step 2).

    For example: ui/apps/platform/src/Containers/Whatevers/WhateversRoute.tsx

### API

#### Frontend request and response types

Given a change to a backend data structure:

1. Create or edit a corresponding file with camel case name in the ui/apps/platform/src/types folder:
    - whateverService.proto.ts for request or response in proto/api/v1/whatever_service.proto
    - whatever.proto.ts for storage/whatever.proto
    - whatEver.proto.ts for storage/what_ever.proto

2. For type and property names:
    - If a backend type is declared within the scope of a parent type or has a generic name, you might prefix the frontend type to prevent name collisions or increase specificity, for example: `ContainerSecurityContext` or `PolicySeverity`
    - If a backend property has underscore case like `service_id` the frontend property has camelcase like `serviceId`

3. For property types, follow existing examples of the correspondence between backend proto files and frontend TypeScript files

#### Frontend REST services

Given a change to a backend whatever_service.proto file in the proto/api/v1 folder:

1. Create or edit a corresponding file in the ui/apps/platform/src/services folder:
    - Classic naming convention is title case and plural: WhateversService.ts
    - Consistent naming convention is camel case and singular: whateverService.ts

2. For request and response types:
    - Import from type files (described in the previous section)
    - For function arguments and return types, follow existing examples of the correspondence between backend services and frontend functions
