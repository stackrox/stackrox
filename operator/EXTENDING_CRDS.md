# Extending our Custom Resource Definitions

This document describes the required steps for extending the CRDs (for *StackRox Central* and *StackRox Secured Cluster*) supported by our Kubernetes Operator alongside 
some general rules and best practices for modifying the CRDs.

Let us assume that there is a new (fictitious) feature "Low Energy Consumption Mode", which we would like to implement support for. Prerequisite is that support for this new feature has already been implemented in the relevant Helm charts (`stackrox-central-services` and/or `stackrox-secured-cluster-services`). See [a separate document on how to do this](../image/templates/CHANGING_CHARTS.md).

## Add new Setting

Add a new setting for the feature to the appropriate structs within `operator/api/<VERSION>/securedcluster_types.go` and/or `operator/api/<VERSION>/central_types.go`.
Note the [style recommendations](#style-recommendations) below.

For example:

```go
// EnergyConsumptionMode is a type for values of spec.energyConsumptionMode.
// +kubebuilder:validation:Enum=High;Low
type EnergyConsumptionMode string

const (
	// EnergyConsumptionModeHigh configures central to use as much energy as it needs.
    EnergyConsumptionModeHigh EnergyConsumptionMode = "High"
	// EnergyConsumptionModeLow configures central to save energy, at the cost of some performance.
    EnergyConsumptionModeLow EnergyConsumptionMode = "Low"
)

type CentralSpec struct {
	// ...

	// Central energy consumption mode.
	// The default is: High.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=42
	EnergyConsumptionMode *EnergyConsumptionMode `json:"energyConsumptionMode,omitempty"`
}
```

Note the leaf field should be a pointer, and specify `json:"omitempty"` (unless `required`, but see below about required fields).

The `operator-sdk` CSV marker in the comment is used for exposing the setting as a user-visible configuration option during operator installation.
See [API markers](https://sdk.operatorframework.io/docs/building-operators/golang/references/markers/) for more information on this.

The `kubebuilder` validation marker ensures the only possible values are the enumerated ones.
For a description of the `kubebuilder`-markers we use in `central_types.go` and `securedcluster_types.go`, see the
[kubebuilder manual](https://book.kubebuilder.io/reference/markers.html).

Note that as of May 2025 we avoid using static default values, as in e.g.

```go
//+kubebuilder:default=...
```

You might still see examples of these in legacy code. They are being removed as part of ROX-22588.
Instead, we use:
- a line in the field's description comment to describe the default (see above), and
- runtime defaulting in the operator code,
  ([central](https://github.com/stackrox/stackrox/blob/master/operator/internal/central/values/defaults/defaults.go),
  [secured cluster](https://github.com/stackrox/stackrox/blob/master/operator/internal/securedcluster/values/defaults/defaults.go))

In this toy example:

```go
var staticDefaults = platform.CentralSpec{
	// ...

    EnergyConsumptionMode: platform.EnergyConsumptionModeHigh,
}
```

Note that in this case, the last line in the field description should explain how the default is set, using the specific syntax shown above.
There are unit tests that enforce that the comment matches the default set in the code.

See the [section on defaulting](#defaulting) below for more details, if you have more complex needs.

## Update generated Files

Run the following command:

```sh
make -C operator generate manifests bundle
```

to update all auto-generated files if required. For example, after adding a new field to `central_types.go` the following files are updated by the above `make` command:
```
operator/apis/platform/v1alpha1/zz_generated.deepcopy.go
operator/bundle/manifests/platform.stackrox.io_centrals.yaml
operator/bundle/manifests/rhacs-operator.clusterserviceversion.yaml
operator/config/crd/bases/platform.stackrox.io_centrals.yaml
operator/config/manifests/bases/rhacs-operator.clusterserviceversion.yaml
```

## Map CRD Setting to Helm chart configuration

In order for the new setting to be effective, the new field added to a CRD needs to be translated into the appropriate Helm chart configuration.
This translation needs to be added to `operator/pkg/central/values/translation/translation.go` and/or `operator/pkg/securedcluster/values/translation/translation.go`.
Tests related to the translation of the new setting need to be added to the corresponding `translation_test.go` files.

For example, assuming that the corresponding Helm chart setting is a boolean named `lowEnergyConsumption`, use something like

```go
if c.Spec.EnergyConsumption != nil {
    v.SetBoolValue("lowEnergyConsumption", *c.Spec.EnergyConsumptionMode == EnergyConsumptionModeLow)
}
```

## Defaulting

This section provides more detail about possible ways to set defaults.
You can safely skip it, if the above satisfies your needs.

Generally speaking, the translation logic described above will set the corresponding Helm values field only for explicitly set values.
For absent values, it will do nothing like above, thus deferring to the chart's defaulting logic (see [example](https://github.com/stackrox/rox/blob/84d841c870f59d2c423f78eb7ecd44a196f8a659/operator/pkg/central/values/translation/translation.go#L86)).

On one hand this allows some level of consistency, but on the other hand it is not great for maintainability,
because it requires diving into the Helm charts in order to discover what the default for the operator is.
Therefore, it is better to explicitly provide a default for the operator.

To do this, you need to first decide which defaulting mechanism to use.
There exist different kinds of defaults:

* Schema-level (a.k.a. static) defaults: These are set in the schema via `+kubebuilder` directives. 
  If a field value is not set by the user, the default will be inserted automatically upon object creation and persisted.
  These values will be visible during translation, but only if the enclosing struct field is already present. Changing a schema-level default
  counts as a breaking API change, but it is treated as such only semantically, nothing will fail at runtime (see [example](https://github.com/stackrox/rox/blob/84d841c870f59d2c423f78eb7ecd44a196f8a659/operator/apis/platform/v1alpha1/central_types.go#L188))

  As mentioned above, as of May 2025 we stopped adding such static defaults.
  Instead, use one of the following mechanisms.

* Ad-hoc defaults on the level of translation logic: The translation logic will recognize an absent (`nil`) value, decide on its meaning, and will set a corresponding
  value in the Helm values (see [example](https://github.com/stackrox/rox/blob/84d841c870f59d2c423f78eb7ecd44a196f8a659/operator/pkg/central/values/translation/translation.go#L120)).
  
  This method may be necessary in some cases (like mutually exclusive fields) but in general please try to use one of the following mechanisms.

* Explicit setting using a `DefaultingExtension` (see [below](#defaulting-extension-mechanism)).

### Defaulting Extension Mechanism

The DefaultingExtension runs early in the reconcilliation process and executes "defaulting flows" in sequence.
Each defaulting flow has the ability to populate `Central.Defaults` (of type `CentralSpec`) resp. `SecuredCluster.Defaults` (of type `SecuredClusterSpec`).
These `.Defaults` fields are then applied onto their sibling `.Spec` fields in a way that:
- preserves user choices,
- does not persist in the cluster, such that we have the ability to dynamically change the default in the future.

#### Generic static defaulting flow

There is a generic "static defaulting" flow, which is the appropriate place for defaulting of most simple cases.
To use it, simply add a default value for your new field to the `staticDefaults` struct, as [described above](#add-new-setting).

#### Custom defaulting flow

For more complex cases, such as using different defaults for upgrade vs. new installation scenarios, you should add a custom defaulting flow.
Such defaulting flow can:
  - Implement complex defaulting logic (beyond what static CRD defaulting supports).
  - Persist defaulting decisions in the custom resource's metadata as feature-specific annotation.
  - Differentiate between green-field (fresh installation) and brown-field (upgrade) scenarios when making defaulting decision.
  - Ensure that subsequent reconciler extensions work with a custom resource spec that already includes all relevant defaulting decisions.

##### Reference Implementation

The two defaulting flows

* [`operator/internal/common/defaulting/central_scanner_v4_enabling.go`](https://github.com/stackrox/stackrox/blob/3864927b0825ebb95a1377daf8fb6afb0da8cfa7/operator/internal/common/defaulting/central_scanner_v4_enabling.go)
* [`operator/internal/common/defaulting/secured_cluster_scanner_v4_enabling.go`](https://github.com/stackrox/stackrox/blob/3864927b0825ebb95a1377daf8fb6afb0da8cfa7/operator/internal/common/defaulting/secured_cluster_scanner_v4_enabling.go)

can be used as blueprints when implementing new defaulting flows. New defaulting flows need to be added to
`operator/internal/central/extensions/reconcile_defaulting.go:defaultingFlows` resp.
`operator/internal/securedcluster/extensions/reconcile_defaulting.go:defaultingFlows`.

##### Annotation Format

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

## Breaking changes

The CR of the ACS operator is the public API of the ACS configuration.
Additionally, we need to keep in mind that CRs are often managed in CI/CD pipelines which would
break existing automations.

* Never remove a CR field from a CRD.
  Instead, you can mark it as deprecated (and ignored) in its description and hide it in the operator console UI
  using the `urn:alm:descriptor:com.tectonic.ui:hidden` CSV `xDescriptor`; you can find plenty of examples in the code.
  * For example `central-db` is still a valid config section in the Central CR even though users must use postgres
* Never remove an enum value from a CRD. Instead, document that this value is deprecated and which value will be used
  instead if this one is selected.
  * For example in SecuredCluster `spec.perNode.collector.collection`, value `KernelModule` remains a valid value as an alias for `CORE_BPF`
* Never add a new `required` annotation for a new or existing field.
* Defaults **can** change when the change is not going to break things

**Introducing a breaking change:**

This is in theory possible by introducing a new API version, e.g. from `v1alpha1` to `v1beta`.
In practice, we do not want to do this, because currently (mid-2025) it requires shipping a round-trip-safe conversion webhook,
which is painful both to develop and operate.

### crd-schema-checker

We have a [CI check](../.github/workflows/check-crd-compatibility.yaml) based on [crd-schema-checker](https://github.com/openshift/crd-schema-checker) - a tool for finding breaking changes and violations of best practices in CRDs.

`crd-schema-checker` has limitations. It cannot find all types of breaking changes and violations of best practices.
+A passing grade from crd-schema-checker does not guarantee that there are no breaking changes or violations of best practices.
However, it does find the above types of breakage.

You can also run the checker manually:

crd-schema-checker can be run on one CRD, to check for violations of best practices, or it can be run on two CRDs to find breaking changes and violations of best practices introduced into the new CRD.

When running crd-schema-checker on ACS CRDs, many violations of best practices are reported. These include use of booleans and maps. Most if not all of these violations cannot be fixed without introducing breaking changes.

The following shows how to run crd-schema-checker.

```
crd-schema-checker check-manifests [--existing-crd-filename=] --new-crd-filename=
```

Example

```
cd operator/config/crd/bases
git show release-4.4:./platform.stackrox.io_securedclusters.yaml > platform.stackrox.io_securedclusters-4.4.yaml
crd-schema-checker check-manifests --existing-crd-filename=platform.stackrox.io_securedclusters-4.4.yaml --new-crd-filename=platform.stackrox.io_securedclusters.yaml
```

## Style Recommendations

### Naming

See the [Kube Builder book](https://book.kubebuilder.io/cronjob-tutorial/api-design.html) and [API Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#naming-conventions). To summarize:

* Use declarative names, not imperative. Do not use an `is` prefix.
* Do not use abbreviations, be careful with acronyms.
* Use camelCase field names.
* For modelling units, see [unit-related conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#units).

### Data Type Choices

For certain use-cases some data types are recommended, in particular:
* `resource.Quantity`: See [kubebuilder book](https://book.kubebuilder.io/cronjob-tutorial/api-design.html).
* `metav1.Time`: provides a stable serialization format for timestamps, see [kube builder book](https://book.kubebuilder.io/cronjob-tutorial/api-design.html).
* `ObjectReference` to refer to specific objects, see [API conventions](https://book.kubebuilder.io/cronjob-tutorial/api-design.html).
* Use integers with specific width, see [API conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#primitive-types).

Some data types are discouraged, in particular:
* Avoid floats in spec, avoid floats in status if possible.
* Avoid unsigned integers.
* Avoid iota-based enums, prefer named string constants instead.
* Think twice about booleans: See [API conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#primitive-types).
* Maps: See [API conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#lists-of-named-subobjects-preferred-over-maps).

### Other considerations related to data types:

* Optional/Required values: See [API conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#optional-vs-required).
* Strings, regex-based validation
* Constants/Enums: See [API conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#constants).
* Unions: See [API conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#unions).
* Defaulting: See [API conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#defaulting) and [Kubernetes Documentation](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#defaulting).
* Nullability: See [Kubernetes Documentation](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#defaulting-and-nullable).
* Late initialization: See [API Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#late-initialization).
* Labels, Selector and Annotations: See [API conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#label-selector-and-annotation-conventions).
