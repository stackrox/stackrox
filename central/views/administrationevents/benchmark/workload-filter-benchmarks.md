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

The bound values were `2026-01-01 00:00:00 +00:00`, `2027-01-01 00:00:00 +00:00`, `Image`, `app`, `payments`, and `Image`. The page query selects serialized events, orders by last occurrence descending, and returns up to 50. These plan observations used `plan_cache_mode=auto` after six executions.

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

Each pair below is **EXISTS / IDs in Go**. Times are median milliseconds per request from three Go benchmark samples with a one-second target. Each dataset and plan mode has its own database fixture. PostgreSQL uses the specified mode for every execution, including the warmup and both statements of the two-query strategy. The measurements include query construction, SQL compilation, database calls, and page decoding. KiB/op measures cumulative Go heap allocation. PostgreSQL memory and process peak RSS were outside this measurement.

**`force_custom_plan`**

| Events | Matching IDs | Count (ms) | Page (ms) | Count (KiB/op) | Page (KiB/op) |
|---:|---:|---:|---:|---:|---:|
| 1,000 | 10 | 0.58 / 0.73 | 0.79 / 0.96 | 77 / 62 | 161 / 146 |
| 10,000 | 250 | 82.02 / 3.66 | 95.85 / 4.30 | 77 / 389 | 161 / 472 |
| 100,000 | 1,000 | 108.10 / 22.88 | 123.73 / 23.20 | 77 / 1,417 | 161 / 1,500 |
| 100,000 | 10,000 | 111.84 / 44.67 | 139.29 / 49.69 | 77 / 14,527 | 161 / 14,611 |
| 100,000 | 100,000 | 386.05 / 284.47 | 391.46 / 301.56 | 78 / 151,987 | 161 / 152,068 |

**`force_generic_plan`**

| Events | Matching IDs | Count (ms) | Page (ms) | Count (KiB/op) | Page (KiB/op) |
|---:|---:|---:|---:|---:|---:|
| 1,000 | 10 | 68.31 / 0.87 | 68.49 / 1.10 | 77 / 62 | 161 / 146 |
| 10,000 | 250 | 9.25 / 22.90 | 10.69 / 23.23 | 77 / 389 | 161 / 472 |
| 100,000 | 1,000 | 110.05 / 332.26 | 124.14 / 322.28 | 77 / 1,416 | 161 / 1,499 |
| 100,000 | 10,000 | 115.90 / 1,515.51 | 136.88 / 1,503.58 | 77 / 14,526 | 161 / 14,609 |
| 100,000 | 100,000 | 389.78 / 14,212.69 | 401.97 / 14,086.57 | 78 / 151,985 | 161 / 152,071 |

**Automatic plan selection at 100,000 IDs.** A separate diagnostic uses `plan_cache_mode=auto` and records the event statement's plan counters after each request. In this run, PostgreSQL used custom plans for the first five executions of both reads. Execution six switched to a generic plan and took 14.08 s for count and 14.01 s for page. The timings above keep those two regimes separate.

## Reproduce from the repository root

Use the backend branch of the stack containing the administration event view and these tests. The local runs used PostgreSQL 15.20 on arm64, Go 1.27.2 on an Apple M4 Pro, `work_mem=4MB`, `shared_buffers=128MB`, JIT on and separate forced custom and generic plans for the benchmarks. The prepared-execution diagnostic uses `plan_cache_mode=auto`. The test creates an isolated database, applies the repository schemas and indexes, bulk-loads two containers per deployment and one timestamped event per second (90% image, 10% node), and runs `ANALYZE`. It uses one database connection. The exact plan and elapsed time can vary with the server, statistics and settings.

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

# Run all 40 variants across five sizes, two plan modes, two reads and two strategies.
go test -tags sql_integration "$pkg" -run '^$' -bench '^BenchmarkWorkloadFilter$' \
  -benchmem -benchtime=1s -count=3 -timeout=25m

# Timings and prepared-plan counters for the two-query approach in auto mode.
go test -tags 'sql_integration workload_experiments' "$pkg" \
  -run '^TestWorkloadPreparedExecutions$' -v -count=1 -timeout=20m
```

To isolate the custom-plan and JIT effects, use these per-command overrides.

```sh
PGOPTIONS='-c plan_cache_mode=force_custom_plan' go test -tags sql_integration "$pkg" \
  -run '^TestWorkloadFilterQueryPlan$/^small$' -v -count=1

PGOPTIONS='-c jit=off' go test -tags sql_integration "$pkg" \
  -run '^$' -bench '^BenchmarkWorkloadFilter$' -benchmem -benchtime=1s -count=3 -timeout=25m

PGOPTIONS='-c jit=off' go test -tags sql_integration "$pkg" \
  -run '^$' -bench '^BenchmarkWorkloadFilter$/^100000_events_10000_images$/^force_custom_plan$/^page$' \
  -benchmem -benchtime=1s -count=3
```

The implementation and fixtures are in [the benchmark](workload_benchmark_test.go) and [the prepared-plan diagnostic](workload_plan_experiments_test.go). The query-builder examples are in [common_test.go](../../../../pkg/search/postgres/common_test.go) and [select_query_test.go](../../../../pkg/search/postgres/select_query_test.go). To remove the disposable database container, run `docker stop workload-filter-pg`.
