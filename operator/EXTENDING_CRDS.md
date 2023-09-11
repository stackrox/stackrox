# Extending our Custom Resource Definitions

This document describes the required steps for extending the CRDs (for *StackRox Central* and *StackRox Secured Cluster*) supported by our Kubernetes Operator alongside 
some general rules and best practices for modifying the CRDs.

Let us assume that there is a new (fictitious) feature "Low Energy Consumption Mode", which we would like to implement support for. Prerequisite is that support for this new feature has already been implemented in the relevant Helm charts (`stackrox-central-services` and/or `stackrox-secured-cluster-services`). See [a separate document on how to do this](../image/templates/CHANGING_CHARTS.md).

## Add new Setting

* Add a new setting for the feature to the appropriate structs within `operator/apis/platform/<VERSION>/securedcluster_types.go` and/or `operator/apis/platform/<VERSION>/central_types.go`. For example:

```go
	// Set this to 'true' to enable low energy consumption mode.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=42
	LowEnergyConsumption *bool `json:"lowEnergyConsumption,omitempty"`
```

The `operator-sdk`-marker in the comment is used for exposing the setting as a user-visible configuration option during operator installation.
See [API markers](https://sdk.operatorframework.io/docs/building-operators/golang/references/markers/) for more information on this.

For a description of the `kubebuilder`-markers we use in `central_types.go` and `securedcluster_types.go`, see the
[kubebuilder manual](https://book.kubebuilder.io/reference/markers.html). In particular they are used for configuring
default values, as in e.g.

```go
//+kubebuilder:default=false
```

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

In order for the new setting to be effective, the new field added to a CRD needs to be translated into the appropriate Helm chart configuration. This translation needs to be added to `operator/pkg/central/values/translation/translation.go` and/or `operator/pkg/securedcluster/values/translation/translation.go`. Tests related to the translation of the new setting need to be added to the corresponding `translation_test.go` files.

For example, assuming that the corresponding Helm chart setting is named `lowEnergyConsumption`, use something like

```go
v.SetBoolValue("lowEnergyConsumption", sc.Spec.LowEnergyConsumption)
```

Regarding defaulting, note that there exist different kinds of defaults:

* Schema-level defaults: These are set via `+kubebuilder` directives, and if not set by the user, will be inserted automatically upon object creation.
  These values will be visible during translation, but only if the enclosing struct field is already present. Changing a schema-level default
  counts as a breaking API change, but it is treated as such only semantically, nothing will fail at runtime (see [example](https://github.com/stackrox/rox/blob/84d841c870f59d2c423f78eb7ecd44a196f8a659/operator/apis/platform/v1alpha1/central_types.go#L188))

* Translation logic-level defaults: The translation logic will recognize an absent (`nil`) value, decide on its meaning, and will set a corresponding
  value in the Helm values (see [example](https://github.com/stackrox/rox/blob/84d841c870f59d2c423f78eb7ecd44a196f8a659/operator/pkg/central/values/translation/translation.go#L120)).

* Propagating chart-level defaults: The translation logic will set the corresponding Helm values field only for explicitly set values; for absent
  values, it will do nothing, thus deferring to the chart's defaulting logic (see [example](https://github.com/stackrox/rox/blob/84d841c870f59d2c423f78eb7ecd44a196f8a659/operator/pkg/central/values/translation/translation.go#L86)).

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
* Use ints with specific width, see [API conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#primitive-types).

Some data types are discouraged, in particular:
* Avoid floats in spec, avoid floats in status if possible.
* Avoid unsigned ints.
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

## Breaking changes

The CR of an operator acts like a public API to the ACS configuration.
Additionally we need to keep in mind that CRs are often managed in CI/CD pipelines which would 
break existing automations.

 * Never remove a CR field from a CRD
 * Never remove an enum field from a CRD
 * Defined values must continue to stay valid values, i.e.: 
   * `KERNEL_MODULE` defaults to `EBPF`, but `KERNEL_MODULE` remains a valid value as an alias for `EBPF`
   * central-db is still a valid config in the Central CR even though users must use postgres
 * Defaults can change when the change is not going to break things

**Introducing a breaking change:**

This is possible by promoting a CR, i.e. from `v1alpha1` to `v1beta`. Technically this 
is equally to introducing moving from a `/v1` to a `/v2` API.

 * Using conversion webhooks from v1alpha1 to v1beta1 (introducing a CRD with a new version)
 * Kubernetes deprecation notices are 2 years on their resources

