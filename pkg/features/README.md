# Feature Flags

You can use feature flags to control visibility of a feature or a block of code. The end user will not see the feature unless the flag is enabled.
With this, you can control the release of said feature or allow a user to enable/disable some functionality in production.
Currently, feature flags use environment variables to power the toggles.

Feature flags can be valuable to ship features in a preview state, to provide the end user a way to disable some functionality or to control any boolean setting.

> :warning: Feature flags cannot be used inside migrations or schema changes. In other words any such change will always be applied regardless of any feature flag value.

## Adding a feature flag

To add a feature flag, add a variable with your feature to `list.go`. To register this feature flag variables, you are required to provide the following:

* Name: This is a short description of what this flag is about
* Environment Variable name: This is the environment variable which needs to be set to override the default value of this flag. Env var names **must** start with `ROX_`
* Type: This is the type of the feature flag. An unchangeable or a changeable one. See below for more details
* Default value: The default value to use if the flag has not been overridden

The variable can be one of two types of feature flag:

#### An unchangeable feature:
This flag cannot be changed from its default value on release builds (i.e. "production"). To enable or disable it, you must make a code change.
On development builds, the setting _can_ be changed.

#### A changeable feature:
This is the default feature flag. The value of the flag can be changed in both release and development builds.
Use this if you want the end user to be able to enable or disable the setting.

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
