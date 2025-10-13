# Feature Flags

You can use feature flags to control visibility of a feature or a block of code. The end user will not see the feature unless the flag is enabled.
With this, you can control the release of said feature or allow a user to enable/disable some functionality in production.
Currently, feature flags use environment variables to power the toggles.

Feature flags can be valuable to ship features in a preview state, to provide the end user a way to disable some functionality or to control any boolean setting.

## Use cases

### Feature release stages

To promote a feature through the release stages, consider the following progression:

1. Until the feature is ready to be tested by customers, the flag should be disabled by default and unchangeable in production.
2. The flag, disabled by default, can be made changeable to allow customers share feedback on the feature.
3. When the feature is complete and generally available, the flag can be enabled by default.
4. After the feature has been tested in production for some time, but not longer, remove the flag.

### Feature deprecation

When an existing feature needs to be removed, a feature flag should be used to gradually deprecate the feature:

1. Create a feature flag, disabled by default, with a variable having `HIDDEN`, `SUPPRESSION` or similar word in the name. Do not use `DISABLED`, because the condition for the enabled feature will read badly.
2. Put the functionality being deprecated under condition, e.g., `if !featureSuppression.Enabled() { feature }`, and announce the deprecation.
3. After some grace period, make the flag enabled by default and announce removal.

### Dependencies

This package does not implement any support for managing dependencies between flags. Make sure to document potential dependencies close to the flag definition.

> :warning: Feature flags cannot be used inside migrations or schema changes.
> Migrations must be merged to the master branch without any feature flag gate, and must not break any current features.

## Adding a feature flag

To add a feature flag, add a variable with your feature to `list.go`. To register feature flag variables, you are required to provide the following:

* Name: This is a short description of what this flag is about.
* Environment variable name: This is the environment variable which users set to override the default value of this flag. Env var names **must** start with `ROX_`.
* Options:
  * `unchangeableInProd` makes the flag unchangeable in release builds. On development builds, the setting _can_ be changed. Consider using this option in the early stage of feature development.
  * `enabled` makes the flag enabled by default. Use this option when the feature is complete, meets GA requirements, and needs to be enabled.

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

1. First ensure that the feature is enabled by default and a sufficient amount of time has passed such that no one will want to disable it again.
2. Remove all references in code to the feature flag variable. Remove any unreachable code. Take note of any tests and scripts (for example deploy scripts). Take note to remove references in UI as well.
3. Delete the variable from `list.go`
