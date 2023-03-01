Notes:

- Might be useful to rename the main `graphql()` function that is generated to avoid confusion on automatic imports.

Bugs or potential bugs found during conversion of WorkloadCVE section:

- The server-side response of type `Time` is actually a `string`. It was previously asserted to be a `Date` client side.
- The `image.metadata.v1.layers` array could contain `null` values. It was previously asserted that all values would be non-null.
- `image.operatingSystem` was incorrect typed as nullable, when it will never be null.
- All places where the top level `image`, `imageCVE`, or `deployment` were queried by ID could have be null, asserted non-null client side.