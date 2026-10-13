# Upgrade-test resource profile

The Central and PostgreSQL GKE upgrade jobs opt in with
`UPGRADE_TEST_RESOURCE_PROFILE=ci` and use one `e2-standard-16` node.
Other callers retain their existing resources.

The shared library builds a small Go Helm post-renderer from the current checkout,
not the historical checkout used to generate old charts. It caps CPU requests at
500m without raising smaller requests and removes CPU limits. Memory requests and
limits remain unchanged. It covers Deployment, DaemonSet and
StatefulSet application and init containers; hook Jobs are unchanged. It supports
executable post-renderers in Helm 3 and an isolated local plugin in Helm 4.

The profile is passed to the initial Central/secured-cluster installations and the
HEAD Central Helm upgrade. The CI profile requires exactly one node: the scale
launcher leaves resource requests unchanged on one node, so post-scale resource
patches are skipped. Multi-node profile use fails explicitly. Callers without the
profile retain the historical Central/DB CPU patches. Sensor bundle/upgrader tests deliberately retain their production
resource restoration and preservation checks; do not blanket-patch their resources.

Lower CPU requests permit bursting but do not prove reduced CPU demand. Both
complete upgrade jobs must pass without OOMs or capacity-related Pending pods;
check CPU contention, scale/upgrade-overlap memory and runtime. Sensor resource
restoration tests intentionally retain their production CPU requests.

Local checks:

```sh
go test ./tests/upgrade/resource-profile
bats scripts/ci/bats/upgrade_resource_profile.bats
```

Rollback the workflow profile activation and machine type together if CI validation
fails. Production chart defaults, replica counts, images and storage are unchanged.
