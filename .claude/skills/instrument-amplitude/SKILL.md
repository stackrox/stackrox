---
name: instrument-amplitude
description: Instrument StackRox features with Amplitude telemetry. Use when adding analytics tracking to new or existing features, implementing event tracking in the UI (React/TypeScript) or backend (Go), or when the user mentions telemetry, analytics, tracking, Amplitude, or Segment. Also use when building new UI features, pages, or API endpoints that would benefit from usage data collection, even if the user doesn't explicitly mention analytics.
---

# Amplitude Telemetry Instrumentation

Add concise, analysis-friendly telemetry to StackRox features. The system uses Segment as the transport layer with Amplitude as the analytics destination.

## Before You Start

1. Identify what user behavior or system event provides actionable insight.
2. Ask: "What question will this data answer in Amplitude?" If there is no clear question, do not add the event.
3. Prefer fewer, well-structured events over many granular ones. One event with descriptive properties is better than five separate events.

---

## Frontend Instrumentation (React/TypeScript)

All frontend telemetry flows through the `useAnalytics` hook and the `AnalyticsEvent` union type.

### Key Files

| File | Purpose |
|------|---------|
| `ui/apps/platform/src/hooks/useAnalytics.ts` | Event constants, `AnalyticsEvent` type, `useAnalytics` hook |
| `ui/apps/platform/src/init/initializeAnalytics.ts` | Segment SDK setup (do not modify) |
| `ui/apps/platform/src/utils/analyticsEventTracking.ts` | Filter tracking utility |

### Step 1: Define the Event Constant

Add a named constant to `useAnalytics.ts` in the appropriate section (grouped by feature area). Use `Title Case` with a descriptive past-tense or noun-phrase name.

```typescript
// Section: your feature area
export const MY_FEATURE_ACTION_COMPLETED = 'My Feature Action Completed';
```

**Naming rules:**
- Format: `[Feature Area]: [Action]` or `[Action]` if unambiguous.
- Use past tense for actions: "Created", "Submitted", "Downloaded", "Applied".
- Use present for states: "Opened", "Viewed", "Toggled".
- Match the existing style in the file. Existing examples: `'Cluster Created'`, `'Network Graph: Generate Network Policies'`, `'Workload CVE Filter Applied'`.

### Step 2: Add to the AnalyticsEvent Union Type

Events are either a plain string constant (no properties) or an object with typed properties.

**Simple event (no properties):**

```typescript
export type AnalyticsEvent =
    // ...existing events...
    | typeof MY_FEATURE_ACTION_COMPLETED;
```

**Event with properties:**

```typescript
export type AnalyticsEvent =
    // ...existing events...
    | {
          event: typeof MY_FEATURE_ACTION_COMPLETED;
          properties: {
              source: 'Page A' | 'Page B';
              itemCount: number;
          };
      };
```

### Step 3: Track the Event in Your Component

```typescript
import useAnalytics, { MY_FEATURE_ACTION_COMPLETED } from 'hooks/useAnalytics';

function MyComponent() {
    const { analyticsTrack } = useAnalytics();

    function handleAction() {
        // Simple event:
        analyticsTrack(MY_FEATURE_ACTION_COMPLETED);

        // Event with properties:
        analyticsTrack({
            event: MY_FEATURE_ACTION_COMPLETED,
            properties: { source: 'Page A', itemCount: 5 },
        });
    }
}
```

### Page Views

Use `analyticsPageVisit` for page-level tracking:

```typescript
const { analyticsPageVisit } = useAnalytics();

useEffect(() => {
    analyticsPageVisit('Feature Area', 'Page Name');
}, [analyticsPageVisit]);
```

### Filter Tracking

For search/filter events, reuse the `createFilterTracker` utility from `utils/analyticsEventTracking.ts`:

```typescript
import { createFilterTracker } from 'utils/analyticsEventTracking';

const trackAppliedFilter = createFilterTracker(analyticsTrack);
trackAppliedFilter(MY_FILTER_APPLIED, searchPayload);
```

If your filter event tracks values, add the safe filter categories to the `searchCategoriesWithFilter` tuple in `useAnalytics.ts`. Only add categories whose values contain no customer-specific data (e.g., `'Severity'`, `'Fixable'`).

The tuple is defined as a `const` assertion so callers get exact string types:

```typescript
// In useAnalytics.ts
export const searchCategoriesWithFilter = [
    'Severity',
    'Fixable',
    'Category',
    'Lifecycle Stage',
    // Add your safe category here (values must not contain customer-specific data).
] as const;
```

### What to Track by Feature Type

| Feature Pattern | Track These Events |
|---|---|
| CRUD page (list/create/edit/delete) | Create, Delete (edits rarely useful) |
| Wizard / multi-step form | Step changes (step name enum), Submit (success as 0/1) |
| Filter / search | Filter applied (category enum, redacted value) |
| Report / export | Generation triggered, Download attempted (success as 0/1) |
| Toggle / settings change | Toggled (setting name enum, new state as 0/1) |
| Modal dialog | Opened (source/entry-point as enum) |
| Entity detail view | Context viewed (entity type enum) |

Do not track: routine page navigation, every table sort, read-only data loads, intermediate form field changes.

---

## Backend Instrumentation (Go)

Backend telemetry uses the phone-home framework with interceptors and gatherers.

### Key Packages

| Package | Purpose |
|---------|---------|
| `pkg/telemetry/phonehome` | Client framework, interceptors, gatherers |
| `pkg/telemetry/phonehome/telemeter` | `Telemeter` interface and options |
| `central/telemetry/centralclient` | Central-specific wrapper |
| `central/<feature>/datastore/telemetry.go` | Feature-specific gatherer (preferred location) |
| `central/telemetry/gatherers` | Cross-cutting infrastructure metrics |

### API Call Tracking (Interceptors)

Interceptors fire on every API call and decide whether to emit an event. They are chained: if any interceptor returns `false`, the event is suppressed.

```go
import "github.com/stackrox/rox/pkg/telemetry/phonehome"

c.AddInterceptorFuncs("My Feature Used",
    func(rp *phonehome.RequestParams, props map[string]any) bool {
        if rp.Path != "/v1/myfeature" || rp.Method != "POST" {
            return false
        }
        props["status"] = rp.Code
        return true
    },
)
```

Register interceptors in `central/telemetry/centralclient/client.go` or in a feature-specific init function that receives the client.

### Periodic Data Gathering (Cross-Cutting)

For cross-cutting infrastructure metrics (database stats, API call counts), add gatherers in `central/telemetry/gatherers/`. For feature-specific metrics, use the datastore gatherer pattern below instead.

```go
g := c.Gatherer()
g.AddGatherer(func(ctx context.Context) (map[string]any, error) {
    count, err := myStore.Count(ctx)
    if err != nil {
        return nil, err
    }
    return map[string]any{
        "My Feature Item Count": count,
    }, nil
})
```

### Datastore Gatherer (Preferred for Feature Metrics)

The most common backend telemetry pattern: add a `telemetry.go` file alongside the datastore that owns the data. 15+ features use this convention.

**File location:** `central/<feature>/datastore/telemetry.go`

**Singleton variant** — when the datastore has a `Singleton()` accessor:

```go
var Gather phonehome.GatherFunc = func(ctx context.Context) (map[string]any, error) {
    ctx = sac.WithGlobalAccessScopeChecker(ctx,
        sac.AllowFixedScopes(
            sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
            sac.ResourceScopeKeys(resources.MyResource)))
    props := make(map[string]any)

    items, err := Singleton().GetAll(ctx)
    if err != nil {
        return nil, err
    }
    _ = phonehome.AddTotal(ctx, props, "My Features", phonehome.Len(items))
    return props, nil
}
```

**Injected variant** — when the datastore must be passed in:

```go
func Gather(ds DataStore) phonehome.GatherFunc {
    return func(ctx context.Context) (map[string]any, error) {
        ctx = sac.WithGlobalAccessScopeChecker(ctx, /* ... */)
        props := make(map[string]any)
        count, err := ds.Count(ctx)
        if err != nil {
            return nil, err
        }
        _ = phonehome.AddTotal(ctx, props, "My Features", phonehome.Constant(count))
        return props, nil
    }
}
```

**Registration:** Add to `addCentralIdentityGatherers` in `central/main.go`:

```go
add(myFeatureDS.Gather)                    // singleton variant
add(myFeatureDS.Gather(myFeatureDS.Singleton())) // injected variant
```

See `central/notifier/datastore/telemetry.go` (singleton) and `central/cloudsources/datastore/telemetry.go` (injected) as reference implementations.

### Entity Lifecycle Tracking

When a feature manages persistent entities (clusters, integrations), track registration and identity updates so Amplitude attributes events to each entity.

```go
c.Track("My Entity Registered", props,
    telemeter.WithClient(entity.GetId(), "My Entity", version),
)

c.Identify(
    telemeter.WithClient(entity.GetId(), "My Entity", version),
    telemeter.WithTraits(map[string]any{"Type": entity.GetType().String()}),
    telemeter.WithNoDuplicates(time.Now().Format(time.DateOnly)),
)
```

- `WithClient` — sets the entity as the event source (not Central). Use for entity-scoped events.
- `WithTraits` — attaches persistent properties to the entity identity.
- `WithNoDuplicates` — deduplicates per time period (e.g., daily). Use for periodic identity updates.

Do NOT use `WithClient` for ephemeral request-scoped events — those should use the default Central identity. See `central/cluster/datastore/telemetry.go` as the canonical example.

### Direct Tracking

For one-off events outside the request path:

```go
t := c.Telemeter()
t.Track("My Feature Migrated", map[string]any{
    "item_count": count,
    "duration_seconds": elapsed.Seconds(),
})
```

### Frontend vs. Backend: What Goes Where

When a feature spans both UI and API (e.g., report wizard + report generation endpoint):

- **Frontend tracks user intent and workflow:** wizard steps, button clicks, modal opens, form submissions. These answer: "How do users interact with this feature?"
- **Backend tracks system outcomes and aggregate state:** entity counts, configuration distribution, operation results. These answer: "How is this feature configured across the fleet?"

Do not duplicate the same event in both places. If the UI tracks "Report Created" with wizard properties, the backend should gather "Total Reports" as a periodic metric instead. Track the same action in both places only if you need to measure drop-off between client submission and server completion.

**Example: Report Generation Feature**

- **Frontend tracks intent:** wizard step changes (`step: 'parameters' | 'clusters' | 'review'`), "Report Creation Submitted" (with format, schedule type).
- **Backend tracks outcomes:** "Total Reports" count gatherer, "Reports by Format" distribution.
- **Do not** duplicate a "Report Created" event in both frontend and backend.
- **Exception:** track "Report Creation Submitted" (frontend) and "Report Generation Completed" (backend) when measuring client-to-server drop-off.

---

## Property Design Rules

Well-designed properties make Amplitude charts and funnels work correctly. Follow these rules for both frontend and backend.

### Use Enums, Not Free-Text

Properties used for grouping or filtering must have a bounded set of values. Free-text fields create cardinality explosions in Amplitude.

```typescript
// GOOD: bounded enum values.
properties: { source: 'Table row' | 'Details page' }

// BAD: unbounded free-text.
properties: { source: string }
```

### Use 0/1 for Booleans

Amplitude aggregates numeric values but not booleans. Use `0 | 1` (typed as `AnalyticsBoolean` in TypeScript) so you can SUM and AVERAGE in charts.

```typescript
type AnalyticsBoolean = 0 | 1;

properties: {
    EMAIL_NOTIFIER: AnalyticsBoolean;
    TEMPLATE_MODIFIED: AnalyticsBoolean;
}
```

### Use Counts, Not Lists

Track the count of items, not the items themselves. Lists bloat payloads and are hard to query.

```typescript
// GOOD
properties: { filterCount: number }

// BAD
properties: { filters: string[] }
```

### Bucket High-Cardinality Numeric Values

Metrics with high cardinality or wide variance (e.g., database size, cluster count, duration) are difficult to group by or segment in Amplitude when tracked as raw numbers. Break them into labeled intervals so they work as chart dimensions.

```go
// GOOD: bucketed for grouping and filtering.
func bucketDBSize(bytes int64) string {
    gb := bytes / (1 << 30)
    switch {
    case gb < 1:
        return "<1 GB"
    case gb < 10:
        return "1-10 GB"
    case gb < 100:
        return "10-100 GB"
    default:
        return "100+ GB"
    }
}
props["Database Size Range"] = bucketDBSize(sizeBytes)

// BAD: raw value creates thousands of unique groups.
props["Database Size Bytes"] = sizeBytes
```

```typescript
// GOOD: bucketed interval.
function bucketItemCount(count: number): string {
    if (count === 0) return '0';
    if (count <= 10) return '1-10';
    if (count <= 100) return '11-100';
    return '100+';
}
properties: { clusterCountRange: bucketItemCount(clusters) }

// BAD: raw number as a grouping dimension.
properties: { clusterCount: clusters }
```

Track the raw value alongside the bucket only if you need precise averages — use the bucket for segmentation, the raw value for aggregation.

### Avoid Dynamic Property Keys

Never iterate over a map or collection to generate property keys. This creates unbounded schema cardinality in Amplitude, making charts unusable.

```go
// BAD: each permission becomes a distinct property key.
for p, a := range req.GetPermissions() {
    props[p] = a.String()
}

// GOOD: track a fixed count instead.
props["Total Permissions"] = len(req.GetPermissions())
```

### Property Naming

- **Frontend:** Use `camelCase` for simple properties, `UPPER_SNAKE_CASE` for boolean flags that represent toggleable options (matching the existing pattern in `VULNERABILITY_REPORT_CREATED`).
- **Backend:** Use `Title Case` for Go property keys (e.g., `"My Feature Item Count"`), matching the existing convention in gatherers.

---

## Common Anti-Patterns

Real mistakes found in this codebase. Avoid these.

**Unbounded string properties (frontend):** Properties used for grouping must be union literals, not `string`. The type system catches this if you use it correctly.

```typescript
// BAD:  areaOfConcern: string
// GOOD: areaOfConcern: 'User workloads' | 'Platform' | 'All vulnerable images'

// BAD:  reportStatus: string
// GOOD: reportStatus: 'WAITING' | 'PREPARING' | 'GENERATED' | 'ERROR'

// BAD:  step: string
// GOOD: step: 'parameters' | 'clusters' | 'profiles' | 'review'
```

**Boolean literals (frontend):** Use `AnalyticsBoolean` (0/1), not `true`/`false`. Amplitude cannot aggregate booleans numerically.

```typescript
// BAD:  state: true | false; success: true | false
// GOOD: state: AnalyticsBoolean; success: AnalyticsBoolean
```

**Customer-identifiable values (backend):** Registry URLs and UUIDs leak customer data and create high cardinality.

```go
// BAD:  "Main Image": cluster.GetMainImage()  // exposes customer registry URL
// BAD:  "Cluster ID": cluster.GetId()          // raw UUID as event property

// GOOD: use WithClient(id, type, version) for entity identity — never as a property
// GOOD: extract only the image tag/version, not the full registry path
```

---

## Privacy and Data Redaction

**Never track customer-specific data.** The system automatically redacts URLs and search parameters, but you must also avoid it in event properties.

### Never Include

- Cluster names, namespace names, deployment names.
- Image names, registry URLs, CVE IDs tied to specific images.
- User names, email addresses, IP addresses.
- Policy names or custom policy content.
- Any string that could identify a specific customer environment.

### Safe to Include

- Counts and aggregates (number of clusters, number of CVEs).
- Predefined category values (severity levels, status enums).
- UI interaction metadata (which page, which button, which step).
- Feature flag states.
- Error types (categorized, not raw messages).

If you are unsure whether a value is safe, track only its category or a boolean indicating its presence, not the value itself.

---

## Checklist

Before submitting telemetry changes, verify:

- [ ] Event name follows `Title Case` naming convention.
- [ ] Event constant is exported and added to the `AnalyticsEvent` union type.
- [ ] Properties use bounded enums, not free-text strings.
- [ ] Boolean properties use `0 | 1`, not `true | false`.
- [ ] No customer-specific data in any property value.
- [ ] Filter tracking uses `searchCategoriesWithFilter` allowlist for tracked values.
- [ ] Backend properties use `Title Case` keys.
- [ ] Each event answers a specific analytical question.
- [ ] Backend gatherers placed in `central/<feature>/datastore/telemetry.go`, not `central/telemetry/gatherers/`.
- [ ] No dynamic property keys generated from map/collection iteration.
- [ ] Entity lifecycle events use `WithClient()` — raw entity IDs never appear as event properties.
- [ ] For dual-instrumented features: frontend tracks intent, backend tracks outcomes — no duplication.
- [ ] Manually verified event fires in development environment.
- [ ] Confirmed event appears in Segment debugger or Amplitude live view.
- [ ] Tested edge cases (error states, empty data, etc.) fire appropriate property values.

---

## What NOT to Instrument

- **Routine navigation** between pages (unless measuring a specific funnel).
- **Every button click** — only track meaningful actions that complete a workflow step.
- **Internal API calls** between microservices (Sensor-to-Central, Scanner-to-Central).
- **Debug or development actions** that do not happen in production.
- **High-frequency events** that would overwhelm Amplitude (e.g., every keystroke, every scroll).

---

## Quick Reference: Adding a Frontend Event

1. Add `export const MY_EVENT = 'My Event';` to `useAnalytics.ts`.
2. Add the event (with or without properties) to the `AnalyticsEvent` union type.
3. In your component: `const { analyticsTrack } = useAnalytics();` then call `analyticsTrack(...)`.
4. Run `npm run tsc` from `ui/apps/platform` to verify types.

## Quick Reference: Adding a Backend Event

1. For feature metrics: create `central/<feature>/datastore/telemetry.go` with a `Gather` function, register in `central/main.go`.
2. For API tracking: add an interceptor via `c.AddInterceptorFuncs(...)`.
3. For cross-cutting infrastructure metrics: add a gatherer in `central/telemetry/gatherers/`.
4. For one-off events: call `c.Track(...)` directly.
5. Run `go build ./central/...` to verify compilation.
