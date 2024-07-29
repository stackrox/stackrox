# Feature Flags

You can use feature flags to control visibility of a feature or a block of code. The end user will not see the feature unless the flag is enabled.
With this, you can control the release of said feature or allow a user to enable/disable some functionality in production.
Currently, feature flags use environment variables to power the toggles.

Feature flags can be valuable to ship features in a preview state, to provide the end user a way to disable some functionality or to control any boolean setting.

Difference between feature flags and configuration options (DB, `pkg/env`):

* Feature flags support the development and the release of a feature, protecting the master branch from unfinished functionality. Features under flags are considered **unstable** and **not ready** for production use. Tech-preview flags in the last stage of development could be enabled for canary testing.
* Configuration options allow customers to configure a feature. Functionality has to be tested for the range of allowed values, including when no value is provided, and for the transition from one value to another.

## Feature lifecycle

1. A feature is being developed on a branch, gated by a dev-preview (default) feature flag.
2. The code meets the [GA requirements](../../PR_GA.md) with the feature disabled: the branch can be merged to the master branch.
3. The feature is ready to be tested by the customers: the flag can be upgraded to tech-preview. Consult [the article on Products & Services](https://access.redhat.com/articles/6966848) for the differences between the two stages.
4. The feature can be enabled for everybody: the flag can be enabled.
5. The feature has been *tested in production* for some time: the flag can be removed.

> :warning: Feature flags cannot be used inside migrations or schema changes.
> Migrations must be merged to the master branch without any feature flag gate, and must not break any current features.
>
> :warning: If a `--feature-flag` parameter is passed to the `pg-table-bindings-wrapper` generator, the schema is only applied if the flag is enabled.

## Adding a feature flag

To add a feature flag, add a variable with your feature to `list.go`. To register this feature flag variables, you are required to provide the following:

* Name: This is a short description of what this flag is about.
* Environment Variable name: This is the environment variable which needs to be set to override the default value of this flag. Env var names **must** start with `ROX_`.
* Options:
  * `devPreview` or `techPreview`: whether the feature is in early development or ready to be tested by customers. Flags are of the **dev-preview** stage by default.
  * `unchangeableInProd`: whether the flag state can be changed via the associated environment variable setting on release builds. Flags are **changeable** by default.
  * `enabled`: whether the change is activated by default.

> :warning: To introduce features that could be disabled in release builds, you must be cautious to ensure that Central returns to "normal" state after disabling the feature.
> Sometimes it is not as simple as turning off the feature flag to return Central to the "normal" state for various reasons, including (but not limited to) schema and data changes.

## Overriding the default value of a feature flag

To change the value of a feature flag on a running container, you must set or update the underlying environment variable to the desired value.

For example, if there exists a feature flag with an environment variable `ROX_MY_FEATURE_FLAG` and a default value of `false` and you want to override it to `true` in central (stackrox namespace), then run:

```sh
kubectl set env -n=stackrox deploy/central ROX_MY_FEATURE_FLAG=true
```

Use `oc` for OpenShift clusters.

## Using a feature flag in code

To use a feature flag, simply import the `features` package, and use `features.YOUR_FEATURE_VARIABLE.Enabled()`

The `Enabled` method will return true if the feature is enabled (either due to the default value or due to an override).

## Testing with a feature flag

In tests, it is recommended to test both the path where the flag is disabled and where it is enabled. To change the value, simple use
`T().Setenv(features.YOUR_FEATURE_VARIABLE.EnvVar(), "true")` (or `"false"` to disable).

Note that if the feature flag is an unchangeable one, the override will be ignored on tests that are run on release builds.

## Using feature flags in UI

All feature flags registered in `list.go` will be returned in the `v1/featureflags` API response with its corresponding status (true or false).
These values can be read by the UI to determine if a feature should be displayed or not.

## Removing a feature flag

Feature flags can be removed safely by following these steps:

1. First ensure that when the flag is disabled or enabled, the feature meets all GA requirements for being deployed in production, and a sufficient amount of time has passed such that no one will want to disable it again.
2. Remove all references in code to the feature flag variable and the associated environment variable. Remove any unreachable code. Take note of any tests and scripts (for example deploy scripts). Take note to remove references in UI as well.
3. Delete the variable from `list.go`.
