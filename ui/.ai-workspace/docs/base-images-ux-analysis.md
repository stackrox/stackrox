# Base Images UX Analysis - Current vs Alternative Approach

## Overview

This document analyzes two UX approaches for the Base Images feature in ACS, evaluating their strengths, weaknesses, and alignment with product direction (modular views for OCP plugin).

---

## Current Approach: Modal-Based Management

### Description

**Location:** Vulnerabilities → Workload CVEs → Overview page

**Components:**
1. **"Manage base images" button** on Overview page (similar to "Manage watched images")
2. **Modal dialog** to add/remove base images
   - Shows table of managed base images
   - Add/remove functionality
3. **Image Details page enhancements:**
   - Base image assessment information
   - Layer type distinction in CVE breakdown
4. **Search filter:** "Base Image Layer Type"
   - Options: Application, Base Image
   - Filters vulnerability tables
5. **Component table column:** "Layer Type"
   - Shows in expandable row
   - Badge: "Base" or "Application"

### User Flow

```
Overview Page
    ↓
Click "Manage base images"
    ↓
Modal opens with table of base images
    ↓
User adds "ubuntu:22.04"
    ↓
Modal closes
    ↓
??? (nothing visible changes on Overview)
    ↓
User navigates to Image Details page
    ↓
Sees base image assessment information
    ↓
Can filter by layer type
```

---

## Issues with Current Approach

### 1. **UX Disconnect - No Immediate Feedback**

**Problem:**
- User performs action (add base image) on Overview page
- No visible change occurs on Overview page
- Must navigate elsewhere to see any effect

**Impact:**
- Confusing user experience
- "Did that work?" uncertainty
- Breaks expectation of cause-and-effect

**Example:**
```
User: *Adds ubuntu:22.04 via modal*
User: *Modal closes*
User: "Okay... now what? Did anything happen?"
```

### 2. **Discoverability Problem**

**Problem:**
- Feature is "hidden" behind a button
- No indication of what managing base images does
- No preview of value before using feature

**Impact:**
- Users won't understand the feature without trying it
- Low adoption rate
- Requires documentation/training

### 3. **Scattered Information Architecture**

**Problem:**
- Management happens on Overview page
- Consumption happens on Image Details page
- Filter happens on the CVEs table (each CVE row can be expanded to see a components table)
- No single place to understand base images holistically

**Impact:**
- Cognitive overhead to understand the feature
- Hard to discover relationships (which images use which bases)
- Information fragmentation

### 4. **Not Aligned with Modular Views Strategy**

**Problem:**
- Modal-based pattern doesn't translate well to OCP plugin
- Embedded in existing overview page (tightly coupled)
- No standalone view that can be reused

**Impact:**
- Will need to be redesigned for OCP plugin
- Doesn't leverage modular architecture
- Technical debt for future work

### 5. **Limited Base Image Context**

**Problem:**
- Modal only shows list of managed bases
- No insight into:
  - Which images use each base
  - CVE counts per base
  - Deployment impact
  - Scanning status

**Impact:**
- Management is "blind" - add bases without understanding impact
- Can't prioritize which bases to track
- No actionable insights

### 6. **Asymmetric Pattern with Watched Images**

**Problem:**
- Watched Images has similar modal pattern
- But watched images show up in Overview table
- Base images don't (in current approach)
- Creates inconsistent mental model

**Impact:**
- Users expect similar behavior
- Frustration when patterns don't match

---

## Alternative Approach: Dedicated Base Images View

### Description

**Location:** New top-level view: Vulnerabilities → **Base Images** (tab)

**Components:**

1. **Base Images List View** (`/vulnerabilities/base-images`)
   - Table showing all tracked base images
   - Columns:
     - Base Image Name (e.g., `ubuntu:22.04`)
     - Scanning Status (In Progress / Completed)
     - Deployments Using (count + link)
     - CVE Counts (Critical / High / Medium / Low badges)
     - Actions (Remove)
   - "Add base image" button above table
   - Add functionality:
     - Modal or inline form
     - Enter base image name
     - Immediate feedback on validation

2. **Base Image Detail View** (`/vulnerabilities/base-images/{id}`)
   - Accessed by clicking a base image from list
   - **Header section:**
     - Base image name and tags
     - Scan date/status
     - Summary metrics (CVE counts by severity)
   - **Tabbed interface** (aligns with vulnerability results overview pattern):
     - **CVEs tab** (default):
       - Table of all CVEs in the base image
       - Columns: CVE ID, Severity, CVSS Score, Component, Version, Layer, Fix Status
       - Expandable rows showing affected components
       - Filters: Severity, Fix Status, Component
     - **Images tab**:
       - Table of application images that use this base
       - Columns: Image Name, Tag, CVE Count (by severity), Last Scan, Deployments Count
       - Links to individual image detail pages
       - Filters: CVE severity, deployment status
     - **Deployments tab**:
       - Table of deployments running images with this base
       - Columns: Deployment Name, Namespace, Cluster, Image, CVE Count, Risk Priority
       - Links to deployment details
       - Filters: Cluster, namespace, risk level

3. **Enhanced Image Details Page**
   - Shows detected base image
   - Button: `[+ Track this base image]`
   - If already tracked: Badge "Tracked base image"
   - Links to Base Images view

4. **Search/Filter Integration**
   - On Overview page: Filter images by base
   - On Base Images view: Filter by CVE severity, deployment count

### User Flow

```
Vulnerabilities → Base Images tab
    ↓
See table of base images (empty initially)
    ↓
Click "Add base image"
    ↓
Enter "ubuntu:22.04"
    ↓
Table immediately shows new row:
    - ubuntu:22.04
    - Status: Scanning...
    - Deployments: - (pending)
    - CVEs: - (pending)
    ↓
After scan completes, row updates:
    - ubuntu:22.04
    - Status: ✓ Completed
    - Deployments: 12
    - CVEs: 5 Critical, 12 High, 23 Medium, 8 Low
    ↓
Click on "ubuntu:22.04"
    ↓
Base Image Detail view opens (CVEs tab by default)
    ↓
See CVE breakdown with component details
    ↓
Switch to Images tab → See 12 application images using this base
    ↓
Switch to Deployments tab → See deployments running those images
    ↓
Click on a deployment → Navigate to Image Details
```

**Alternative discovery path:**
```
User viewing Image Details for app:v1.2.3
    ↓
Sees: "Base Image: ubuntu:22.04"
    ↓
Click "[+ Track this base image]"
    ↓
Base image added to Base Images view
    ↓
Navigate to Base Images view to see tracking
```

---

## Comparative Analysis

| Aspect | Current Approach | Alternative Approach |
|--------|------------------|---------------------|
| **Immediate Feedback** | ❌ None - modal closes, nothing visible changes | ✅ Row appears in table immediately with status |
| **Discoverability** | ⚠️ Hidden behind button | ✅ Dedicated tab, clear purpose |
| **Information Architecture** | ❌ Scattered across Overview, Image Details, filters | ✅ Centralized in Base Images view |
| **Modular Views Alignment** | ❌ Tightly coupled to Overview page | ✅ Standalone view, easily portable to OCP plugin |
| **Context & Insights** | ⚠️ Limited - just list of bases | ✅ Rich context: deployments, CVEs, status |
| **Actionable Data** | ❌ Can't see impact of managed bases | ✅ Can prioritize based on CVE counts, deployment impact |
| **Scanning Feedback** | ❌ No visibility into scan status | ✅ Real-time status updates |
| **Navigation** | ⚠️ Requires navigating to Image Details to see value | ✅ Self-contained, drill-down from list to detail |
| **Consistency** | ⚠️ Asymmetric with Watched Images pattern | ✅ Follows standard list → detail pattern; tabbed interface matches vuln results overview |
| **Scalability** | ⚠️ Modal doesn't scale well with many bases | ✅ Table scales naturally, supports pagination, sorting |

---

## Advantages of Alternative Approach

### 1. **Immediate Value & Feedback**

**What users get:**
- Add base image → Immediately see it in table
- See scanning progress in real-time
- Metrics appear as scan completes
- No need to navigate elsewhere

**Why it matters:**
- Builds user confidence
- Reinforces mental model
- Encourages exploration and experimentation

### 2. **Base Images as a Workspace**

**What it enables:**
- Dedicated space to manage and monitor base images
- Answer questions like:
  - "Which bases have the most CVEs?"
  - "Which bases are most widely used?"
  - "Should I migrate from ubuntu:20.04 to ubuntu:22.04?"
- Strategic planning, not just tactical management

**Why it matters:**
- Aligns with security team workflows
- Supports organizational base image governance
- Provides decision-making data

### 3. **Better Information Scent**

**What users see:**
- Clear tab label: "Base Images"
- Purpose is obvious
- Hierarchy: List → Detail (familiar pattern)

**Why it matters:**
- Reduces onboarding friction
- Self-documenting UI
- Follows web conventions

### 4. **Contextual Cross-Links**

**What it enables:**
- Base Images view links to affected deployments
- Image Details links to Base Images view
- Bi-directional navigation
- Discoverability from multiple entry points

**Example:**
```
Security Engineer workflow:
"I want to see all images using ubuntu:22.04"
→ Go to Base Images view
→ Click on ubuntu:22.04
→ Default CVEs tab shows base image vulnerabilities
→ Switch to Images tab → See list of application images using this base
→ Switch to Deployments tab → See which deployments are affected
→ Click on deployment → Navigate to Image Details
→ See full vulnerability context
```

```
Developer workflow:
"My image has 50 CVEs, what should I do?"
→ View Image Details
→ See "Base Image: ubuntu:22.04 (40 CVEs)"
→ Click "Track this base image"
→ Navigate to Base Images view
→ Compare ubuntu:22.04 vs ubuntu:24.04
→ Make informed decision to update
```

### 5. **UI Pattern Consistency**

**What users get:**
- Tabbed interface matches vulnerability results overview page
- Same navigation pattern: CVEs / Images / Deployments
- Reuses familiar table components and filters
- No need to learn new interaction patterns

**Why it matters:**
- Reduces cognitive load
- Leverages existing mental models
- Faster user adoption
- Reinforces platform-wide design language

### 6. **Future-Proof Architecture**

**What it enables:**
- OCP plugin can embed Base Images view standalone
- View is self-contained, minimal dependencies
- Can add features without affecting other views
- Supports future enhancements:
  - Base image recommendations
  - CVE trending over time
  - Policy integration
  - Update notifications

**Why it matters:**
- Reduces technical debt
- Faster feature development
- Consistent UX across platforms

---

## Addressing Potential Concerns

### Concern: "This adds another tab to the navigation"

**Response:**
- Yes, but it's a meaningful, high-value view
- Reduces clutter in Overview page
- Users interested in base images have clear destination
- Users not interested can ignore the tab

**Mitigation:**
- Progressive disclosure: Show count badge when bases are tracked
- Empty state can educate users on feature value

### Concern: "Requires building a new page"

**Response:**
- True, but it's more sustainable long-term
- Alternative (modal-based) will need redesign for OCP plugin anyway
- Upfront investment pays off in better UX and modularity

**Mitigation:**
- Can build MVP with simple list view first
- Detail view can come in Phase 2

### Concern: "What if users don't discover the tab?"

**Response:**
- Multiple discovery paths:
  1. Tab navigation (primary)
  2. "Track this base image" button on Image Details (contextual)
  3. Empty state on Overview can prompt users
  4. Documentation and onboarding

**Mitigation:**
- Add contextual prompts on first use
- Link from Image Details when base is detected

---

## Recommended Implementation Plan

### Phase 1: Base Images List View (MVP)

**Goal:** Establish dedicated workspace for base images

**Components:**
1. New route: `/vulnerabilities/base-images`
2. Navigation tab: "Base Images" under Vulnerabilities
3. Base Images List View:
   - Table with columns: Name, Status, Deployments, CVEs, Actions
   - "Add base image" button
   - Simple modal/form to add base
   - Real-time status updates
4. Empty state:
   - Explains feature value
   - CTA to add first base image

**Files to create:**
- `ui/apps/platform/src/Containers/Vulnerabilities/BaseImages/`
  - `BaseImagesPage.tsx` (main view)
  - `BaseImagesTable.tsx` (table component)
  - `AddBaseImageModal.tsx` (add functionality)
  - `baseImagesService.ts` (API calls)

**API Requirements:**
- `GET /v1/base-images` - List tracked bases with metrics
- `POST /v1/base-images` - Add new base
- `DELETE /v1/base-images/{id}` - Remove base

### Phase 2: Base Image Detail View

**Goal:** Deep-dive into specific base image with tabbed navigation

**Components:**
1. New route: `/vulnerabilities/base-images/{id}`
2. Base Image Detail View:
   - Header with metadata and summary metrics
   - Tabbed interface with three tabs:
     - **CVEs tab**: Table of CVEs in base image with expandable component details
     - **Images tab**: Table of application images using this base
     - **Deployments tab**: Table of deployments running those images

**Files to create:**
- `BaseImageDetailPage.tsx` (main view with tab navigation)
- `BaseImageHeader.tsx` (header with summary)
- `tabs/`
  - `BaseImageCVEsTab.tsx` (CVEs table)
  - `BaseImageImagesTab.tsx` (images table)
  - `BaseImageDeploymentsTab.tsx` (deployments table)

**Implementation Notes:**
- Reuse existing tab component pattern from vulnerability results overview
- Each tab should have its own filters and sorting
- Use existing table components where possible (CVE table, image table, deployment table)

### Phase 3: Cross-Links & Integration

**Goal:** Connect base images to existing views

**Components:**
1. **Image Details page:**
   - Show detected base image
   - "Track this base image" button
   - Link to Base Images view
2. **Overview page:**
   - Optional: Add base image filter
   - Optional: Link to Base Images view in filters

**Files to modify:**
- `ImagePage.tsx` - Add base image section
- `searchFilterConfig.ts` - Add base image filter

---

## User Stories (Alternative Approach)

### Story 1: Security Team Manages Base Images
```
As a security engineer,
When I navigate to the Base Images view,
I want to see all tracked base images with CVE counts,
So that I can prioritize remediation efforts.
```

### Story 2: Developer Tracks Base Image
```
As a developer,
When I view my image details and see it uses ubuntu:22.04,
I want to click "Track this base image",
So that I can monitor CVEs specific to the base.
```

### Story 3: Platform Team Monitors Adoption
```
As a platform team lead,
When I view a base image's detail page,
I want to see which deployments use it,
So that I can plan migration rollouts.
```

### Story 4: Real-Time Feedback
```
As a user,
When I add a new base image,
I want to immediately see its scanning status,
So that I know the system is working.
```

---

## Conclusion

### Current Approach Issues Summary
1. ❌ No immediate feedback or visible change
2. ❌ Poor discoverability
3. ❌ Scattered information architecture
4. ❌ Not aligned with modular views strategy
5. ❌ Limited context and actionable insights

### Alternative Approach Benefits
1. ✅ Immediate, visible feedback
2. ✅ Dedicated workspace with rich context
3. ✅ Aligned with modular views for OCP plugin
4. ✅ Scalable, future-proof architecture
5. ✅ Better information architecture and navigation

### Recommendation

**Adopt the Alternative Approach: Dedicated Base Images View**

**Rationale:**
- Provides immediate value and feedback
- Aligns with product direction (modular views)
- Better UX for both discovery and power users
- Supports future enhancements without redesign
- Creates a strategic workspace, not just a management dialog

**Next Steps:**
1. Review with design and product teams
2. Validate with backend team on API feasibility
3. Create detailed wireframes/mockups
4. Implement Phase 1 MVP (List View)
5. Gather user feedback
6. Iterate and expand to Phases 2 & 3

---

*Document created: 2025-10-10*
*Status: Proposal - awaiting team review*
