# PR 18935 Timing Data

## Build workflow — warm run 21890925999 (all HITs)

| Job | Cache | Time |
|-----|-------|------|
| pre-build-go-binaries (default, amd64) | HIT | 226s |
| pre-build-go-binaries (default, arm64) | HIT | 219s |
| pre-build-cli | HIT | 1,321s |
| pre-build-docs | HIT | 151s |
| operator STACKROX amd64 | HIT | 513s |
| operator STACKROX arm64 | HIT | 477s |
| operator RHACS amd64 | HIT | 639s |
| operator RHACS arm64 | HIT | 645s |

## Build workflow — cold run 21889095112 (all MISSes)

| Job | Cache | Time |
|-----|-------|------|
| pre-build-go-binaries (default, amd64) | MISS | 608s |
| pre-build-go-binaries (default, arm64) | MISS | 566s |
| pre-build-cli | MISS | 1,234s |
| pre-build-docs | MISS | 250s |
| operator STACKROX amd64 | MISS | 973s |
| operator STACKROX arm64 | MISS | 1,077s |
| operator RHACS amd64 | MISS | 1,200s |
| operator RHACS arm64 | MISS | 1,327s |

## Build workflow — second warm run 21894266785 (partial, still running)

| Job | Cache | Time |
|-----|-------|------|
| pre-build-go-binaries (default, amd64) | HIT | 244s |
| pre-build-go-binaries (default, arm64) | HIT | 205s |
| pre-build-docs | HIT | 151s |
| operator STACKROX amd64 | HIT | 534s |
| operator STACKROX arm64 | HIT | 456s |
| Unit test jobs | | still running |

## Scanner workflow — warm run 21890925983

| Job | Cache | Time |
|-----|-------|------|
| scanner amd64 | HIT | 135s |
| scanner arm64 | HIT | 295s |

## Style workflow — warm run 21890926012

| Job | Cache | Time |
|-----|-------|------|
| check-generated-files | HIT | 1,000s |
| style-check | HIT | 1,840s |
| golangci-lint | MISS | 1,393s |

## Unit tests — run 21894266766 (all HITs except sensor-integration)

| Job | Cache | Time |
|-----|-------|------|
| go (GOTAGS="") | HIT | 2,625s |
| go (GOTAGS=release) | HIT | 2,591s |
| go-postgres (GOTAGS="", 15) | HIT | 2,025s |
| go-postgres (GOTAGS=release, 15) | HIT | 1,928s |
| go-bench | HIT | 1,451s |
| local-roxctl-tests | HIT | 901s |
| sensor-integration-tests | MISS | 1,893s |

sensor-integration-tests MISS: runs on bare runner without container,
different GOCACHE path from container-based jobs.

## Summary comparison

| Job | Cold | Warm | Change |
|-----|------|------|--------|
| go-binaries amd64 | 608s | 226-244s | ~2.5x faster |
| go-binaries arm64 | 566s | 205-219s | ~2.6x faster |
| pre-build-cli | 1,234s | 1,321s | ~same (variance) |
| pre-build-docs | 250s | 151s | 1.7x faster |
| operator STACKROX amd64 | 973s | 513-534s | ~1.9x faster |
| operator STACKROX arm64 | 1,077s | 456-477s | ~2.3x faster |
| operator RHACS amd64 | 1,200s | 639s | 1.9x faster |
| operator RHACS arm64 | 1,327s | 645s | 2.1x faster |
| scanner amd64 | ~580s | 135s | 4.3x faster |
| scanner arm64 | ~550s | 295s | 1.9x faster |
| check-generated-files | ~1,000s | 1,000s | ~same |
| style-check | ~1,840s | 1,840s | ~same |
| golangci-lint | ~1,393s | not measured | MISS (awk bug fixed) |
| go (GOTAGS="") | ~2,600s | 2,625s | ~same (test-dominated) |
| go (GOTAGS=release) | ~2,600s | 2,591s | ~same |
| go-postgres (GOTAGS="") | ~2,000s | 2,025s | ~same |
| go-postgres (GOTAGS=release) | ~1,900s | 1,928s | ~same |
| go-bench | ~1,450s | 1,451s | ~same |
| local-roxctl-tests | ~900s | 901s | ~same |
| sensor-integration-tests | ~1,900s | 1,893s | MISS (bare runner) |

Run links:
- Warm build: https://github.com/stackrox/stackrox/actions/runs/21890925999
- Cold build: https://github.com/stackrox/stackrox/actions/runs/21889095112
- Warm style: https://github.com/stackrox/stackrox/actions/runs/21890926012
- Warm scanner: https://github.com/stackrox/stackrox/actions/runs/21890925983
- In-progress (unit tests): https://github.com/stackrox/stackrox/actions/runs/21894266785
