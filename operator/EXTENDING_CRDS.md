# Extending our Custom Resource Definitions

This document describes the required steps for extending the CRDs (for *StackRox Central* and *StackRox Secured Cluster*) supported by our Kubernetes Operator alongside 
some general rules and best practices for modifying the CRDs.

## Guide: How to add a new field

Let us assume that there is a new (fictitious) feature "Low Energy Consumption Mode", which we would like to implement support for. Prerequisite is that support for this new feature has already been implemented in the relevant Helm charts (`stackrox-central-services` and/or `stackrox-secured-cluster-services`). See [a separate document on how to do this](../image/templates/CHANGING_CHARTS.md).

### 1. Add the new setting to the API

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
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=42,displayName="Energy consumption mode"
	EnergyConsumptionMode *EnergyConsumptionMode `json:"energyConsumptionMode,omitempty"`
}
```

Note the leaf field should be a pointer, and specify `json:"omitempty"` (unless `required`, but see below about required fields).

The `operator-sdk` CSV marker in the comment is used for exposing the setting as a user-visible configuration option during operator installation.
See [API markers](https://sdk.operatorframework.io/docs/building-operators/golang/references/markers/) for more information on this.
You can also have a look at other fields in the file, to see how the markers are being used.

The `kubebuilder` validation marker ensures the only possible values are the enumerated ones.
For a description of the `kubebuilder`-markers we use in `central_types.go` and `securedcluster_types.go`, see the
[kubebuilder manual](https://book.kubebuilder.io/reference/markers.html).

Note that as of 2025 Q3 we no longer use static default values, as in e.g.

```go
//+kubebuilder:default=...
```

Instead, we use:
- a line in the field's description comment to describe the default (see above), and
- runtime defaulting in the operator code, which you will add next


### 2. Set the default value

Add something like this in the [central](https://github.com/stackrox/stackrox/blob/master/operator/internal/central/values/defaults/defaults.go) `defaults.go` file:

```go
var staticDefaults = platform.CentralSpec{
	// ...

	EnergyConsumptionMode: platform.EnergyConsumptionModeHigh,
}
```

There is also a [secured cluster](https://github.com/stackrox/stackrox/blob/master/operator/internal/securedcluster/values/defaults/defaults.go) counterpart if you are modifying the `SecuredCluster` CRD.

Note that the last line in the field description should explain how the default is set, using the specific syntax shown above.
There are unit tests that enforce that the comment matches the default set in the code.

See the [separate document on defaulting](DEFAULTING.md) below for more details, if you have more complex needs than a static default.

### 3. Update generated files

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

### 4. Map CRD Setting to Helm chart configuration

In order for the new setting to be effective, the new field added to a CRD needs to be translated into the appropriate Helm chart configuration.
This translation needs to be added to `operator/pkg/central/values/translation/translation.go` and/or `operator/pkg/securedcluster/values/translation/translation.go`.
Tests related to the translation of the new setting need to be added to the corresponding `translation_test.go` files.

For example, assuming that the corresponding Helm chart setting is a boolean named `lowEnergyConsumption`, use something like

```go
if c.Spec.EnergyConsumption != nil {
    v.SetBoolValue("lowEnergyConsumption", *c.Spec.EnergyConsumptionMode == EnergyConsumptionModeLow)
}
```

### 5. Prepare a pull request

Please make sure you include the generated files in your PR.

Also, unless the field you added is hidden, please:
- [deploy your changed operator using OLM](README.md#installing-operator-via-olm)
- go to the OpenShift console for creating a new `Central` or `SecuredCluster` resource, as appropriate
- make a screenshot that shows your new field and paste it in the PR description

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
  Note however that we depart somewhat from the conventions as described in [DEFAULTING.md](DEFAULTING.md).
* Nullability: See [Kubernetes Documentation](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#defaulting-and-nullable).
* Late initialization: See [API Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#late-initialization).
* Labels, Selector and Annotations: See [API conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#label-selector-and-annotation-conventions).
