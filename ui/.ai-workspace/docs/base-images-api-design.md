# Base Images API Design

## Overview

This document specifies the API design for the Base Images feature in StackRox ACS. The API follows StackRox's existing patterns, particularly mirroring the Watched Images implementation.

**Last Updated:** 2025-10-13
**Status:** Design proposal

---

## Design Principles

### Key Patterns from StackRox Codebase

1. **Protocol Buffers First** - All APIs defined in `.proto` files
2. **RawQuery for Filtering** - Search using format: `"Field:value+Field:value"`
3. **Pagination** - Embedded in RawQuery with `limit`, `offset`, `sortOption`
4. **gRPC + REST** - gRPC service with HTTP annotations for REST mapping
5. **Watched Images Pattern** - Simple CRUD at `/v1/baseimages` (lowercase, no hyphens)

---

## Proto Definitions

### Storage Messages

**File:** `proto/storage/base_image.proto`

```protobuf
syntax = "proto3";

package storage;

import "google/protobuf/timestamp.proto";

option go_package = "./storage;storage";
option java_package = "io.stackrox.proto.storage";

// BaseImage represents a tracked base image (stored in database)
message BaseImage {
  string name = 1; // @gotags: sql:"pk"
  google.protobuf.Timestamp created_at = 2;
}

// BaseImageDetails contains enriched information about a base image
// This is computed/aggregated data returned in API responses
message BaseImageDetails {
  string id = 1;
  string name = 2;
  string normalized_name = 3;
  ScanStatus scanning_status = 4;
  google.protobuf.Timestamp last_scanned = 5;
  google.protobuf.Timestamp created_at = 6;

  message CVECount {
    int32 critical = 1;
    int32 high = 2;
    int32 medium = 3;
    int32 low = 4;
    int32 total = 5;
  }
  CVECount cve_count = 7;

  int32 deployment_count = 8;
  int32 image_count = 9;
  int32 last_base_layer_index = 10;

  enum ScanStatus {
    UNSET = 0;
    IN_PROGRESS = 1;
    COMPLETED = 2;
    FAILED = 3;
  }
}

// BaseImageInfo is embedded in Image messages to indicate detected base
message BaseImageInfo {
  string name = 1;                  // Detected base image name (e.g., "ubuntu:22.04")
  bool is_managed = 2;              // True if user is tracking this base
  int32 last_layer_index = 3;      // Last layer index belonging to base (0-indexed)
  string base_image_id = 4;        // ID of tracked base image (if is_managed=true)
}
```

**Modifications to:** `proto/storage/image.proto`

```protobuf
// Add to existing Image or ImageV2 message
message Image {
  // ... existing fields ...

  BaseImageInfo base_image = 20; // Use next available tag number

  // ... rest of existing fields ...
}
```

---

### API Service

**File:** `proto/api/v1/base_image_service.proto`

```protobuf
syntax = "proto3";

package v1;

import "api/v1/empty.proto";
import "api/v1/search_service.proto";
import weak "google/api/annotations.proto";
import "storage/base_image.proto";
import "google/protobuf/timestamp.proto";

option go_package = "./api/v1;v1";
option java_package = "io.stackrox.proto.api.v1";

// ============================================================================
// Request/Response Messages
// ============================================================================

message AddBaseImageRequest {
  // The name of the base image to track.
  // This must be fully qualified, including a tag.
  // Examples: "ubuntu:22.04", "docker.io/library/alpine:3.18"
  string name = 1;
}

message AddBaseImageResponse {
  string id = 1;
  string name = 2;
  string normalized_name = 3;
  storage.BaseImageDetails.ScanStatus scanning_status = 4;
  string message = 5;
}

message RemoveBaseImageRequest {
  // The name of the base image to stop tracking.
  // Should match the name of a previously tracked base image.
  string name = 1;
}

message GetBaseImagesResponse {
  repeated storage.BaseImageDetails base_images = 1;
}

message GetBaseImageRequest {
  string id = 1;
}

message GetBaseImageVulnerabilitiesRequest {
  string id = 1;
  RawQuery query = 2;  // For filtering by severity, fixability, component, etc.
}

message GetBaseImageVulnerabilitiesResponse {
  repeated BaseImageVulnerability vulnerabilities = 1;
  int32 total_count = 2;
}

message BaseImageVulnerability {
  string cve_id = 1;
  string severity = 2;
  float cvss_score = 3;
  string summary = 4;
  string fixed_by = 5;
  repeated Component components = 6;

  message Component {
    string name = 1;
    string version = 2;
    int32 layer_index = 3;
  }
}

message GetBaseImageImagesRequest {
  string id = 1;
  RawQuery query = 2;  // For pagination, sorting
}

message GetBaseImageImagesResponse {
  repeated BaseImageImage images = 1;
  int32 total_count = 2;
}

message BaseImageImage {
  string image_id = 1;
  string name = 2;
  string sha = 3;
  google.protobuf.Timestamp last_scanned = 4;
  CVECounts cve_count = 5;
  int32 deployment_count = 6;

  message CVECounts {
    int32 critical = 1;
    int32 high = 2;
    int32 medium = 3;
    int32 low = 4;
    int32 total = 5;
    int32 base_image_cves = 6;
    int32 application_layer_cves = 7;
  }
}

message GetBaseImageDeploymentsRequest {
  string id = 1;
  RawQuery query = 2;  // For filtering by cluster, namespace, pagination
}

message GetBaseImageDeploymentsResponse {
  repeated BaseImageDeployment deployments = 1;
  int32 total_count = 2;
}

message BaseImageDeployment {
  string deployment_id = 1;
  string name = 2;
  string namespace = 3;
  string cluster = 4;
  string image = 5;
  CVECounts cve_count = 6;
  int64 risk_priority = 7;

  message CVECounts {
    int32 critical = 1;
    int32 high = 2;
    int32 medium = 3;
    int32 low = 4;
  }
}

// ============================================================================
// Service Definition
// ============================================================================

service BaseImageService {
  // AddBaseImage marks a base image name to be tracked.
  // This will initiate a scan of the base image if it hasn't been scanned recently.
  rpc AddBaseImage(AddBaseImageRequest) returns (AddBaseImageResponse) {
    option (google.api.http) = {
      post: "/v1/baseimages"
      body: "*"
    };
  }

  // RemoveBaseImage marks a base image name to no longer be tracked.
  // It returns successfully if the base image is no longer being tracked
  // after the call, irrespective of whether it was already being tracked.
  rpc RemoveBaseImage(RemoveBaseImageRequest) returns (Empty) {
    option (google.api.http) = {delete: "/v1/baseimages"};
  }

  // GetBaseImages returns the list of base images that are currently
  // being tracked, with enriched metadata (CVE counts, deployment counts, etc.).
  rpc GetBaseImages(RawQuery) returns (GetBaseImagesResponse) {
    option (google.api.http) = {get: "/v1/baseimages"};
  }

  // GetBaseImage returns detailed information for a specific tracked base image.
  rpc GetBaseImage(GetBaseImageRequest) returns (storage.BaseImageDetails) {
    option (google.api.http) = {get: "/v1/baseimages/{id}"};
  }

  // GetBaseImageVulnerabilities returns all vulnerabilities found in the base image layers.
  rpc GetBaseImageVulnerabilities(GetBaseImageVulnerabilitiesRequest)
      returns (GetBaseImageVulnerabilitiesResponse) {
    option (google.api.http) = {get: "/v1/baseimages/{id}/vulnerabilities"};
  }

  // GetBaseImageImages returns all application images that use this base image.
  rpc GetBaseImageImages(GetBaseImageImagesRequest)
      returns (GetBaseImageImagesResponse) {
    option (google.api.http) = {get: "/v1/baseimages/{id}/images"};
  }

  // GetBaseImageDeployments returns all deployments running images that use this base.
  rpc GetBaseImageDeployments(GetBaseImageDeploymentsRequest)
      returns (GetBaseImageDeploymentsResponse) {
    option (google.api.http) = {get: "/v1/baseimages/{id}/deployments"};
  }
}
```

---

## REST API Specification

### 1. Add Base Image

**Endpoint:** `POST /v1/baseimages`

**Purpose:** Register a new base image to track

**Request Body:**
```json
{
  "name": "ubuntu:22.04"
}
```

**Response:** `200 OK`
```json
{
  "id": "base-image-sha256:abc123...",
  "name": "ubuntu:22.04",
  "normalizedName": "docker.io/library/ubuntu:22.04",
  "scanningStatus": "IN_PROGRESS",
  "message": "Base image added and scanning initiated"
}
```

**Error Responses:**
- `400 Bad Request` - Invalid image name format
- `409 Conflict` - Base image already being tracked
- `500 Internal Server Error` - Server error

---

### 2. Remove Base Image

**Endpoint:** `DELETE /v1/baseimages?name=<name>`

**Purpose:** Remove a base image from tracking

**Query Parameters:**
- `name` (required): The name of the base image to untrack

**Example:**
```
DELETE /v1/baseimages?name=ubuntu:22.04
```

**Response:** `200 OK`
```json
{}
```

**Notes:**
- Returns success even if the base image wasn't being tracked
- Does not delete scan data, just stops tracking

---

### 3. List Base Images

**Endpoint:** `GET /v1/baseimages`

**Purpose:** Retrieve all tracked base images with summary metrics

**Query Parameters (RawQuery format):**
- `query` - Search query string (e.g., `"Base Image:ubuntu"`)
- `pagination.limit` - Number of results per page (default: 50)
- `pagination.offset` - Starting offset for pagination (default: 0)
- `pagination.sortOption.field` - Field to sort by (e.g., `"name"`, `"cveCount"`, `"lastScanned"`)
- `pagination.sortOption.reversed` - Sort in descending order (default: false)

**Example:**
```
GET /v1/baseimages?pagination.limit=20&pagination.sortOption.field=cveCount&pagination.sortOption.reversed=true
```

**Response:** `200 OK`
```json
{
  "baseImages": [
    {
      "id": "base-image-sha256:abc123...",
      "name": "ubuntu:22.04",
      "normalizedName": "docker.io/library/ubuntu:22.04",
      "scanningStatus": "COMPLETED",
      "lastScanned": "2025-10-13T10:30:00Z",
      "createdAt": "2025-10-10T08:00:00Z",
      "cveCount": {
        "critical": 5,
        "high": 12,
        "medium": 23,
        "low": 8,
        "total": 48
      },
      "deploymentCount": 12,
      "imageCount": 15,
      "lastBaseLayerIndex": 5
    }
  ]
}
```

---

### 4. Get Base Image Details

**Endpoint:** `GET /v1/baseimages/{id}`

**Purpose:** Get detailed information about a specific base image

**Path Parameters:**
- `id` (required): Base image ID

**Example:**
```
GET /v1/baseimages/base-image-sha256:abc123...
```

**Response:** `200 OK`
```json
{
  "id": "base-image-sha256:abc123...",
  "name": "ubuntu:22.04",
  "normalizedName": "docker.io/library/ubuntu:22.04",
  "scanningStatus": "COMPLETED",
  "lastScanned": "2025-10-13T10:30:00Z",
  "createdAt": "2025-10-10T08:00:00Z",
  "cveCount": {
    "critical": 5,
    "high": 12,
    "medium": 23,
    "low": 8,
    "total": 48
  },
  "deploymentCount": 12,
  "imageCount": 15,
  "lastBaseLayerIndex": 5
}
```

**Error Responses:**
- `404 Not Found` - Base image not found or not tracked

---

### 5. Get Base Image Vulnerabilities

**Endpoint:** `GET /v1/baseimages/{id}/vulnerabilities`

**Purpose:** List all CVEs found in the base image layers

**Path Parameters:**
- `id` (required): Base image ID

**Query Parameters (RawQuery format):**
- `query` - Filter query (e.g., `"Severity:CRITICAL+Fixable:true"`)
- `pagination.limit` - Page size
- `pagination.offset` - Page offset

**Supported Filter Fields:**
- `Severity`: CRITICAL, IMPORTANT, MODERATE, LOW
- `Fixable`: true, false
- `Component`: Component name
- `CVE`: CVE ID

**Example:**
```
GET /v1/baseimages/base-image-sha256:abc123.../vulnerabilities?query=Severity:CRITICAL,IMPORTANT&pagination.limit=50
```

**Response:** `200 OK`
```json
{
  "vulnerabilities": [
    {
      "cveId": "CVE-2024-1234",
      "severity": "CRITICAL",
      "cvssScore": 9.8,
      "summary": "Buffer overflow in libssl",
      "fixedBy": "1.2.3-4ubuntu1",
      "components": [
        {
          "name": "libssl1.1",
          "version": "1.2.3-3ubuntu1",
          "layerIndex": 2
        }
      ]
    }
  ],
  "totalCount": 48
}
```

---

### 6. Get Base Image Images

**Endpoint:** `GET /v1/baseimages/{id}/images`

**Purpose:** List all application images that use this base image

**Path Parameters:**
- `id` (required): Base image ID

**Query Parameters (RawQuery format):**
- `query` - Filter query (e.g., `"Image:myapp"`)
- `pagination.limit` - Page size
- `pagination.offset` - Page offset
- `pagination.sortOption.field` - Sort by field
- `pagination.sortOption.reversed` - Sort descending

**Example:**
```
GET /v1/baseimages/base-image-sha256:abc123.../images?pagination.limit=20
```

**Response:** `200 OK`
```json
{
  "images": [
    {
      "imageId": "sha256:def456...",
      "name": "myapp:v1.2.3",
      "sha": "sha256:def456...",
      "lastScanned": "2025-10-13T10:00:00Z",
      "cveCount": {
        "critical": 7,
        "high": 15,
        "medium": 28,
        "low": 10,
        "total": 60,
        "baseImageCves": 48,
        "applicationLayerCves": 12
      },
      "deploymentCount": 3
    }
  ],
  "totalCount": 15
}
```

---

### 7. Get Base Image Deployments

**Endpoint:** `GET /v1/baseimages/{id}/deployments`

**Purpose:** List all deployments running images that use this base

**Path Parameters:**
- `id` (required): Base image ID

**Query Parameters (RawQuery format):**
- `query` - Filter query (e.g., `"Cluster:prod+Namespace:default"`)
- `pagination.limit` - Page size
- `pagination.offset` - Page offset

**Supported Filter Fields:**
- `Cluster`: Cluster name
- `Namespace`: Namespace name
- `Deployment`: Deployment name
- `Severity`: Filter by CVE severity in deployment

**Example:**
```
GET /v1/baseimages/base-image-sha256:abc123.../deployments?query=Cluster:prod-us-west
```

**Response:** `200 OK`
```json
{
  "deployments": [
    {
      "deploymentId": "deploy-789",
      "name": "web-frontend",
      "namespace": "production",
      "cluster": "prod-us-west-1",
      "image": "myapp:v1.2.3",
      "cveCount": {
        "critical": 7,
        "high": 15,
        "medium": 28,
        "low": 10
      },
      "riskPriority": 85
    }
  ],
  "totalCount": 12
}
```

---

### 8. Enhanced Image Endpoint

**Endpoint:** `GET /v1/images/{id}` (existing, enhanced)

**Purpose:** Get image details with base image information

**Additions to Response:**
```json
{
  "id": "sha256:def456...",
  "name": { "fullName": "myapp:v1.2.3" },
  // ... existing fields ...
  "baseImage": {
    "name": "ubuntu:22.04",
    "isManaged": true,
    "lastLayerIndex": 5,
    "baseImageId": "base-image-sha256:abc123..."
  },
  "scan": {
    "components": [
      {
        "name": "libssl1.1",
        "layerIndex": 2,
        // Component is in base if layerIndex <= baseImage.lastLayerIndex
        "vulns": [
          {
            "cve": "CVE-2024-1234",
            // Can compute: isFromBaseImage = (component.layerIndex <= baseImage.lastLayerIndex)
          }
        ]
      }
    ]
  }
}
```

---

## Query String Format (RawQuery)

StackRox uses a structured query format for filtering and searching:

**Format:** `"Field:value1,value2+Field2:value3"`

**Examples:**

```
# Filter by severity
Severity:CRITICAL,HIGH

# Filter by multiple criteria
Severity:CRITICAL+Fixable:true

# Search for specific image
Image:ubuntu

# Cluster and namespace filtering
Cluster:prod-us-west+Namespace:default

# CVE filtering
CVE:CVE-2024-1234
```

**Pagination is separate:**
```
?query=Severity:CRITICAL&pagination.limit=50&pagination.offset=100
```

**Sorting:**
```
?pagination.sortOption.field=name&pagination.sortOption.reversed=true
```

---

## Implementation Checklist

### Backend

- [ ] Create `proto/storage/base_image.proto`
- [ ] Add `BaseImageInfo` to `proto/storage/image.proto`
- [ ] Create `proto/api/v1/base_image_service.proto`
- [ ] Generate Go code: `make proto-generated-srcs`
- [ ] Create PostgreSQL schema for `base_images` table
- [ ] Implement BaseImageService gRPC service
- [ ] Implement datastore layer (CRUD for base_images)
- [ ] Implement base image detection logic
- [ ] Add layer analysis to compute `lastBaseLayerIndex`
- [ ] Update image enrichment to populate `baseImage` field
- [ ] Add search/filter support for base image queries
- [ ] Write unit tests for base image detection
- [ ] Write integration tests for API endpoints

### Frontend

- [ ] Generate TypeScript types from proto
- [ ] Create `baseImageService.ts` API client
- [ ] Implement Base Images list page
- [ ] Implement Base Image detail page with tabs (CVEs, Images, Deployments)
- [ ] Add "Track this base image" button to Image Details page
- [ ] Update Image Details to show base/app layer distinction
- [ ] Add base image filters to overview page
- [ ] Add base image column to images table (optional)

### Documentation

- [ ] API documentation (Swagger/OpenAPI)
- [ ] User guide for base images feature
- [ ] Migration guide (if needed)

---

## Security Considerations

1. **Access Control**
   - Base image management requires `WRITE` permission on `WatchedImage` resource (reuse existing)
   - Reading base images requires `READ` permission
   - Consider separate permission if needed: `BaseImage` resource

2. **Validation**
   - Validate image name format (must include tag, no SHA)
   - Sanitize user input to prevent injection
   - Rate limit scanning requests

3. **Resource Limits**
   - Limit number of base images per tenant
   - Prevent scanning of excessively large images
   - Implement timeout for base image scans

---

## Performance Considerations

1. **Caching**
   - Cache base image scan results
   - Cache computed metrics (CVE counts, deployment counts)
   - Invalidate cache when base image is rescanned

2. **Indexing**
   - Index `base_images.name` for fast lookups
   - Index `images.base_image_id` for joins
   - Consider materialized views for aggregate queries

3. **Pagination**
   - Always use pagination for list endpoints
   - Default to reasonable page sizes (50-100)
   - Support cursor-based pagination for large result sets

---

## Testing Strategy

### Unit Tests
- Base image detection logic
- Layer boundary computation
- CVE attribution (base vs app)
- Query parsing and filtering

### Integration Tests
- Full CRUD cycle for base images
- Image scanning with base detection
- Multi-stage build handling
- Filter and pagination accuracy

### E2E Tests
- Add base image via UI
- View base image detail page
- Filter images by base
- Track/untrack workflow

---

## Future Enhancements

**Phase 2:**
- Multi-stage build support (track multiple FROM instructions)
- Base image version comparison
- Automated base image update recommendations
- Base image CVE trending over time

**Phase 3:**
- Policy integration for base image governance
- Base image approval workflows
- Automated Dockerfile generation with recommended bases
- Base image vulnerability SLAs and alerting

---

## References

- [Watched Images Implementation](../central/watchedimage/)
- [Image Service Proto](../proto/api/v1/image_service.proto)
- [Search Query Format](../proto/api/v1/search_service.proto)
- [Base Images UX Design](./base-images-ux-design.md)
- [Base Images UX Analysis](./base-images-ux-analysis.md)

---

*Document created: 2025-10-13*
*Last updated: 2025-10-13*
*Status: Design proposal - Ready for team review*
