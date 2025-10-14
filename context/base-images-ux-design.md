# Base Images UX Design - Learning Document

## Problem Statement

In ACS, we want to clearly identify for a container image which CVEs are detected in container base image layers versus which CVEs are detected in layers added on top of a base image (application layers). This helps users:

- **Understand remediation paths**: Base image CVEs require updating the FROM statement, while app layer CVEs require updating dependencies
- **Prioritize effectively**: Distinguish between vulnerabilities they control vs those from upstream base images
- **Make informed decisions**: Choose better base images or understand the security posture of their base choices

---

## Current State of the Codebase

### Existing Infrastructure

✅ **Layer information already exists**
- Location: `ui/apps/platform/src/Containers/Vulnerabilities/WorkloadCves/Tables/table.utils.ts`
- `ImageMetadataContext` includes `metadata.v1.layers[]` with instruction and value
- Each component has `layerIndex` mapping it to a specific layer
- `DockerfileLayer` component already renders layer information

✅ **Component-level vulnerability tracking**
- Each `ImageComponent` includes:
  - `name`, `version`, `location`, `source`
  - `layerIndex` - which layer contains this component
  - `imageVulnerabilities[]` - CVEs affecting this component

✅ **Watched Images pattern**
- Location: `ui/apps/platform/src/Containers/Vulnerabilities/WorkloadCves/WatchedImages/`
- Provides a modal-based management UI for tracking specific images
- Integrated into overview page with "Manage watched images" button
- Good pattern to follow for "Manage base images"

### Key Files Reference

**Image Details Page:**
- `ui/apps/platform/src/Containers/Vulnerabilities/WorkloadCves/Image/ImagePage.tsx`
- Shows image name, SHA, badges, tabs for Vulnerabilities/Resources/Signature verification

**Vulnerabilities Tab:**
- `ui/apps/platform/src/Containers/Vulnerabilities/WorkloadCves/Image/ImagePageVulnerabilities.tsx`
- Lines 287-309: Summary cards section
- Lines 269-285: Advanced filters toolbar

**CVE Table:**
- `ui/apps/platform/src/Containers/Vulnerabilities/WorkloadCves/Tables/ImageVulnerabilitiesTable.tsx`
- Lines 66-131: Column definitions
- Lines 209-263: Table header
- Lines 334-458: Table rows

**Component Table:**
- `ui/apps/platform/src/Containers/Vulnerabilities/WorkloadCves/Tables/ImageComponentVulnerabilitiesTable.tsx`
- Already displays DockerfileLayer component (line 111-115)

**Overview Page:**
- `ui/apps/platform/src/Containers/Vulnerabilities/WorkloadCves/Overview/WorkloadCvesOverviewPage.tsx`
- Line 345-356: Where "Manage watched images" button exists

---

## UI Changes Needed for Base vs Application Layer Distinction

### 1. Image Details Page - Summary Cards

**File:** `ImagePage.tsx` (lines 189-266)

Add two summary cards showing base vs application layer CVE breakdown:

```tsx
<SummaryCardLayout>
  <SummaryCard>
    <Title>Base Image CVEs</Title>
    <Text>32 CVEs from ubuntu:22.04</Text>
    <Text>🔴 5 Critical  🟠 12 High  🟡 15 Medium</Text>
  </SummaryCard>
  <SummaryCard>
    <Title>Application Layer CVEs</Title>
    <Text>13 CVEs from application layers</Text>
    <Text>🔴 2 Critical  🟠 8 High  🟡 3 Medium</Text>
  </SummaryCard>
</SummaryCardLayout>
```

### 2. Vulnerabilities Table - Layer Type Column

**File:** `ImageVulnerabilitiesTable.tsx`

Add new column to show which layer type each CVE belongs to:

```tsx
// In defaultColumns (lines 66-131)
layerType: {
  title: 'Layer Type',
  isShownByDefault: true,
}

// In table header (lines 209-263)
<Th>Layer Type</Th>

// In table body (lines 334-458)
<Td dataLabel="Layer Type">
  <Badge color={isBaseLayer ? 'blue' : 'green'}>
    {isBaseLayer ? 'Base Image' : 'Application'}
  </Badge>
</Td>
```

### 3. Search Filters - Layer Type Filter

**File:** `searchFilterConfig.ts`

Add filter option:

```typescript
{
  displayName: 'Layer Type',
  searchTerm: 'Layer Type',
  inputType: 'select',
  options: ['Base Image Layer', 'Application Layer']
}
```

### 4. Dockerfile Layer Component - Visual Distinction

**File:** `DockerfileLayer.tsx`

Add visual indicator (border color, background) for base vs app layers:

```tsx
<CodeBlock className={isBaseLayer ? 'base-layer' : 'app-layer'}>
  {isBaseLayer && <Badge>Base Image</Badge>}
  <CodeBlockCode>
    {layer.line} {layer.instruction} {layer.value}
  </CodeBlockCode>
</CodeBlock>
```

### 5. Determine Base Image Boundary

**File:** `table.utils.ts`

Add utility function:

```typescript
/**
 * Determines which layers are part of the base image vs application layers.
 * Typically, base image = first FROM instruction through to the last base layer.
 * Application layers = everything after (RUN, COPY, ADD commands for app code).
 */
export function getBaseImageBoundary(
  layers: { instruction: string; value: string }[]
): { lastBaseLayerIndex: number; baseImageName: string } | null {
  // Find first FROM instruction
  const fromLayerIndex = layers.findIndex(l => l.instruction === 'FROM');
  if (fromLayerIndex === -1) return null;

  const baseImageName = layers[fromLayerIndex].value;

  // Find where application layers start (first COPY, ADD, or app-specific RUN)
  // This logic needs to be refined based on backend support
  const appLayerStart = layers.findIndex((l, i) =>
    i > fromLayerIndex &&
    (l.instruction === 'COPY' || l.instruction === 'ADD')
  );

  const lastBaseLayerIndex = appLayerStart > 0 ? appLayerStart - 1 : fromLayerIndex;

  return { lastBaseLayerIndex, baseImageName };
}
```

---

## Managing Base Images - UX Flow Options

### Context: The MVP Challenge

**Proposed MVP:**
- Add "Manage Base Images" button on Overview page
- Modal to add/remove base images
- Show base image info **only** on Image Details page

**Problem:** This creates a UX disconnect:
1. User adds base image via modal on Overview page
2. Nothing visible changes on Overview page
3. User must navigate to Image Details to see any effect
4. No discoverability, no immediate value

---

## Alternative UX Approaches

### Option A: Minimal Visibility on Overview ⭐ **RECOMMENDED FOR MVP**

**Changes:**

1. **Add "Base Image" column to Images table**
   - File: `ImageOverviewTable.tsx`
   - Shows detected base image name (e.g., `ubuntu:22.04`)
   - Badge if it's a "managed" base image

2. **Add base image filter to search**
   - File: `searchFilterConfig.ts`
   - Users can filter: "Show me all images using `ubuntu:22.04`"

3. **Optional: Small indicator in CVE count**
   - In CVE severity column: `45 (32 base)`
   - Provides immediate insight without cluttering UI

**User Flow:**
1. User clicks "Manage base images"
2. Adds `ubuntu:22.04` to tracked bases
3. Modal closes
4. **Images table now shows `ubuntu:22.04` in "Base Image" column**
5. User can filter by base image to see all images using it
6. Click into image details for full breakdown

**Pros:**
- ✅ Immediate feedback when you manage a base image
- ✅ Can filter/search for images by base
- ✅ Minimal UI changes, still MVP-appropriate
- ✅ Natural discovery: "I can filter by base image? Let me manage which ones I care about"

**Cons:**
- ❌ Adds one more column to already information-dense table
- ❌ Might push other columns off screen on smaller displays

**Implementation:**

```tsx
// WorkloadCvesOverviewPage.tsx - Add button
<Button variant="secondary" onClick={openManageBaseImagesModal}>
    Manage base images
</Button>

// ImageOverviewTable.tsx - Add column
<Th>Base Image</Th>
<Td dataLabel="Base Image">
    {image.baseImageName || 'Not detected'}
    {image.isBaseManaged && <Badge>Tracked</Badge>}
</Td>

// searchFilterConfig.ts - Add filter
{
    displayName: 'Base Image',
    searchTerm: 'Base Image Name',
    inputType: 'autocomplete',
}
```

---

### Option B: Move Management to Image Details

**Changes:**

1. **Remove "Manage Base Images" from overview**
2. **Add management contextually in Image Details page**
   - User sees: "Base Image: `ubuntu:22.04`"
   - Button next to it: `[★ Track this base image]`

3. **Create dedicated Base Images view**
   - New route: `/vulnerabilities/base-images`
   - Shows list of bases user is tracking
   - Each row links to images using that base
   - Becomes a curated "workspace" for base images

**User Flow:**
1. User views an image details page
2. Sees base image with "Track this base image" button
3. Clicks to add to tracked bases
4. Can navigate to `/vulnerabilities/base-images` to see all tracked bases
5. From there, can view all images using each base

**Pros:**
- ✅ Contextual management: track a base when you encounter it
- ✅ Creates a dedicated "workspace" for base image tracking
- ✅ Cleaner overview page (no extra button)
- ✅ More scalable for future features

**Cons:**
- ❌ Requires creating a new page (more than "minimal" MVP)
- ❌ Less discoverable - users might not find the feature
- ❌ More complex navigation

---

### Option C: Hybrid - Summary Cards Appear After Registration

**Changes:**

1. **Overview page shows nothing by default**
2. **After user adds base images**, new section appears:
   ```
   ┌─────────────────────────────────────────────┐
   │ Your Tracked Base Images                    │
   ├─────────────────────────────────────────────┤
   │ ubuntu:22.04       45 CVEs   12 images      │
   │ alpine:3.18        8 CVEs    5 images       │
   │ node:18-alpine     23 CVEs   8 images       │
   └─────────────────────────────────────────────┘
   ```
3. **Clicking a base image** filters images table to show only those using that base

**Pros:**
- ✅ Progressive disclosure: feature appears when relevant
- ✅ Provides immediate value after managing bases
- ✅ Acts as quick navigation to filtered views

**Cons:**
- ❌ Dynamic UI (section appears/disappears) can be confusing
- ❌ Takes up valuable real estate on overview page
- ❌ More complex to implement

---

## Backend Flow Explained (For Frontend Engineers)

### What Happens When User Adds a Base Image?

**Frontend Request:**
```typescript
// User adds "ubuntu:22.04"
await addBaseImage('ubuntu:22.04');

// API call
POST /v1/base-images
{
  "name": "ubuntu:22.04"
}
```

**Backend Processing:**
1. Saves `ubuntu:22.04` to `managed_base_images` table
2. **Might** trigger background scan for this image in registry
3. **Might** analyze existing images to detect if they use this base
4. **Does NOT** immediately re-scan all images or recalculate CVE assignments

**Backend Response:**
```json
200 OK
{
  "id": "base-image-123",
  "name": "ubuntu:22.04",
  "normalizedName": "docker.io/library/ubuntu:22.04",
  "status": "tracked"
}
```

**Frontend Update:**
```typescript
await refetchManagedBaseImages(); // Refresh the list
showSuccessToast("Base image added");
```

---

### How Frontend Determines Base vs App Layers

**Option 1: Backend Provides the Information** (Preferred)

Backend adds computed fields to image query response:

```graphql
query getImageDetails($id: ID!) {
  image(id: $id) {
    baseImage {
      name: "ubuntu:22.04"
      isManaged: true           # User added this via modal
      lastLayerIndex: 5         # Layers 0-5 are base, 6+ are app
    }
    imageVulnerabilities {
      cve
      isFromBaseImage: true     # Backend calculated this
      imageComponents {
        name
        layerIndex
        isInBaseImage: true     # Based on lastLayerIndex
      }
    }
  }
}
```

**Option 2: Frontend Calculates On-the-Fly**

Backend returns managed bases separately, frontend does the logic:

```typescript
// Fetch managed bases
const managedBases = await getManagedBaseImages();
// Returns: [{ name: "ubuntu:22.04", lastLayerIndex: 5 }]

// In component, determine if CVE is from base
const isBaseLayerCVE = (component) => {
  const baseImage = managedBases.find(base =>
    image.layers[0].value.includes(base.name)
  );
  return baseImage && component.layerIndex <= baseImage.lastLayerIndex;
};

// Split CVEs
const baseImageCVEs = vulnerabilities.filter(vuln =>
  vuln.imageComponents.some(comp => isBaseLayerCVE(comp))
);

const appLayerCVEs = vulnerabilities.filter(vuln =>
  vuln.imageComponents.every(comp => !isBaseLayerCVE(comp))
);
```

---

### Key Frontend Insights

1. **Adding a base image = registering metadata**, not triggering re-scans
2. **Layer data already exists** in image metadata (already being fetched)
3. **The "magic" is determining**: which layers = base vs app
4. **Backend might provide this logic**, or frontend calculates client-side
5. **Managed bases list is separate** from individual image queries

**Data Flow:**
```
User adds "ubuntu:22.04"
    ↓
Backend saves to managed_base_images table
    ↓
Next time you fetch images, include managed bases in query
    ↓
Frontend matches image layers to managed bases
    ↓
Calculate which CVEs are from base vs app layers
    ↓
Show breakdown in UI
```

**Key Insight:** You're not changing the images themselves, you're adding metadata that helps you _interpret_ existing layer data differently.

---

## Recommended MVP Implementation

### Phase 1: Core Distinction (No Management Yet)

**Goal:** Show base vs app layer CVE breakdown on Image Details page

**Requirements:**
- Backend provides base image detection (from first FROM instruction)
- Backend calculates `isFromBaseImage` flag for CVEs
- Frontend displays the breakdown

**Files to modify:**
- `ImagePage.tsx` - Add summary cards for base/app CVEs
- `ImagePageVulnerabilities.tsx` - Split summary by layer type
- `ImageVulnerabilitiesTable.tsx` - Add "Layer Type" column
- `searchFilterConfig.ts` - Add layer type filter

**No "Manage Base Images" feature yet** - just show the distinction

---

### Phase 2: Add Management + Minimal Visibility (Option A)

**Goal:** Let users track specific base images and filter by them

**Requirements:**
- API endpoints: `GET/POST/DELETE /v1/base-images`
- Frontend modal for managing base images
- Add "Base Image" column to overview table
- Add base image filter

**Files to create:**
- `BaseImages/BaseImagesModal.tsx`
- `BaseImages/BaseImagesForm.tsx`
- `BaseImages/BaseImagesTable.tsx`

**Files to modify:**
- `WorkloadCvesOverviewPage.tsx` - Add "Manage base images" button
- `ImageOverviewTable.tsx` - Add base image column
- `searchFilterConfig.ts` - Add base image name filter
- `services/imageService.ts` - Add API calls

---

### Phase 3: Enhanced Features (Future)

**Potential additions:**
- Dedicated Base Images page (`/vulnerabilities/base-images`)
- Base image update recommendations
- CVE comparison between base image versions
- Policy integration for base image governance
- Automated Dockerfile generation with recommended bases

---

## Decision Points for Implementation

### 1. How to Determine Base Image Boundary?

**Options:**
- **A. First FROM instruction only** - Simple but might miss multi-stage builds
- **B. All FROM instructions** - Better for multi-stage, but complex
- **C. Backend configuration** - Let users override detection logic
- **D. Heuristic-based** - First COPY/ADD after FROM marks app layer start

**Recommendation:** Start with A, evolve to D with backend support

---

### 2. Multi-stage Build Support?

**Question:** How to handle Dockerfiles with multiple FROM instructions?

```dockerfile
FROM node:18 AS builder
COPY . .
RUN npm build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
```

**Options:**
- **A. Only track final FROM** - nginx:alpine is the base
- **B. Track all FROM instructions** - Both node:18 and nginx:alpine
- **C. User specifies** - Let user choose which is the "base"

**Recommendation:** Start with A for MVP, add B in Phase 2

---

### 3. What if Component Exists in Both Base and App Layers?

**Scenario:** User installs `curl` in base layer, then updates it in app layer

**Options:**
- **A. Mark as "Both"** - Special indicator
- **B. Prefer app layer** - If in app layer, that's what matters
- **C. Show separate entries** - One row for base, one for app

**Recommendation:** B for simplicity, C for complete accuracy

---

## Open Questions for Backend Team

1. **Does backend already provide base image detection?**
   - Or should frontend infer from layer metadata?

2. **Can backend calculate `isFromBaseImage` flag for CVEs?**
   - Or should frontend calculate based on `layerIndex`?

3. **What's the performance impact of base image tracking?**
   - Does adding a managed base trigger re-scans?

4. **How to handle base image updates?**
   - If user's base is `ubuntu:22.04`, do we track version changes?

5. **API design for managed bases:**
   - CRUD operations needed: `GET/POST/DELETE /v1/base-images`
   - Query images by base: `GET /v1/images?baseImage=ubuntu:22.04`

6. **Multi-stage build support:**
   - Does backend track all FROM instructions?
   - How to determine which is the "primary" base?

---

## User Stories

### Story 1: Developer Investigates High CVE Count
```
As a developer,
When I view my application image with 50 CVEs,
I want to see that 40 CVEs are from the ubuntu:20.04 base image,
So that I know updating the base image will fix most issues.
```

### Story 2: Security Team Standardizes Base Images
```
As a security engineer,
When I manage base images,
I want to track all images using ubuntu:20.04,
So that I can plan migration to ubuntu:22.04 across the organization.
```

### Story 3: Platform Team Provides Approved Bases
```
As a platform team lead,
When I provide approved base images,
I want developers to see CVE counts for each approved base,
So that they can make informed decisions when choosing a base.
```

---

## Success Metrics

**MVP Success Criteria:**
- Users can distinguish base vs app layer CVEs on image details page
- Users can track specific base images they care about
- Users can filter images by base image name

**Future Success Metrics:**
- % of users who adopt recommended base image updates
- Reduction in CVE count after base image updates
- Time saved in CVE remediation (base update vs app fixes)

---

## References

**Key Codebase Files:**
- Vulnerabilities overview: `WorkloadCves/Overview/WorkloadCvesOverviewPage.tsx`
- Image details: `WorkloadCves/Image/ImagePage.tsx`
- CVE table: `WorkloadCves/Tables/ImageVulnerabilitiesTable.tsx`
- Layer utilities: `WorkloadCves/Tables/table.utils.ts`
- Watched images pattern: `WorkloadCves/WatchedImages/WatchedImagesModal.tsx`

**Design Patterns to Follow:**
- Modal management: WatchedImagesModal
- Summary cards: CvesByStatusSummaryCard
- Table columns: ImageOverviewTable
- Search filters: searchFilterConfig.ts

---

## Next Steps

1. **Backend team:** Confirm base image detection approach
2. **Design team:** Review Option A vs Option B for management UX
3. **Frontend team:** Prototype Phase 1 (core distinction) without management
4. **Product team:** Validate user stories with customers
5. **All teams:** Align on multi-stage build handling

---

*Document created: 2025-01-09*
*Last updated: 2025-01-09*
*Status: Design exploration - awaiting decisions on UX approach and backend support*
