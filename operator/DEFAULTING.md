# Defaulting

This document provides more detail about possible ways to set defaults.
You can safely skip it, if the simple instructions included in [EXTENDING_CRDs.md](EXTENDING_CRDS.md) satisfy your needs.

Generally speaking, the translation logic [described in the aforementioned document](EXTENDING_CRDS.md#map-crd-setting-to-helm-chart-configuration) will set the corresponding Helm values field only for explicitly set values.
For absent values, it will simply do nothing, thus deferring to the chart's defaulting logic (see [example](https://github.com/stackrox/rox/blob/84d841c870f59d2c423f78eb7ecd44a196f8a659/operator/pkg/central/values/translation/translation.go#L86)).

On one hand this allows some level of consistency, but on the other hand it is not great for maintainability,
because it requires diving into the Helm charts in order to discover what the default for the operator is.
Therefore, it is better to explicitly provide a default for the operator.

To do this, you need to first decide which defaulting mechanism to use.
There exist different kinds of defaults:

* **Explicit setting using a `DefaultingExtension`** described in [the following section](#defaulting-extension-mechanism).

  This is the recommended mechanism.

* **Ad-hoc defaults on the level of translation logic.**

  The translation logic will recognize an absent (`nil`) value, decide on its meaning, and will set a corresponding
  value in the Helm values (see [example](https://github.com/stackrox/rox/blob/84d841c870f59d2c423f78eb7ecd44a196f8a659/operator/pkg/central/values/translation/translation.go#L120)).

  This method may be necessary in some cases (like mutually exclusive fields) but in general, if possible, please try to use the recommended one.

* _The following mechanism is not used for the ACS operator:_

  **Schema-level (a.k.a. static) defaults.**

  These are set in the schema via `+kubebuilder` directives.

  If a field value is not set by the user, the default will be inserted automatically upon object creation.
  This happens:
  * on OpenShift _when the user creates the CR using the form-based creation UI_, the OpenShift console sets
    the defaults everywhere in the CR, according to the schema.
  * on any Kubernetes platform, the kube API server also fills in defaults according to the schema,
    but _only if the enclosing struct field is already present_.

  Values set this way are persisted by the API server, and will be visible during translation.

  Changing a schema-level default counts as a breaking API change, but it is treated as such only semantically, nothing will fail at runtime (see [example](https://github.com/stackrox/rox/blob/84d841c870f59d2c423f78eb7ecd44a196f8a659/operator/apis/platform/v1alpha1/central_types.go#L188)).

  As of 2025 Q3 we stopped adding such static defaults.
  Instead, use one of the above mechanisms.


## Defaulting Extension Mechanism

The DefaultingExtension runs early in the reconciliation process and executes "defaulting flows" in sequence.
Each defaulting flow has the ability to populate `Central.Defaults` (of type `CentralSpec`) resp. `SecuredCluster.Defaults` (of type `SecuredClusterSpec`).
These `.Defaults` fields are then applied onto their sibling `.Spec` fields in a way that:
- preserves user choices,
- does not persist in the cluster, such that we have the ability to dynamically change the default in the future.

### Generic static defaulting flow

There is a generic "static defaulting" flow, which is the appropriate place for defaulting of most simple cases.
To use it, simply add a default value for your new field to the `staticDefaults` struct and mention the default value in the last line of field description,
as [described in EXTENDING_CRDS.md](EXTENDING_CRDS.md).

### Custom defaulting flow

For more complex cases, such as using different defaults for upgrade vs. new installation scenarios, you should add a custom defaulting flow.
Such defaulting flow can:
- Implement complex defaulting logic (beyond what static CRD defaulting supports).
- Persist defaulting decisions in the custom resource's metadata as feature-specific annotation.
- Differentiate between green-field (fresh installation) and brown-field (upgrade) scenarios when making defaulting decision.
- Perform validation which can differentiate between explicitly provided and defaulted values ([example](https://github.com/stackrox/stackrox/blob/master/operator/internal/central/defaults/central_db.go#L20-L23)).
- Ensure that subsequent reconciler extensions work with a custom resource spec that already includes all relevant defaulting decisions.

#### Reference Implementation

The two defaulting flows

* [`operator/internal/common/defaulting/central_scanner_v4_enabling.go`](https://github.com/stackrox/stackrox/blob/3864927b0825ebb95a1377daf8fb6afb0da8cfa7/operator/internal/common/defaulting/central_scanner_v4_enabling.go)
* [`operator/internal/common/defaulting/secured_cluster_scanner_v4_enabling.go`](https://github.com/stackrox/stackrox/blob/3864927b0825ebb95a1377daf8fb6afb0da8cfa7/operator/internal/common/defaulting/secured_cluster_scanner_v4_enabling.go)

can be used as blueprints when implementing new defaulting flows. New defaulting flows need to be added to
`operator/internal/central/extensions/reconcile_defaulting.go:defaultingFlows` resp.
`operator/internal/securedcluster/extensions/reconcile_defaulting.go:defaultingFlows`.

#### Annotation Format

Every defaulting decision that is persisted as an annotation should follow this naming convention:
```
metadata:
  annotations:
    "feature-defaults.platform.stackrox.io/<FEATURE_IDENTIFIER>": "<VALUE>"
```

Example:
```
metadata:
  annotations:
    "feature-defaults.platform.stackrox.io/scannerV4": "Enabled"
```
This annotation is added by the defaulting flow responsible for determining whether Scanner V4 should be enabled.
If the defaulting logic decides that Scanner V4 should be enabled by default, it adds this annotation to the custom resource.
This preserves the decision across reconciliation cycles and ensures consistent behavior during future upgrades.
