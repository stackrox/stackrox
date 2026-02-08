# TestPod flake investigation (pod.events empty)

## Context / symptom

`TestPod` sometimes fails because GraphQL `pod(id).events` does not include expected process executions (e.g. `/bin/date`, `/bin/sleep`), even though other signals indicate those processes executed and were ingested.

The repeated failure signature in CI logs looked like:

- `Pod: required processes: [/usr/sbin/nginx /bin/sh /bin/date /bin/sleep] not found in events: [...]`
- Often `pod.events` contained only a subset (sometimes even empty).

## What the latest diagnostics prove (high confidence)

From the final-attempt diagnostics in `result` (Dec 12 run), we have **strong, consistent evidence**:

- **Sensor is healthy and not restarting**
  - `Central cluster health: sensor=HEALTHY collector=HEALTHY overall=HEALTHY`
  - K8s sensor container `restart=0`

- **Central has ProcessIndicators for the deployment, pod, and containers**
  - ProcessService groups include `/bin/date`, `/bin/sleep`, `/usr/sbin/nginx`, etc.
  - Raw `ProcessIndicator` samples show the expected identity fields:
    - `podId="<pod name>"`
    - `podUid="<central pod UUID>"`
    - `container="1st"|"2nd"`

- **ProcessIndicators are searchable by every relevant identifier**
  - Total:
    - `Deployment ID` -> **23**
    - `Deployment ID + Pod ID(<podName>)` -> **23**
    - `Deployment ID + Pod UID(<podUUID>)` -> **23**
    - `Deployment ID + Pod Name(<podName>)` -> **23**
  - Container-scoped (this is the key “strict” proof):
    - `... + Container Name: 1st` -> **20** (for Pod ID / Pod UID / Pod Name variants)
    - `... + Container Name: 2nd` -> **3** (for Pod ID / Pod UID / Pod Name variants)

- **GraphQL can see process activity at the deployment level**
  - `deployment(id).processActivityCount = 23`
  - `deployment(id).groupedProcesses` returns expected exec paths (sample includes `/bin/date`, `/bin/sleep`, `/usr/sbin/nginx`, …)

## What is broken (and where it breaks)

Even with the above, GraphQL remains empty at the pod/container event level:

- **GraphQL `pod(id).events` is empty**
  - `GraphQL pod.events count=0 names=[]`

- **GraphQL `groupedContainerInstances` finds the groups but emits zero events**
  - Queried both ways:
    - `Deployment ID + Pod ID(uuid)` -> `groups=2`, but `1st.events=0` and `2nd.events=0`
    - `Deployment ID + Pod Name` -> `groups=2`, but `1st.events=0` and `2nd.events=0`

## Running conclusion

This is **not** an ingestion gap and **not** a Sensor-health issue.

It is very likely a **GraphQL-layer bug/mismatch in “process activity event” resolution for pod/container scopes**:

- Deployment-level GraphQL (`deployment.groupedProcesses`, `deployment.processActivityCount`) works and sees the indicators.
- The same indicators are searchable via ProcessService for deployment+pod and deployment+pod+container.
- Yet pod/container event resolvers return empty without error.

In other words: **GraphQL event resolution/filtering is diverging from ProcessIndicator search/count semantics.**

## Relevant backend code paths (starting points)

These resolvers assemble `DeploymentEvent`s and are the likely root-cause area:

- `central/graphql/resolvers/pods.go`
  - `podResolver.Events()` calls `processActivityEvents()` which builds a query:
    - `DeploymentID == pod.deploymentId`
    - `PodID == pod.name`
  - Then calls `root.getProcessActivityEvents(ctx, query)`

- `central/graphql/resolvers/container_instances.go`
  - `populateEvents()` calls `processActivityEvents()` which builds a query:
    - `DeploymentID == group.deploymentID`
    - `PodID == group.podID.Name`
    - `ContainerName == group.name`
  - Then calls `root.getProcessActivityEvents(ctx, query)`

- `central/graphql/resolvers/deploymentevents.go`
  - `Resolver.getProcessActivityEvents(ctx, query)`:
    - checks `readDeploymentExtensions(ctx)`
    - calls `ProcessIndicatorStore.SearchRawProcessIndicators(ctx, query)`
    - converts results into `ProcessActivityEventResolver`s

Given the data and counts, the divergence is likely in:

- the exact query produced in these GraphQL paths, OR
- the search-field mapping for pod/container constraints when executed via GraphQL’s store/search layer, OR
- a subtle scoping/permission behavior applied in GraphQL contexts.

## Suggested next debugging steps (if continuing)

1. **Compare query protos**:
   - Log (or temporarily expose via diagnostics) the exact `*v1.Query` used by GraphQL `getProcessActivityEvents` for pods/containers.
   - Compare it to `search.ParseQuery("Deployment ID: ... + Pod ID: ... + Container Name: ...")` output (which we know counts correctly).

2. **Verify field mapping in the ProcessIndicators schema**:
   - Confirm that `search.PodID`, `search.PodUID`, and `search.ContainerName` map to the expected columns (`podid`, `poduid`, `containername`) for the process indicators schema.

3. **Check for GraphQL-only scoping differences**:
   - `readDeploymentExtensions(ctx)` is checked in GraphQL before searching indicators; ensure it doesn’t apply unexpected scoping or filter-out behavior for these queries.


