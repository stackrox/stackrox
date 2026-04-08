# StackRox Kubernetes Security Platform Web Application (UI)

Single-page application (SPA) for StackRox Kubernetes Security Platform. Built with
React 18, TypeScript, and Vite.

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

A subset of the code can also be embedded in the OpenShift Console UI using
[webpack federated modules](https://webpack.js.org/concepts/module-federation/).
The build tooling for this is completely separate from the build tooling for the
standalone version, but both versions share a large amount of application code.

For additional reference, see the
[OpenShift Console Plugin SDK docs](https://github.com/openshift/console/tree/main/frontend/packages/console-dynamic-plugin-sdk)
and the [console-plugin-template](https://github.com/openshift/console-plugin-template?tab=readme-ov-file#development)
repository.

#### How the plugin works

OpenShift Console uses webpack Module Federation to load plugins at runtime. The
key concepts are:

- **Host / Remote**: The console is the "host" application. Our plugin is a
  "remote" that exposes named modules (React components) via a manifest.
- **Shared singletons**: Certain dependencies (React, Redux, PatternFly
  Topology, react-router, etc.) are provided by the console as singletons.
  Plugins use the console's copy at runtime -- they cannot bundle their own.
  See [Compatibility](#compatibility) for the full list and implications.
- **ConsolePlugin CRD**: In production, the console discovers plugins via a
  `ConsolePlugin` custom resource that points to the Service and base path
  serving the plugin manifest and bundles.

The plugin webpack config (`webpack.ocp-plugin.config.js`) uses the
`ConsoleRemotePlugin` from `@openshift-console/dynamic-plugin-sdk-webpack`,
which wraps Module Federation with console-specific conventions. It generates
the manifest, declares shared modules, and registers console
[extensions](https://github.com/openshift/console/tree/main/frontend/packages/console-dynamic-plugin-sdk/docs)
(routes, nav items, resource tabs, context providers).

#### Authentication and request flow

The plugin never talks to Central directly. All API requests flow through the
console's proxy and `sensor-proxy`, which handles authentication and
authorization using the user's existing OpenShift session.

```txt
Browser (OpenShift Console)
  |
  |  Plugin component calls axios.get('/v1/...')
  |
  v
consoleFetchAxiosAdapter (src/ConsolePlugin/consoleFetchAxiosAdapter.ts)
  |  Overrides axios default adapter
  |  Injects ACS-AUTH-NAMESPACE-SCOPE header (active namespace)
  |  Calls consoleFetch() from SDK (adds user's OCP bearer token + CSRF)
  |
  v
Console Proxy
  |  Route: /api/proxy/plugin/advanced-cluster-security/api-service/...
  |  ConsolePlugin CRD proxy config: authorization: UserToken
  |  Console injects the user's bearer token into the upstream request
  |
  v
sensor-proxy (in-cluster Service, port 443)
  |  Validates OCP token against Kubernetes RBAC
  |  Applies ACS RBAC based on namespace scope header
  |  Forwards authenticated request to Central
  |
  v
Central
  |  Generates dynamic access scope
  |  Processes request with full auth context
  |  Returns data filtered by user permissions
```

#### Code structure

The plugin-specific code lives in `src/ConsolePlugin/`. Everything else under
`src/` (providers, services, hooks, components in `Containers/`) is shared
between the standalone UI and the plugin.

```txt
src/
├── index.tsx                         # Standalone UI entry point
├── ConsolePlugin/                    # Plugin-specific code and wrappers
│   ├── PluginProvider.tsx            # Context provider: sets up axios adapter,
│   │                                 #   wraps shared providers (auth, flags, etc.)
│   ├── consoleFetchAxiosAdapter.ts   # Bridges axios -> consoleFetch (SDK)
│   ├── ScopeContext.tsx              # Tracks active namespace from console
│   ├── PluginContent.tsx             # Permission gate wrapper
│   ├── hooks/                        # Plugin-specific hooks
│   │   ├── useAnalyticsPageView.ts
│   │   ├── useDefaultWorkloadCveViewContext.ts
│   │   └── useWorkloadId.ts
│   ├── Components/                  # Plugin-specific general UI components
│   │
│   │   # Exposed modules (entry points registered as console extensions):
│   ├── SecurityVulnerabilitiesPage/  # Top-level /acs/security/vulnerabilities route
│   ├── CveDetailPage/               # CVE detail route
│   ├── ImageDetailPage/             # Image detail route
│   ├── WorkloadSecurityTab/         # "Security" tab on Deployment, StatefulSet, etc.
│   ├── AdministrationNamespaceSecurityTab/  # "Security" tab on Namespace
│   └── ProjectSecurityTab/          # "Security" tab on Project
│
├── Containers/Vulnerabilities/       # Vuln Management page components - shared
├── providers/                        # Shared context providers
├── services/                         # Shared API service functions
└── hooks/                            # Shared hooks
```

Each exposed module is a thin wrapper that imports shared
components from `Containers/` and adds plugin-specific concerns like namespace
scoping and analytics tracking.

#### Adding a new plugin extension

To add a new UI surface to the console plugin (e.g. a new tab on a Kubernetes
resource, or a new route), follow these steps. For the full list of available
extension types, see the
[Console SDK extension docs](https://github.com/openshift/console/tree/main/frontend/packages/console-dynamic-plugin-sdk/docs).

1. **Create the entry point component** in `src/ConsolePlugin/YourExtension/YourExtension.tsx`.

    Keep it minimal -- import shared components and add only what's plugin-specific.
    Use existing entry points as templates. For example, a resource tab:

    ```tsx
    // src/ConsolePlugin/MyResourceSecurityTab/MyResourceSecurityTab.tsx
    import { useParams } from 'react-router-dom-v5-compat';

    import SomeSharedComponent from 'Containers/SomeArea/SomeSharedComponent';
    import { useAnalyticsPageView } from '../hooks/useAnalyticsPageView';

    export function MyResourceSecurityTab() {
        useAnalyticsPageView();
        const { ns, name } = useParams();

        return <SomeSharedComponent namespace={ns} name={name} />;
    }
    ```

2. **Register the exposed module** in `webpack.ocp-plugin.config.js` under
   `pluginMetadata.exposedModules`:

    ```js
    exposedModules: {
        // ...existing modules
        MyResourceSecurityTab: './ConsolePlugin/MyResourceSecurityTab/MyResourceSecurityTab',
    },
    ```

3. **Add the console extension** in the `extensions` array in the same file.

    For a horizontal nav tab on a Kubernetes resource:

    ```js
    {
        type: 'console.tab/horizontalNav',
        properties: {
            model: {
                group: 'apps',
                kind: 'MyResource',
                version: 'v1',
            },
            page: {
                name: 'Security',
                href: 'security',
            },
            component: { $codeRef: 'MyResourceSecurityTab.MyResourceSecurityTab' },
        },
    },
    ```

    For a new route:

    ```js
    {
        type: 'console.page/route',
        properties: {
            exact: true,
            path: '/acs/my-area/my-page',
            component: { $codeRef: 'MyPage.MyPage' },
        },
    },
    ```

4. **Test it** by running the plugin dev environment (see [Running the plugin](#running-the-plugin)
   below) and navigating to the resource or route in the console.

#### Compatibility

The plugin's runtime environment is controlled by the OpenShift Console, not by
us. The console provides a set of
[shared singleton modules](https://github.com/openshift/console/blob/release-4.19/frontend/packages/console-dynamic-plugin-sdk/src/shared-modules/shared-modules-meta.ts)
that plugins **must** use -- you cannot bundle your own copy of these libraries.
At runtime, the console's version is what executes, regardless of what version
is in our `package.json`.

The shared modules ([shared-modules-meta.ts](https://github.com/openshift/console/blob/main/frontend/packages/console-dynamic-plugin-sdk/src/shared-modules/shared-modules-meta.ts)) (as of console 4.19) are:

- `react` / `react-dom`
- `react-redux`
- `react-router`
- `react-router-dom`
- `react-router-dom-v5-compat`
- `react-i18next`
- `redux`
- `redux-thunk`
- `@openshift-console/dynamic-plugin-sdk`
- `@openshift-console/dynamic-plugin-sdk-internal`
- `@patternfly/react-topology`

All are singletons with no fallback allowed.

Libraries **not** in this list (e.g. `@patternfly/react-core`,
`@patternfly/react-table`, `@patternfly/react-icons`, `axios`, `@apollo/client`)
are bundled in our plugin and can be versioned independently.

**Note that although _we_ provide `@patterfly/react-core`, the console plugin build strips out PatternFlyCSS.
This means that although we do ship the PatternFly runtime code, we are still limited to the styles provided
by the console.**

**What this means in practice:**

- **React version**: Console 4.19 ships React 17. Our `package.json` declares
  React 18, but the plugin runs on React 17 at runtime. Avoid React 18-only
  APIs (`useId`, `useDeferredValue`, `useTransition`, `createRoot`, automatic
  batching) in any code path reachable from the plugin.
- **react-router**: Console 4.19 ships react-router v5. We use
  `react-router-dom-v5-compat` for v6-style APIs (`useParams`, `useNavigate`).
  Note that both `react-router-dom` and `react-router-dom-v5-compat` are
  deprecated in newer console versions in favor of `react-router` (v7+).
- **PatternFly**: Non-shared PF packages (react-core, react-table, etc.) are
  bundled by us, so minor version differences are fine. However, large version
  gaps between our bundled PF and the console's PF can cause visual
  inconsistencies (spacing, colors, component behavior).
  major version bumps that will require migration work when we target newer
  console releases.

Our webpack config declares `dependencies: { '@console/pluginAPI': '>=4.19.0' }`,
which means the console will only load our plugin if its API version satisfies
that constraint.

#### Prerequisites

You need:

1. A running OpenShift cluster and kubeconfig available in order to run the plugin.
2. `podman` or `docker`
3. `oc`

#### Architecture

A plugin development environment has the following network components:

1. A running OpenShift installation with StackRox secured cluster services installed
2. A local OpenShift console container
3. A local development server for the plugin
4. An exposed `sensor-proxy` service via LoadBalancer

The plugin uses OpenShift user authentication and proxies all API requests through the `sensor-proxy` service, which handles authentication/authorization and forwards requests to Central. This matches the production flow where the console plugin communicates through sensor-proxy rather than directly to Central.

#### Running the plugin

First, start the webpack dev server to make the plugin configuration files and js bundles available:

```sh
# In a new terminal
npm run start:ocp-plugin
```

This will run a webpack development server on http://localhost:9001 serving the plugin files.

Next, start a local development version of the console in another terminal:

**Note: running the below `./scripts/start-ocp-console.sh` script will create a LoadBalancer that exposes `sensor-proxy` to the internet. Ensure you are only connected to a development cluster before proceeding.**


```sh
# With kubectx pointing to your OpenShift cluster, login via web browser
oc login --web

# Run the following script to start a local instance of the OCP console.
# This will automatically:
# - Expose sensor-proxy via a LoadBalancer with NetworkPolicy
# - Configure the console to use the sensor-proxy endpoint
# - Clean up resources when the defined expiration time has elapsed
./scripts/start-ocp-console.sh
```

This will start the console on http://localhost:9000 with user authentication disabled; you will be logged in automatically using the token retrieved via `oc login --web` above. The script handles all backend connectivity automatically. Visit http://localhost:9000 in your browser to develop and test the plugin.

**Configuration options**

The console startup script supports the following environment variables:

- `SENSOR_PROXY_NAMESPACE` - Namespace containing sensor-proxy (default: `stackrox`)
- `SENSOR_PROXY_EXPIRY_HOURS` - Hours until LoadBalancer auto-cleanup (default: `8`)
- `CONSOLE_PORT` - Local console port (default: `9000`)
- `CONSOLE_IMAGE` - Console container image (default: `quay.io/openshift/origin-console:latest`)

Example with custom configuration:

```sh
SENSOR_PROXY_EXPIRY_HOURS=12 ./scripts/start-ocp-console.sh
```

_Note: At this time https is not supported for local plugin development._

### Testing

#### Unit Tests

Use `npm run test` to run all unit tests and show test coverage. To run specific tests,
use `npm run test -- --testNamePattern="TestName"` or `npm run test -- src/path/to/test.test.ts`.

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
export OPENSHIFT_CONSOLE_URL=<url-to-web-console-ui>
# Set ORCHESTRATOR_FLAVOR, which is typically only available in CI
export ORCHESTRATOR_FLAVOR='openshift'
# Runs Cypress OCP tests ignoring authentication
OCP_BRIDGE_AUTH_DISABLED=true npm run cypress-open:ocp
```

2. Running against a deployed version of the console with username/password credentials

```sh
# If necessary, export the target URL
export OPENSHIFT_CONSOLE_URL=<url-to-web-console-ui>
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

    Add string in alphabetical order on its own line to minimize merge conflicts when multiple people add or delete strings.

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
    * Add code below to `deploy_central_via_operator` function in tests/e2e/lib.sh

        ```sh
        customize_envVars+=$'\n      - name: ROX_WHATEVER'
        customize_envVars+=$'\n        value: "true"'
        ```

    The value of feature flags for **demo** and **release** builds is in pkg/features/list.go

5. To turn on a feature flag for **local deployment**, do either or both of the following:

    * Before you enter `npm run deploy-local` command in **ui** directory, enter `export ROX_WHATEVER=true` command
    * Before you enter `npm run cypress-open` command in **ui/apps/platform** directory, enter `export CYPRESS_ROX_WHATEVER=true` command

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
    * Delete code below from `deploy_central_via_operator` function in tests/e2e/lib.sh

        ```sh
        customize_envVars+=$'\n      - name: ROX_WHATEVER'
        customize_envVars+=$'\n        value: "true"'
        ```

### Routes

#### Add a route

Read and obey comments to add strings or properties **in alphabetical order to minimize merge conflicts**.

1. Edit ui/apps/platform/src/routePaths.ts file.

    * Add a path **without** params for link from sidebar navigation and, if needed, path **with** param for the `Route` element.
        * Use a **plural** noun for something like **clusters**.
        * Use a **singular** noun for something like **compliance**.

        ```ts
        export const whateversBasePath = `${mainPath}/whatevers`;
        export const whateversPathWithParam = `${whateversBasePath}/:id?`;
        ```

    * Add a string to `RouteKey` type.

        ```ts
        | 'whatevers'
        ```

    * Add a property to `routeRequirementsMap` object.

        Specify a feature flag during development of a new route.

        Specify minimum resource requirements. Component files might have conditional rendering for additional resources.

        ```ts
        'whatevers': {
            featureFlagDependency: ['ROX_WHATEVERS'],
            resourceAccessRequirements: everyResource(['Whichever']),
        },
        ```

2. Edit ui/apps/platform/src/Containers/MainPage/Body.tsx file.

    * Import the path for the `Route` element.

        ```ts
        whateversPathWithParam,
        ```

    * Add a property to `routeComponentMap` object.

        Specify the path to the root component of the asynchronously-loaded bundle file for the route (see step 4).

        ```ts
        'whatevers': {
            component: asyncComponent(
                () => import('Containers/Whatevers/WhateversRoute')
            ),
            path: whateversPathWithParam,
        },
        ```

3. Edit ui/apps/platform/src/Containers/MainPage/Navigation/NavigationSidebar.tsx file, **if** the route has a link.

    * Import a path **without params**.

    ```ts
    whateversBasePath,
    ```

    * Add a child item for the link in the `navDescriptions` array.

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
