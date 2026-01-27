# Access control resources

The `list.go` file declares the resources used in the product to control
access to services and data.

## Creating a new resource

In general, we strive to avoid adding new resources, especially user-facing ones.
Sometimes it is necessary, but most of the time an existing resource is a better fit.
We avoid adding more resources, especially global ones, and strive to re-use existing
authorization patterns as well as existing resources, e.g., by editing
[the object type to resource mapping](https://github.com/stackrox/stackrox/blob/master/tools/generate-helpers/pg-table-bindings/list.go).

Creating a new resource involves a few steps.

1. Review the existing resources to find a fit for the object type being added.
2. Justify the need for a new resource (explain why the existing resources are not a good fit).
3. Get in touch with the sensors and ecosystems team to discuss the new resource.
4. Create the new resource in `pkg/sac/resources/list.go`.
5. Declare the new resource for the UI in `ui/apps/platform/src/types/roleResources.ts`.
6. Describe the resource as well as the meaning of read and write operations in `ui/apps/platform/src/Containers/AccessControl/PermissionSets/ResourceDescription.tsx`.
7. Request a review from the `stackrox/sensor-ecosystem` team on the PR introducing the new resource.