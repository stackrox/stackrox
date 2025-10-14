# Base Images UI Prototype Requirements

## Overview

Quick and dirty MVP prototype for Base Images feature - UI only with hardcoded data.

**Goal:** Validate the UX approach with stakeholders before building full backend integration.

**Scope:** Frontend only, all data hardcoded, focus on core user flows.

---

## Out of Scope (For Prototype)

- ❌ Backend API integration
- ❌ Real scanning or data fetching
- ❌ Authentication/permissions
- ❌ Error handling beyond basic validation
- ❌ Full accessibility compliance (nice to have, not MVP)
- ❌ Advanced filtering (just basic search/filter)
- ❌ Pagination (show all hardcoded items)

---

## Core User Flows to Demonstrate

### Flow 1: Discover and Track a Base Image
1. User views Image Details page for an application image
2. Sees detected base image name (e.g., "ubuntu:22.04")
3. Clicks "Track this base image" button
4. Toast notification confirms tracking
5. Button changes to "Tracked" badge with link to Base Images view

### Flow 2: View Base Images List
1. User navigates to Vulnerabilities → Base Images tab
2. Sees table of tracked base images with:
   - Name
   - Scan status
   - CVE counts (by severity)
   - Image count (how many app images use this base)
   - Deployment count
3. Can click on a base image to view details

### Flow 3: View Base Image Details
1. User clicks on a base image from the list
2. Detail page opens with three tabs:
   - **CVEs tab** (default): Table of CVEs in the base image
   - **Images tab**: Application images using this base
   - **Deployments tab**: Deployments running those images
3. Can navigate between tabs
4. Can click links to navigate to related pages (image details, deployment details)

### Flow 4: See Base vs App Layer Distinction
1. User views Image Details page
2. Sees summary cards showing:
   - Base Image CVEs (count by severity)
   - Application Layer CVEs (count by severity)
3. In CVE table, sees "Layer Type" column with badges:
   - "Base Image" (blue badge)
   - "Application" (green badge)

---

## Hardcoded Data Structure

### Tracked Base Images (3-4 examples)

```typescript
const MOCK_BASE_IMAGES = [
  {
    id: 'base-image-1',
    name: 'ubuntu:22.04',
    normalizedName: 'docker.io/library/ubuntu:22.04',
    scanningStatus: 'COMPLETED',
    lastScanned: '2025-10-13T10:30:00Z',
    createdAt: '2025-10-10T08:00:00Z',
    cveCount: {
      critical: 5,
      high: 12,
      medium: 23,
      low: 8,
      total: 48
    },
    imageCount: 15,
    deploymentCount: 12,
    lastBaseLayerIndex: 5
  },
  {
    id: 'base-image-2',
    name: 'alpine:3.18',
    normalizedName: 'docker.io/library/alpine:3.18',
    scanningStatus: 'COMPLETED',
    lastScanned: '2025-10-13T09:15:00Z',
    createdAt: '2025-10-09T14:20:00Z',
    cveCount: {
      critical: 0,
      high: 3,
      medium: 5,
      low: 2,
      total: 10
    },
    imageCount: 8,
    deploymentCount: 5,
    lastBaseLayerIndex: 3
  },
  {
    id: 'base-image-3',
    name: 'node:18-alpine',
    normalizedName: 'docker.io/library/node:18-alpine',
    scanningStatus: 'IN_PROGRESS',
    lastScanned: null,
    createdAt: '2025-10-13T11:00:00Z',
    cveCount: {
      critical: 0,
      high: 0,
      medium: 0,
      low: 0,
      total: 0
    },
    imageCount: 0,
    deploymentCount: 0,
    lastBaseLayerIndex: 0
  },
  {
    id: 'base-image-4',
    name: 'nginx:1.25-alpine',
    normalizedName: 'docker.io/library/nginx:1.25-alpine',
    scanningStatus: 'COMPLETED',
    lastScanned: '2025-10-12T16:45:00Z',
    createdAt: '2025-10-08T10:30:00Z',
    cveCount: {
      critical: 2,
      high: 7,
      medium: 15,
      low: 6,
      total: 30
    },
    imageCount: 6,
    deploymentCount: 8,
    lastBaseLayerIndex: 4
  }
];
```

### Base Image CVEs (for detail view CVEs tab)

```typescript
const MOCK_BASE_IMAGE_CVES = {
  'base-image-1': [ // ubuntu:22.04
    {
      cveId: 'CVE-2024-1234',
      severity: 'CRITICAL',
      cvssScore: 9.8,
      summary: 'Buffer overflow in libssl allowing remote code execution',
      fixedBy: '1.2.3-4ubuntu1',
      components: [
        { name: 'libssl1.1', version: '1.2.3-3ubuntu1', layerIndex: 2 }
      ]
    },
    {
      cveId: 'CVE-2024-5678',
      severity: 'HIGH',
      cvssScore: 7.5,
      summary: 'SQL injection vulnerability in sqlite3',
      fixedBy: '3.37.2-2ubuntu1',
      components: [
        { name: 'sqlite3', version: '3.37.2-1ubuntu1', layerIndex: 1 }
      ]
    },
    // ... more CVEs
  ],
  'base-image-2': [ // alpine:3.18
    {
      cveId: 'CVE-2024-9999',
      severity: 'HIGH',
      cvssScore: 8.1,
      summary: 'Privilege escalation in busybox',
      fixedBy: '1.36.1-r2',
      components: [
        { name: 'busybox', version: '1.36.1-r1', layerIndex: 0 }
      ]
    }
  ]
  // ... more base images
};
```

### Application Images Using Base (for detail view Images tab)

```typescript
const MOCK_BASE_IMAGE_IMAGES = {
  'base-image-1': [ // ubuntu:22.04
    {
      imageId: 'sha256:app1',
      name: 'myapp:v1.2.3',
      sha: 'sha256:def456...',
      lastScanned: '2025-10-13T10:00:00Z',
      cveCount: {
        critical: 7,
        high: 15,
        medium: 28,
        low: 10,
        total: 60,
        baseImageCves: 48,
        applicationLayerCves: 12
      },
      deploymentCount: 3
    },
    {
      imageId: 'sha256:app2',
      name: 'web-frontend:2.1.0',
      sha: 'sha256:abc789...',
      lastScanned: '2025-10-13T09:30:00Z',
      cveCount: {
        critical: 5,
        high: 12,
        medium: 25,
        low: 8,
        total: 50,
        baseImageCves: 48,
        applicationLayerCves: 2
      },
      deploymentCount: 2
    }
    // ... more images
  ]
  // ... more base images
};
```

### Deployments Using Base (for detail view Deployments tab)

```typescript
const MOCK_BASE_IMAGE_DEPLOYMENTS = {
  'base-image-1': [ // ubuntu:22.04
    {
      deploymentId: 'deploy-1',
      name: 'web-frontend',
      namespace: 'production',
      cluster: 'prod-us-west-1',
      image: 'myapp:v1.2.3',
      cveCount: {
        critical: 7,
        high: 15,
        medium: 28,
        low: 10
      },
      riskPriority: 85
    },
    {
      deploymentId: 'deploy-2',
      name: 'api-server',
      namespace: 'production',
      cluster: 'prod-us-east-1',
      image: 'web-frontend:2.1.0',
      cveCount: {
        critical: 5,
        high: 12,
        medium: 25,
        low: 8
      },
      riskPriority: 72
    }
    // ... more deployments
  ]
  // ... more base images
};
```

### Enhanced Image Data (for Image Details page)

```typescript
// Add to existing image mock data
const MOCK_IMAGE_WITH_BASE = {
  id: 'sha256:app1',
  name: { fullName: 'myapp:v1.2.3' },
  // ... existing fields ...
  baseImage: {
    name: 'ubuntu:22.04',
    isManaged: true,  // User is tracking this base
    lastLayerIndex: 5,
    baseImageId: 'base-image-1'
  },
  scan: {
    components: [
      {
        name: 'libssl1.1',
        version: '1.2.3-3ubuntu1',
        layerIndex: 2,  // <= 5, so it's in base image
        vulns: [
          {
            cve: 'CVE-2024-1234',
            severity: 'CRITICAL',
            // isFromBaseImage computed as: layerIndex <= baseImage.lastLayerIndex
          }
        ]
      },
      {
        name: 'express',
        version: '4.18.2',
        layerIndex: 7,  // > 5, so it's in application layer
        vulns: [
          {
            cve: 'CVE-2024-5555',
            severity: 'HIGH'
          }
        ]
      }
      // ... more components
    ]
  }
};
```

---

## UI Components to Build

### 1. Base Images List Page

**Route:** `/vulnerabilities/base-images`

**Location:** New tab under Vulnerabilities section

**Components:**
- Page header with title "Base Images" and "Add base image" button
- Empty state (when no base images tracked):
  - Icon/illustration
  - Heading: "Track base images to understand CVE sources"
  - Description: "Monitor CVEs in your base images and see which application images are affected"
  - CTA: "Add your first base image"
- Table with columns:
  - **Base Image Name** (sortable, with link to detail page)
  - **Status** (badge: In Progress / Completed / Failed)
  - **Images Using** (count, sortable)
  - **Deployments** (count, sortable)
  - **CVEs** (severity badges, sortable by total)
  - **Last Scanned** (timestamp, sortable)
  - **Actions** (Remove button - just shows confirmation toast for prototype)

**Interactions:**
- Click row → Navigate to detail page
- Click "Add base image" → Show modal/form
- Click "Remove" → Show confirmation, then remove from list (client-side only)
- Sort by any column
- Basic search/filter by name

### 2. Add Base Image Modal

**Trigger:** "Add base image" button

**Content:**
- Input field for base image name
  - Placeholder: "e.g., ubuntu:22.04, alpine:3.18"
  - Validation: Required, must include tag
- "Cancel" and "Add" buttons

**Behavior (hardcoded):**
- On "Add":
  - Validate input (not empty, has colon for tag)
  - Add to list with status "IN_PROGRESS"
  - Show success toast: "Base image added and scanning initiated"
  - Close modal
  - After 2 seconds, update status to "COMPLETED" (simulate scan)

### 3. Base Image Detail Page

**Route:** `/vulnerabilities/base-images/:id`

**Components:**

**Header Section:**
- Base image name (large, prominent)
- Normalized name (smaller, gray text)
- Status badge
- Last scanned timestamp
- Summary metrics cards:
  - Total CVEs (with severity breakdown)
  - Images Using (count)
  - Deployments Affected (count)

**Tabbed Interface:**
- Three tabs: CVEs | Images | Deployments
- Tab content area with appropriate table for each

**Tab 1: CVEs**
- Table columns:
  - CVE ID (link to CVE details - can be placeholder)
  - Severity (badge)
  - CVSS Score
  - Summary (truncated with tooltip)
  - Fixed By (version)
  - Affected Components (expandable)
  - Layer Index
- Filters: Severity dropdown, Fixable checkbox
- Search: CVE ID or component name

**Tab 2: Images**
- Table columns:
  - Image Name (link to image details)
  - SHA (truncated)
  - Total CVEs (severity badges)
  - Base CVEs (count)
  - App CVEs (count)
  - Deployments (count)
  - Last Scanned
- Sort by CVE counts
- Search by image name

**Tab 3: Deployments**
- Table columns:
  - Deployment Name (link to deployment details - placeholder)
  - Namespace
  - Cluster
  - Image
  - CVEs (severity badges)
  - Risk Priority (numeric score)
- Filter by cluster, namespace
- Search by deployment name

### 4. Enhanced Image Details Page

**Route:** `/vulnerabilities/workload-cves/images/:id` (existing)

**New Components:**

**Base Image Section** (add near top, before tabs):
- Card showing:
  - "Base Image" label
  - Base image name (e.g., "ubuntu:22.04")
  - Status: Tracked or Untracked
  - If tracked: "View base image" link → navigates to base image detail page
  - If not tracked: "Track this base image" button → adds to tracked list

**Summary Cards** (modify existing):
- Split CVE summary into two cards:
  - **Base Image CVEs**
    - Title with base image name
    - CVE count by severity
    - Percentage of total
  - **Application Layer CVEs**
    - Title: "Application Layers"
    - CVE count by severity
    - Percentage of total

**CVE Table** (modify existing):
- Add new column: "Layer Type"
  - Badge: "Base Image" (blue) or "Application" (green)
  - Computed from: component.layerIndex <= baseImage.lastLayerIndex
- Add filter: "Layer Type" dropdown (Base Image / Application)

**Component Table** (expandable in CVE rows):
- Already shows layer info, just add visual distinction:
  - Highlight base image layers differently (background color or border)

### 5. Navigation Updates

**Vulnerabilities Section Tabs:**
- Add new tab: "Base Images"
- Order: Overview | Base Images | Watched Images | ... (existing tabs)

---

## Styling Guidelines

**Color Scheme:**
- Base Image badge: Blue (`--pf-v5-global--palette--blue-400`)
- Application Layer badge: Green (`--pf-v5-global--palette--green-400`)
- Status badges:
  - Completed: Green
  - In Progress: Blue
  - Failed: Red
- Severity badges: Use existing CVE severity colors

**Layout:**
- Follow existing PatternFly patterns
- Reuse table components from Image/CVE tables
- Match spacing and typography of existing pages
- Mobile responsive (basic support)

**Icons:**
- Base Images: Container icon or layers icon
- Status: CheckCircle (completed), InProgress (spinner), ExclamationCircle (failed)

---

## Interaction Patterns

**Navigation:**
- Breadcrumbs on detail pages: Vulnerabilities > Base Images > ubuntu:22.04
- Back button or clickable breadcrumbs
- Tab state preserved in URL (query param: `?tab=images`)

**Loading States:**
- Skeleton loaders for tables (use existing components)
- Spinner for status "IN_PROGRESS"
- Empty states with helpful messaging

**Toasts/Notifications:**
- Success: "Base image added and scanning initiated"
- Success: "Base image removed"
- Info: "Tracking base image..." (when clicking Track button)

**Confirmations:**
- Remove base image: "Are you sure you want to stop tracking ubuntu:22.04?"

---

## Technical Approach

**State Management:**
- React Context or local state (keep it simple)
- Store tracked base images in component state
- No need for Redux/complex state management for prototype

**Routing:**
- Use existing React Router setup
- Add routes:
  - `/vulnerabilities/base-images`
  - `/vulnerabilities/base-images/:id`

**Data Flow:**
- Create `mockData.ts` file with all hardcoded data structures
- Create `useBaseImages()` hook that returns mock data
- Simulate async operations with setTimeout (e.g., scanning status updates)

**Components to Reuse:**
- CVE severity badges (existing)
- Table components (PatternFly)
- Summary cards (existing pattern)
- Tabs (PatternFly tabs)
- Modal (PatternFly modal)

**New Components to Create:**
- `BaseImagesPage.tsx` - List view
- `BaseImageDetailPage.tsx` - Detail view
- `AddBaseImageModal.tsx` - Add modal
- `BaseImageTable.tsx` - Table component
- `BaseImageCVEsTab.tsx` - CVEs tab content
- `BaseImageImagesTab.tsx` - Images tab content
- `BaseImageDeploymentsTab.tsx` - Deployments tab content
- `BaseImageSummaryCards.tsx` - Summary metrics cards
- `mockData/baseImages.ts` - All mock data

---

## User Feedback Mechanisms

**What to Test with Users:**
1. Can users find the Base Images feature?
2. Is the value proposition clear from empty state?
3. Do users understand the base vs app layer distinction?
4. Is the tabbed navigation intuitive?
5. Do the summary cards provide useful insights?
6. Can users complete the tracking workflow easily?

**Prototype Limitations to Document:**
- Data is hardcoded and doesn't reflect real scans
- No actual scanning happens
- Removing/adding doesn't persist across page refreshes
- Some links are placeholders
- Limited to 3-4 base images for demonstration

---

## Success Criteria for Prototype

✅ **User can:**
1. Navigate to Base Images view from Vulnerabilities section
2. See a list of tracked base images with key metrics
3. Add a new base image via modal
4. View detailed information about a base image (3 tabs)
5. Understand CVE counts for base image
6. See which application images use a base
7. See which deployments are affected
8. Track a base image from Image Details page
9. See base vs application layer CVE distinction on Image Details

✅ **Stakeholders can:**
1. Validate the information architecture
2. Evaluate the tabbed navigation pattern
3. Assess the value of base/app layer distinction
4. Provide feedback on missing/unnecessary features
5. Confirm alignment with user workflows

---

## Next Steps After Prototype Validation

1. Gather feedback from stakeholders
2. Iterate on UX based on feedback
3. Document API requirements for backend team
4. Plan backend implementation phases
5. Replace mock data with real API calls
6. Add comprehensive error handling
7. Implement full accessibility compliance
8. Add advanced filtering and search
9. Implement pagination for large datasets
10. Add unit and integration tests

---

*Document created: 2025-10-14*
*Status: Requirements for UI prototype - Ready for implementation*
