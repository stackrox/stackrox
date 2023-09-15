# ACS API Guidelines
An overview of the set of standardized guidelines and practices to ensure that the APIs across different 
projects and teams adhere to a common structure and design. This consistency simplifies the process of understanding 
and maintaining APIs. These guidelines help identify potential pitfalls and design choices that could lead to issues to 
help developers avoid common mistakes and build more robust APIs. When in doubt, It is encouraged to seek resolution
in slack channel [#forum-acs-api-design](https://redhat-internal.slack.com/archives/C05MMG2PP8A).

## Table of Contents

- [Naming Guidelines](#naming-guidelines)
  - [Decoupling](#decoupling)
  - [gRPC Service Name](#grpc-service-name)
  - [gRPC Method Name](#grpc-method-name)
  - [gRPC Message Name](#grpc-message-name)

## Decoupling

While certain data may exist in the database for internal purposes or other functionalities, it is important to 
exercise caution when deciding what data to make accessible through the API. The rule of thumb to follow is that the APIs 
**must not** expose data structures defined for internal purposes. Instead, define a data structure for the sole purpose of 
API use, generally in the same package where the API is defined. This applies to all APIs including the 
inter-component APIs.
- All services defined in the package [/proto/api/v2](https://github.com/stackrox/stackrox/blob/master/proto/api/v2)
**must not** import from the following packages:
  - [/proto/storage](https://github.com/stackrox/stackrox/blob/master/proto/storage) 
  - [/proto/internalapi](https://github.com/stackrox/stackrox/blob/master/proto/internalapi) 
  - [/proto/api/v1](https://github.com/stackrox/stackrox/blob/master/proto/api/v1)
- All new services and methods defined in the package [/proto/api/v1](https://github.com/stackrox/stackrox/blob/master/proto/api/v1) 
**must not** import from the following packages:
  - [/proto/storage](https://github.com/stackrox/stackrox/blob/master/proto/storage)
  - [/proto/internalapi](https://github.com/stackrox/stackrox/blob/master/proto/internalapi)
  - [/proto/api/v2](https://github.com/stackrox/stackrox/blob/master/proto/api/v2)
- All new messages defined in the package [/proto/internalapi](https://github.com/stackrox/stackrox/blob/master/proto/internalapi)
  **must not** import from the following packages:
  - [/proto/storage](https://github.com/stackrox/stackrox/blob/master/proto/storage)
  - [/proto/api/v1](https://github.com/stackrox/stackrox/blob/master/proto/api/v1)
  - [/proto/api/v2](https://github.com/stackrox/stackrox/blob/master/proto/api/v2)
- All services of the type `APIServiceWithCustomRoutes` **must not** import from the following packages:
  - [/proto/storage](https://github.com/stackrox/stackrox/blob/master/proto/storage) 
  - [/proto/internalapi](https://github.com/stackrox/stackrox/blob/master/proto/internalapi)
- Structs used by APIs, such as those defined in [/proto/api/](https://github.com/stackrox/stackrox/blob/master/proto/api/), 
[/proto/internalapi](https://github.com/stackrox/stackrox/blob/master/proto/internalapi), **must not** be 
written to database.

Exposing internal data structures, especially the ones representing and affecting database schema directly at 
the API can pose security risks. By carefully selecting and limiting the data exposed via the API, we can prevent 
unauthorized access and potential data breaches which can have compliance ramifications. 

Furthermore, exposing unnecessary data in the API can lead to overfetching, where the client receives more data 
than required. This inefficiency increases network traffic, consumes extra bandwidth, and impacts API performance. 
By limiting API data to what is actually needed by clients, we optimize resource usage and improve response times. 

Consider the following database/internal structures as an example data structure used in Central.

```
Deployment {
  string id
  string name
  repeated KeyValue labels
  repeated KeyValue annotations
  repeated Container containers
}
  
Container {
  string name
  string image_name
  repeated Volume volumes
}
```

Exposing the above data structure in the API will always lead to reading all the `Container` bytes from the database, 
even if the user does not need them. Instead, design the API such that users can request the information 
they need, as below:

```
// GetDeploymentRequest requests deployment information. Deployment metadata is requested by default.
GetDeploymentRequest {
  Options {
    bool get_container_spec `json: "getContainerSpec"`
  }
}

// Default response
GetDeploymentResponse {
  Metadata {
    string name `json: "name"`
    string id `json: "id"`
  }
}

// If `"getContainerSpec": true`
GetDeploymentResponse {
  Metadata {
    string name `json: "name"`
    string id `json: "name"`
    ...
  }
  ContainerSpec {
    string name `json: "name"`
    string image_name `json: "imageName"`
    ...
  }
}
```

As APIs evolve over time, adding or removing data fields becomes inevitable. By limiting the exposure of internal 
database structures, we decouple the API from the database schema, allowing us to modify the internal structures
without breaking existing client applications.

Consider the following database/internal structures as an example data structure exposed by APIs.
```
Alert {
  string id
  repeated Violation violations
  string policy_name
  string policy_description
  string policy_enforcement
}
```
Changing the above `Alert` data structure to the following structure may break the client application.
```
Alert {
  string id;
  repeated Violation violations
  
  Policy {
    string policy_name
    string policy_description
    string policy_enforcement
  }
} 
```

Keeping the API focused and concise simplifies maintenance efforts and reduces the chances of introducing bugs. 
A clean and manageable codebase improves the overall maintainability and stability of the API.

## Naming Guidelines

### gRPC Service Name
The service name **must** be unique and use a noun that generally refers to a resource or product component and 
**must** end with **Service** e.g. `DeploymentService`, `ReportService`, `ComplianceService`. Intuitive and well-known 
short forms or abbreviations **may** be used in some cases (and could even be preferable) for succinctness 
e.g. `ReportConfigService`, `RbacService`. 
All gRPC methods grouped into a single service **must** generally pertain to the primary resource of the service.

### gRPC Method Name
Methods **should** be named such that they provide insights into the functionality.

Let us look at a few examples.`StartComplianceScan`, `RunComplianceScan`, and `GetComplianceScan` are not the same. 

- `StartComplianceScan` **should** return without waiting for the compliance scan to complete. 
- `RunComplianceScan` is ambiguous because it is unclear if the call waits for the scan to complete. 
The ambiguity can be removed by adding a field to the request that helps clarify the expectation 
e.g. `bool wait_for_scan_completion` if set to `true` informs the method to wait for the compliance 
scan to complete. However, for long-running processes, it is **recommended** to create a job that 
finishes the process asynchronously and return the job ID to the users which can be tracked via 
dedicated job tracking method.
- `GetComplianceScan` **should** not run a compliance scan but only fetches a stored one.

Typically, the method name **should** follow the _VerbNoun_ convention.

| Verb   | Noun           | Method name         |
|--------|----------------|---------------------|
| List   | Deployment     | `ListDeployments`   |
| Get    | Deployment     | `GetDeployment`     |
| Update | Deployment     | `UpdateDeployment`  |
| Delete | Deployment     | `DeleteDeployment`  |
| Notify | Violation      | `NotifyViolation`   |
| Run    | ComplianceScan | `RunComplianceScan` |

It is **recommended** that the verbs be imperative instead of inquisitive. Generally, the noun **should** be the resource type. 
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
<tr>
<td>

```
GetBaselineGeneratedNetworkPolicyForDeployment
```

</td>
<td>

`GetDeploymentBaselineNetworkPolicy` or merely `GetBaselineNetworkPolicy` if the concept of baselines applies
to deployments only.

The following example demonstrates design if that concept of baselines could apply to multiple resource types.

```
GetBaselineNetworkPolicy

GetBaselineNetworkPolicyRequest {
  oneof resource {
    string deployment_id;
    string cluster_id;
  }
}

```

</td>
</tr>
</table>

### gRPC Message Name
The request and response messages **must** be named after method names with suffix `Request` and `Response` unless 
the request/response type is an empty message. Generally, resource type as response message **should** be avoided 
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
GetDeploymentImageScanResponse {
  Image image;
}
```
or,
```
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

Be explicit about conveying the specific purpose of fields e.g. instead of `expires_on` use `expiry_date`(/`timestamp`)
as it informs users if the field returns the date portion of the timestamp or the full timestamp, and use 
`network_data_start_time` instead of `network_data_since` for a similar reason. The fields should convey their purpose without 
requiring users to read the documentation.
