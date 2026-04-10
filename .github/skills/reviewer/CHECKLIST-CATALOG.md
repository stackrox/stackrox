# Checklist Catalog

## Contents
- [Universal Modules (always include)](#universal-modules-always-include)
  - [U: Code Quality](#u-code-quality)
  - [T: Testing](#t-testing)
  - [P: Production Readiness](#p-production-readiness)
  - [A: Architecture](#a-architecture)
  - [D: Documentation & Understandability](#d-documentation--understandability)
- [Language Modules](#language-modules)
  - [Go](#go)
  - [TypeScript/JavaScript](#typescriptjavascript)
  - [Python](#python)
  - [Rust](#rust)
  - [Java/Kotlin](#javakotlin)
- [Framework Modules](#framework-modules)
  - [Temporal Workflow](#temporal-workflow)
  - [React/Frontend](#reactfrontend)
- [Infrastructure Modules](#infrastructure-modules)
  - [Database/ORM](#databaseorm)
  - [API/HTTP](#apihttp)
- [Module Selection Matrix](#module-selection-matrix)

---

## Universal Modules (always include)

### U: Code Quality

| ID | Check | Severity |
|----|-------|----------|
| U1 | Error/exception handling: errors are not swallowed silently; errors include context/stack info | High |
| U2 | Resource cleanup: files, connections, handles are closed/released (defer, finally, using, context managers, RAII) | High |
| U3 | Input validation at system boundaries (user input, API requests, config values, file contents) | High |
| U4 | No hardcoded secrets, credentials, API keys, or tokens in source code | Critical |
| U5 | No SQL injection, command injection, XSS, path traversal, or other injection vulnerabilities | Critical |
| U6 | Consistent naming conventions (casing, prefixes, verb forms) across the codebase | Low |
| U7 | Functions/methods have reasonable length (<50 lines guideline) and cognitive complexity (<15) | Medium |
| U8 | No dead code: unused imports, unreachable branches, commented-out code blocks, unused variables/functions | Low |
| U9 | Constants/enums used instead of magic numbers/strings scattered through code | Medium |
| U10 | Dependency direction is acyclic; no circular imports/dependencies between modules | High |
| U11 | Single responsibility: each file/module/class has one coherent purpose | Low |
| U12 | DRY: no significant code duplication (>10 lines repeated 3+ times) | Medium |
| U13 | Logging is structured (key-value, JSON) not string-interpolated; log levels used appropriately | Medium |
| U14 | Configuration centralized in one module; no scattered env var reads or hardcoded URLs | Medium |

### T: Testing

| ID | Check | Severity |
|----|-------|----------|
| T1 | Test files exist for all non-trivial source files — list coverage gaps by package/module | High |
| T2 | Tests cover both success AND error/edge-case paths | High |
| T3 | Tests are isolated: no shared mutable state, no test ordering dependencies | Medium |
| T4 | Mocks/stubs are specific (not overly permissive catch-alls) | Medium |
| T5 | Test helpers use framework conventions for accurate error reporting (t.Helper in Go, custom assertions in pytest, etc.) | Low |
| T6 | Tests run with race/thread-safety detection enabled (if applicable to language) | High |
| T7 | Integration vs unit tests are clearly separated (different directories, tags, or naming) | Low |
| T8 | Critical business logic paths have thorough test coverage | High |

### P: Production Readiness

| ID | Check | Severity |
|----|-------|----------|
| P1 | Health/readiness endpoints or probes defined (if deployed as service) | Medium |
| P2 | Metrics/observability: key operations are instrumented (latency, error rates, throughput) | Medium |
| P3 | Graceful shutdown: signal handling, drain connections, complete in-flight work | Medium |
| P4 | Container security: non-root user, minimal base image, no unnecessary tools in runtime | High |
| P5 | CI pipeline: lint + test + build + (security scan if applicable) | Medium |
| P6 | Dependencies are pinned to specific versions (lockfile exists and committed) | Medium |
| P7 | Dockerfile multi-stage build separates build dependencies from runtime | Low |

### A: Architecture

| ID | Check | Severity |
|----|-------|----------|
| A1 | Clear separation of concerns: presentation/API layer, business logic, data access, external integrations | Medium |
| A2 | Dependency direction flows inward: domain/models have no outward dependencies | High |
| A3 | External systems accessed through abstraction layer (client/adapter/gateway), not directly from business logic | Medium |
| A4 | Configuration and secrets management separated from business logic | Medium |
| A5 | Extensibility: adding a new integration or feature doesn't require modifying unrelated code | Low |

### D: Documentation & Understandability

| ID | Check | Severity |
|----|-------|----------|
| D1 | Exported/public APIs have doc comments explaining purpose, parameters, and return values | Low |
| D2 | Complex algorithms or non-obvious logic have explanatory comments | Medium |
| D3 | No redundant comments that merely restate the code | Info |
| D4 | TODOs/FIXMEs are tracked — list all with their locations | Info |
| D5 | README exists with: what the project does, how to build, how to run, how to test | Low |

---

## Language Modules

### Go

| ID | Check | Severity |
|----|-------|----------|
| GO1 | Errors wrapped with `fmt.Errorf("context: %w", err)` — never bare `return err` without added context | Medium |
| GO2 | `context.Context` is the first parameter of functions that do I/O, and is propagated (not ignored) | High |
| GO3 | Long-running operations check `ctx.Done()` / `ctx.Err()` for cancellation | Medium |
| GO4 | Interfaces defined by the consumer (not the implementer) and kept small | Medium |
| GO5 | Exported types/functions have godoc comments starting with the name | Low |
| GO6 | `defer` used for cleanup (file close, mutex unlock, response body close) | High |
| GO7 | No goroutine leaks: spawned goroutines have shutdown paths via context or channels | High |
| GO8 | `sync.Mutex` / `sync.RWMutex` used correctly; no data races (verified by `-race` flag) | High |
| GO9 | Table-driven tests with `t.Run(name, ...)` subtests | Low |
| GO10 | `filepath.Join` used instead of string concatenation for file paths | Low |

### TypeScript/JavaScript

| ID | Check | Severity |
|----|-------|----------|
| TS1 | Strict TypeScript enabled (`"strict": true` in tsconfig) | High |
| TS2 | No `any` type usage except where genuinely unavoidable (count instances) | Medium |
| TS3 | Async/await used consistently (no mixing callbacks and promises unnecessarily) | Medium |
| TS4 | Promise rejections always handled (no unhandled promise rejections) | High |
| TS5 | Nullability handled: optional chaining, nullish coalescing, or explicit checks | Medium |
| TS6 | Dependencies in correct section (dependencies vs devDependencies) | Low |
| TS7 | ESLint/Biome configured and CI enforces it | Medium |
| TS8 | No `console.log` in production code (use structured logger) | Low |

### Python

| ID | Check | Severity |
|----|-------|----------|
| PY1 | Type hints used on function signatures and enforced by mypy/pyright | Medium |
| PY2 | Virtual environment / dependency management (poetry, pip-tools, uv) with lockfile | Medium |
| PY3 | Context managers (`with` statements) used for resource management | High |
| PY4 | Exception handling: specific exceptions caught (not bare `except:` or `except Exception`) | High |
| PY5 | No mutable default arguments (e.g., `def f(x=[])`) | High |
| PY6 | Async code: no blocking I/O in async functions; `asyncio.to_thread` for CPU-bound | Medium |
| PY7 | Ruff/flake8/black configured and CI enforces it | Medium |
| PY8 | Tests use pytest with fixtures, not unittest.TestCase (unless legacy) | Low |

### Rust

| ID | Check | Severity |
|----|-------|----------|
| RS1 | `unwrap()` / `expect()` only used where panic is acceptable (tests, infallible cases); production code uses `?` or match | High |
| RS2 | `Clone` not used to work around borrow checker when references would work | Medium |
| RS3 | Error types implement `std::error::Error` with proper `source()` chaining | Medium |
| RS4 | Lifetimes explicit only when necessary; elision used where possible | Low |
| RS5 | `clippy` configured and CI enforces it with `-D warnings` | Medium |
| RS6 | No `unsafe` blocks unless justified with safety comments | High |
| RS7 | Async runtime (tokio) used correctly: no blocking in async context | High |
| RS8 | Cargo.toml: features used for optional dependencies; no unnecessary feature flags | Low |

### Java/Kotlin

| ID | Check | Severity |
|----|-------|----------|
| JV1 | Null safety: `Optional` or `@Nullable`/`@NonNull` annotations (Java); null safety enforced (Kotlin) | High |
| JV2 | Resources closed with try-with-resources (Java) or `.use {}` (Kotlin) | High |
| JV3 | Exceptions: checked exceptions not abused; runtime exceptions have context messages | Medium |
| JV4 | Dependency injection framework configured correctly (Spring, Guice, Dagger, Koin) | Medium |
| JV5 | Thread safety: concurrent collections, synchronized blocks, or immutable objects where shared | High |
| JV6 | Logging via SLF4J/Logback (not System.out.println) with parameterized messages | Medium |

---

## Framework Modules

### Temporal Workflow

| ID | Check | Severity |
|----|-------|----------|
| TW1 | **No `time.Now()` / `Date.now()` / system clock in workflow code** — must use `workflow.Now(ctx)` or SDK equivalent. Trace ALL code paths reachable from workflow functions, including helpers | Critical |
| TW2 | **No goroutines/threads in workflow code** — use `workflow.Go()` / SDK async primitives | Critical |
| TW3 | **No `time.Sleep()` / `Thread.sleep()` in workflow code** — use `workflow.Sleep(ctx, duration)` | Critical |
| TW4 | **No map/dict iteration for ordering in workflow code** — use sorted/ordered collections | Critical |
| TW5 | **No I/O in workflow code** — all file, network, database operations must be in activities | Critical |
| TW6 | **ContinueAsNew for infinite/long-running loops** — workflows that loop with sleep must call ContinueAsNew after N iterations to prevent unbounded history growth (~50K event limit) | Critical |
| TW7 | Activity timeouts always set: every ExecuteActivity has explicit StartToCloseTimeout or ScheduleToCloseTimeout | Critical |
| TW8 | ScheduleToCloseTimeout >= StartToCloseTimeout (both set consistently) | High |
| TW9 | Heartbeat timeout set on long-running activities; activities call RecordHeartbeat periodically | High |
| TW10 | Retry policy: non-idempotent mutations have MaxAttempts=1; reads/queries allow retries | High |
| TW11 | Child workflow IDs are deterministic and unique | Medium |
| TW12 | Query handlers registered at workflow start, before any blocking calls | High |
| TW13 | Workflow state size bounded — no unbounded list/log growth in serialized state | High |
| TW14 | Non-retryable errors marked with `temporal.NewNonRetryableApplicationError` or equivalent | High |
| TW15 | Workflow versioning (`GetVersion`) used for changes to in-flight workflows | Medium |

### React/Frontend

| ID | Check | Severity |
|----|-------|----------|
| FE1 | Components follow single-responsibility (not god components with 500+ lines) | Medium |
| FE2 | State management: no prop drilling beyond 2 levels; context/store used appropriately | Medium |
| FE3 | Effects have correct dependency arrays (no missing deps, no unnecessary re-renders) | High |
| FE4 | User input sanitized before rendering (XSS prevention) | Critical |
| FE5 | Accessibility: semantic HTML, ARIA labels, keyboard navigation | Medium |
| FE6 | Bundle size: no giant dependencies imported for minor utilities | Low |

---

## Infrastructure Modules

### Database/ORM

| ID | Check | Severity |
|----|-------|----------|
| DB1 | Parameterized queries used (no string concatenation for SQL) | Critical |
| DB2 | Migrations are reversible and idempotent | Medium |
| DB3 | Indexes exist for frequently queried columns and foreign keys | Medium |
| DB4 | Connection pooling configured with reasonable limits | High |
| DB5 | Transactions used for multi-step mutations; isolation level appropriate | High |
| DB6 | N+1 query patterns avoided (eager loading or batching) | Medium |

### API/HTTP

| ID | Check | Severity |
|----|-------|----------|
| API1 | Authentication/authorization on all non-public endpoints | Critical |
| API2 | Rate limiting configured for public-facing endpoints | High |
| API3 | Request validation: size limits, type checks, required fields | High |
| API4 | Error responses don't leak internal details (stack traces, SQL errors) | High |
| API5 | CORS configured restrictively (not `*` in production) | Medium |
| API6 | Timeouts set on HTTP clients and servers | High |
| API7 | Pagination on list endpoints (no unbounded result sets) | Medium |

---

## Module Selection Matrix

| Detected signal | Add module(s) |
|----------------|---------------|
| `go.mod` present | Go |
| `package.json` present | TypeScript/JavaScript |
| `pyproject.toml`, `setup.py`, or `requirements.txt` | Python |
| `Cargo.toml` present | Rust |
| `pom.xml` or `build.gradle` | Java/Kotlin |
| Temporal SDK imports in source | Temporal Workflow |
| React imports or `.jsx`/`.tsx` files | React/Frontend |
| SQL queries, ORM usage, or migration files | Database/ORM |
| HTTP handlers, routers, or REST/gRPC definitions | API/HTTP |
