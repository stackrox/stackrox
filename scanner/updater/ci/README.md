# Scanner CI vulnerability bundle

This frozen dataset contains the package/advisory relationships needed by Scanner
and backend tests. Update it when a test needs new coverage or a dependency or
format change requires compatible fixtures. There is no periodic data refresh.
The author of the affected test or compatibility change maintains its fixtures.

## Origin and source of truth

The original minimal ZIP was filtered from a pinned production snapshot using
CI package and vulnerability requirements. During the
[offline generator implementation](https://github.com/stackrox/stackrox/commit/d79122fe33533aaa62330ba22373d640d82c65c0),
a one-off transcription helper converted that ZIP into typed Go fixture tables.
The input ZIP's SHA-256 was
`41c160a910cfa9600f9e6e7b69af0e8ba26032f35bd207373ae3206a6577aa35`.
That temporary helper is not a maintained extraction tool.

Today, the source of truth is the `fixtures_*.go` files under [generate/](generate/).
The [generator](generate/main.go) builds the ZIP offline from those tables;
it does not read a production bundle, tests, or the network. “Synthetic” refers
to the assembled test dataset: package/advisory relationships come from
upstream records, rather than invented findings. The checked-in ZIP lets tests
read the dataset without regeneration or network access.
