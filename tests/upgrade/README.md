# Upgrade-test resource profile

The Central and PostgreSQL GKE upgrade jobs opt in with
`UPGRADE_TEST_RESOURCE_PROFILE=ci` and use one `e2-standard-16` node.
Other callers retain their existing resources.

The shared library passes explicit Roxie `--set` overrides for CPU requests of
500m on Central, Central DB, Scanner, Scanner V4 indexer/DB and Sensor.
CPU limits and memory retain the Roxie CI profile defaults.

The profile is passed to the initial Central and secured-cluster installations.
The operator retains the Central resource settings during the upgrade to HEAD.
The CI profile requires exactly one node: the scale
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
bats scripts/ci/bats/upgrade_resource_profile.bats
```

Rollback the workflow profile activation and machine type together if CI validation
fails. Production chart defaults, replica counts, images and storage are unchanged.
