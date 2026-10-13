# Upgrade-test resource profile

The Central and PostgreSQL GKE upgrade jobs opt in with
`UPGRADE_TEST_RESOURCE_PROFILE=ci` and use one `e2-standard-16` node.
Other callers retain their existing resources.

The shared library passes `ci-central-values.yaml` or
`ci-secured-cluster-values.yaml` from the current checkout, including when using
historical charts. These files lower the larger CPU requests to 500m. Smaller
requests, CPU limits and all memory settings retain chart defaults. This replaces the renderer's CPU
limit removal with request-only overrides. Sidecar resources remain unchanged.
No post-renderer, Go build or Helm plugin is needed.

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
bats scripts/ci/bats/upgrade_resource_profile.bats
```

Rollback the workflow profile activation and machine type together if CI validation
fails. Production chart defaults, replica counts, images and storage are unchanged.
