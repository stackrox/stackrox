# ACS API Guidelines
An overview of the set of standardized guidelines and practices to ensure that the APIs across different 
projects and teams adhere to a common structure and design. This consistency simplifies the process of understanding 
and maintaining APIs. The guidelines helps identify potential pitfalls and design choices that could lead to issues to 
help developers avoid common mistakes and build more robust APIs. When in doubt, It is encouraged to seek resolution
in slack channel [#forum-acs-api-design](https://redhat-internal.slack.com/archives/C05MMG2PP8A).

## Table of Contents

- [Naming Guidelines](#naming-guidelines)
  - [Service Name](#service-name)
  - [Method Name](#method-name)
  - [Message Name](#message-name)

##Naming Guidelines
### Service Name
Service name **must** be unique and should use a noun that generally refers to a resource or product component 
e.g. `DeploymentService`, `ReportService`, `ComplianceService`. Intuitive and well-known short forms or 
abbreviations may be used in some cases (and could even be preferable) for succinctness 
e.g. `ReportConfigService`, `RbacService`. 
All RPC methods grouped into a single service must generally pertain to the primary resource of the service.

All service defined in versioned package `/proto/api/v2` **must not** import from 
`/proto/storage` and `/proto/internalapi` packages.
All new RPC methods defined in versioned package `/proto/api/v1` **must not** import from
`/proto/storage` and `/proto/internalapi` packages.

### Method Name
Methods (and URLs) should be named such that they provide insights into the functionality. Typically, the method 
name should follow the _VerbNoun_ convention. For example, `StartComplianceScan`, `RunComplianceScan`, 
and `GetComplianceScan` are not the same. `StartComplianceScan` must return without waiting for the compliance 
scan to complete. `RunComplianceScan` may or may not wait for a compliance scan; the method name is ambiguous 
and should be avoided if it starts a process and returns without waiting. `GetComplianceScan` should not run 
a compliance scan but only fetches a stored one.

| Verb   | Noun           | Method name         |
|--------|----------------|---------------------|
| List   | Deployment     | `ListDeployments`   |
| Get    | Deployment     | `GetDeployment`     |
| Update | Deployment     | `UpdateDeployment`  |
| Delete | Deployment     | `DeleteDeployment`  |
| Notify | Violation      | `NotifyViolation`   |
| Run    | ComplianceScan | `RunComplianceScan` |

It is **recommended** that the verbs be imperative instead of inquisitive. Generally, the noun should be the resource type. 
In some cases, the noun portion could be composed of multiple nouns e.g. `GetVulnerabilityDeferralState`, `RunPolicyScan`.

| Inquisitive               | Imperative                      |
|---------------------------|---------------------------------|
| `IsRunComplete`           | `GetRunStatus`                  |
| `IsAdmin`                 | `GetUserRole`                   |
| `IsVulnerabilityDeferred` | `GetVulnerabilityDeferralState` |

The noun portion of methods that act on a single resource **must** be singular e.g. `GetDeployment`. Those methods that 
act on the collection of resources **must** be plural e.g. `ListDeployments`, `DeleteDeployments`. Avoid prepositions 
(e.g. for, by) in method names as much as possible. Typically, this can be addressed by using a distinct verb, 
adding a field to the request message, or restructuring _VerbNoun_.

<table>
<tr>
<td><b>Instead of</b></td><td><b>Use</b></td>
</tr>
<tr>
<td>

`GetBaselineGeneratedNetworkPolicyForDeployment`

</td>

<td>

```
GenerateDeploymentNetworkPolicy

GenerateDeploymentNetworkPolicyRequest {
  bool from_baseline;  
  bool from_network_flows;
}
```

</td>
</tr>

<tr>
<td>

`RunPolicyScanForDeployment`  

</td>
<td>

`RunDeploymentPolicyScan`

</td>
</tr>

<tr>

<td>

`DeleteDeploymentsByQuery`

</td>
<td>

```
DeleteDeployments

DeleteDeploymentsRequest {
  string query; 
}
```

</td>
</tr>
</table>

### Message Name
The request and response messages **must** be named after method names with suffix `Request` and `Response` unless 
the request/response type is an empty message. Generally, resource type as response message should be avoided 
e.g. use `GetDeploymentResponse` response instead of `Deployment`. This allows augmenting the response with 
supplemental information in the future.

| Verb                      | Noun           | Method name         | Request message            | Response message            |
|---------------------------|----------------|---------------------|----------------------------|-----------------------------|
| List                      | Deployment     | `ListDeployments`   | `ListDeploymentRequest`    | `ListDeploymentResponse`    |
| Get                       | Deployment     | `GetDeployment`     | `GetDeploymentRequest`     | `GetDeploymentResponse`     |   
| Update                    | Deployment     | `UpdateDeployment`  | `UpdateDeploymentRequest`  | `UpdateDeploymentResponse`  |  
| Delete                    | Deployment     | `DeleteDeployment`  | `DeleteDeploymentRequest`  | `google.protobuf.Empty`     |     
| Get                       | ReportStatus   | `GetReportStatus`   | `GetReportStatusRequest`   | `GetReportStatusResponse`   |  
| Run                       | ComplianceScan | `RunComplianceScan` | `RunComplianceScanRequest` | `RunComplianceScanResponse` | 

Avoid prepositions as much as possible (e.g. “for”, “with”; `DeploymentWithProcessInfo`, `DeploymentWithImageScan`).
In case such a need arises, add a field to the request message and response message.

<table>
<tr>
<td><b>Instead of</b></td><td><b>Use</b></td>
</tr>
<tr>
<td>

`GetDeploymentWithImageScanRequest`

</td>
<td>

```
GetDeploymentRequest {
  bool with_image_scan;
}
```

</td>
</tr>
<tr>
<td>

`GetDeploymentWithImageScanResponse`

</td>
<td>

```
GetDeploymentWithImageScanResponse
GetDeploymentResponse {
  Deployment deployment;
  Image image;
}
```

</td>
</tr>
<tr>
<td>

`RunPolicyScanForDeploymentRequest`

</td>
<td>

`RunDeploymentPolicyScanRequest`

</td>
</tr>
</table>

All fields in the message **must** be lowercase and underscore separated names. The JSON names for the fields are 
autogenerated by the proto compiler. By default, field names are converted to camel case notation.

| Proto field name           | JSON field name        |
|----------------------------|------------------------|
| `network_data_start_time`  | `networkDataStartTime` |
| `expiry_date`              | `expiryDate`           |

Be explicit about conveying the specific purpose of fields e.g. instead of `expires_on`
use `expiry_date`(/`timestamp`), instead of `network_data_since` use `network_data_start_time`.
