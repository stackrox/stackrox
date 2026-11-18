# Administration event workload filter

For the PR example, PostgreSQL sometimes computed the matching image IDs once and sometimes looked up an image for each event. The single-query `EXISTS` approach kept Go allocations nearly constant as the number of matching IDs grew. Fetching IDs into Go first was faster for several smaller workloads. At 100,000 IDs its latency depended heavily on which prepared plan PostgreSQL used.

## Example query and observed plans

The workload filter selects deployments named `app` in namespace `payments`. It returns image events whose resource ID occurs in a matching container and keeps non-image events in the requested time window. This is the count SQL emitted by the event view, with whitespace added.

```sql
SELECT count(*)
FROM administration_events
WHERE administration_events.CreatedAt >= $1
  AND administration_events.CreatedAt < $2
  AND ((administration_events.Resource_Type = $3 AND EXISTS (
        SELECT 1
        FROM (
          SELECT deployments_containers.Image_IdV2 AS image_id
          FROM deployments
          INNER JOIN deployments_containers
            ON deployments.Id = deployments_containers.deployments_Id
          WHERE deployments.Name = $4 AND deployments.Namespace = $5
        ) AS subquery
        WHERE subquery.image_id = administration_events.Resource_Id
      ))
      OR (TRUE AND NOT (administration_events.Resource_Type = $6)))
```

The bound values were `2026-01-01 00:00:00 +00:00`, `2027-01-01 00:00:00 +00:00`, `Image`, `app`, `payments`, and `Image`. The page query selects serialized events, orders by last occurrence descending, and returns up to 50.

| Events | Deployments | Matching distinct IDs / total | Warmed count and page plans |
|---:|---:|---:|---|
| 1,000 | 1,000 | 10 / 100 | Correlated subplan, 900 container scans |
| 10,000 | 10,000 | 250 / 2,500 | Hashed subplan, inner join ran once |
| 100,000 | 20,000 | 1,000 / 10,000 | Hashed subplan, inner join ran once |
| 100,000 | 20,000 | 10,000 / 10,000 | Hashed subplan, inner join ran once |
| 100,000 | 200,000 | 100,000 / 100,000 | Correlated subplan, 90,000 indexed lookups |

For 10,000 matching IDs, the filter referenced `hashed SubPlan 2`. The join of deployments and containers reported `actual rows=40000 loops=1`. PostgreSQL built a hash table of matching image IDs once, then looked up each image event’s resource ID in that table.

For 100,000 matching IDs, the subplan contained the following nodes (abridged).

```text
SubPlan 1
  -> Nested Loop                                      loops=90000
       -> Index Scan using deploymentscontainers_image_idv2
            Index Cond: image_idv2 = administration_events.resource_id
       -> Index Scan using deployments_pkey
            Index Cond: id = deployments_containers.deployments_id
```

The 10,000 non-image events passed through the other branch. Read the loop count on the first node **inside the subplan**. The outer event scan ran once in both cases.

## Time and Go allocations

The two-query implementation selects distinct image IDs through the deployment search API, collects them in Go, then calls the same event view. At the default threshold of 100 exact-match values the compiler switches from `IN` to one array parameter with `ANY`. Both approaches use the same time, resource-type and pagination filters. Tests check counts and compare the ordered page IDs.

Each pair below is **EXISTS / IDs in Go**. Times are median milliseconds per request from three Go benchmark samples with a one-second target. The measurements include query construction, SQL compilation, database calls, and page decoding. KiB/op measures cumulative Go heap allocation. PostgreSQL memory and process peak RSS were outside this measurement.

| Events | Matching IDs | Count (ms) | Page (ms) | Count (KiB/op) | Page (KiB/op) |
|---:|---:|---:|---:|---:|---:|
| 1,000 | 10 | 69.43 / 0.73 | 65.88 / 1.08 | 77 / 62 | 161 / 146 |
| 10,000 | 250 | 9.27 / 3.54 | 10.85 / 3.92 | 77 / 389 | 161 / 472 |
| 100,000 | 1,000 | 106.06 / 21.21 | 113.48 / 22.81 | 77 / 1,417 | 161 / 1,500 |
| 100,000 | 10,000 | 104.12 / 42.26 | 120.85 / 1,427.40 | 77 / 14,527 | 161 / 14,609 |
| 100,000 | 100,000 | 391.24 / 14,111.30 | 398.85 / 14,030.27 | 77 / 151,989 | 161 / 152,065 |

A repeat of the four smaller fixtures gave the same faster strategy for each count and page with JIT enabled. Fifteen of the 16 median times were within 9% of the previous run. The largest shift was the 250-ID EXISTS page, from 9.34 to 10.85 ms.

**Prepared statements explain the two-query result at 100,000 IDs.** The three count samples were 0.285, 14.161 and 14.111 seconds per request. The page samples were 0.293, 14.030 and 14.117 seconds. On a fresh connection we timed six calls and read the event statement's plan counters after each. The first five used custom plans and took 281–296 ms. The sixth used a generic plan and took about 14 seconds for either read. The small EXISTS fixture also changed shape between a custom plan (one hash build) and a generic plan (900 subplan executions).

**JIT and plan-cache settings affect the comparison.** With JIT off, the 100,000-ID EXISTS count and page took 323 and 329 ms, and the Go medians remained around 14 seconds. At 10,000 IDs the JIT-off count was close enough to change order between runs. For the 10,000-ID page, disabling JIT and forcing custom plans brought the two-query approach down to about 42 ms. These measurements represent one local client and synthetic data.

## Reproduce from the repository root

Use the backend branch of the stack containing the administration event view and these tests. The local runs used PostgreSQL 15.20 on arm64, Go 1.27.2 on an Apple M4 Pro, `work_mem=4MB`, `shared_buffers=128MB`, JIT on and `plan_cache_mode=auto`. The test creates an isolated database, applies the repository schemas and indexes, bulk-loads two containers per deployment and one timestamped event per second (90% image, 10% node), and runs `ANALYZE`. It uses one database connection. The exact plan and elapsed time can vary with the server, statistics and settings.

Use a local PostgreSQL server whose login can create databases, or start a disposable instance. The following commands use the local operating-system username and assume `CI` is unset.

```sh
docker run --rm -d --name workload-filter-pg \
  -e POSTGRES_USER="$USER" -e POSTGRES_HOST_AUTH_METHOD=trust \
  -p 127.0.0.1:55432:5432 postgres:15.20
until docker exec workload-filter-pg pg_isready -q -U "$USER" -d postgres; do sleep 1; done
export POSTGRES_HOST=127.0.0.1 POSTGRES_PORT=55432
```

For an existing instance, set the host and port to match. Run each measurement separately. The verbose plan tests print the statement, bound parameters, plan counters, actual rows, loops and buffer usage.

```sh
pkg=./central/views/administrationevents/benchmark
export ROX_FLATTEN_IMAGE_DATA=true ROX_POSTGRES_PARAMETER_THRESHOLD=100
export PGOPTIONS='-c jit=on -c plan_cache_mode=auto -c work_mem=4MB'

# COUNT, GET and SELECT examples, including a filter that excludes matching rows.
go test -tags sql_integration ./pkg/search/postgres \
  -run '^Test(CountQueries|GetQueries|SelectQuery)$/^subquery_' -v -count=1

# SQL and EXPLAIN ANALYZE of the warmed production count and page at all five sizes.
go test -tags sql_integration "$pkg" -run '^TestWorkloadFilterQueryPlan$' -v -count=1

# All 20 application benchmark variants, including the 100,000-ID case.
go test -tags sql_integration "$pkg" -run '^$' -bench '^BenchmarkWorkloadFilter$' \
  -benchmem -benchtime=1s -count=3 -timeout=25m

# Timings and prepared-plan counters for the two-query approach.
go test -tags 'sql_integration workload_experiments' "$pkg" \
  -run '^TestWorkloadPreparedExecutions$' -v -count=1 -timeout=20m
```

To isolate the custom-plan and JIT effects, use these per-command overrides.

```sh
PGOPTIONS='-c plan_cache_mode=force_custom_plan' go test -tags sql_integration "$pkg" \
  -run '^TestWorkloadFilterQueryPlan$/^small$' -v -count=1

PGOPTIONS='-c jit=off' go test -tags sql_integration "$pkg" \
  -run '^$' -bench '^BenchmarkWorkloadFilter$' -benchmem -benchtime=1s -count=3 -timeout=25m

PGOPTIONS='-c jit=off -c plan_cache_mode=force_custom_plan' go test -tags sql_integration "$pkg" \
  -run '^$' -bench '^BenchmarkWorkloadFilter$/^100000_events_10000_images$/^page$' \
  -benchmem -benchtime=1s -count=3
```

The implementation and fixtures are in [the benchmark](workload_benchmark_test.go) and [the prepared-plan diagnostic](workload_plan_experiments_test.go). The query-builder examples are in [common_test.go](../../../../pkg/search/postgres/common_test.go) and [select_query_test.go](../../../../pkg/search/postgres/select_query_test.go). To remove the disposable database container, run `docker stop workload-filter-pg`.
