# Base Images UI Prototype - Progress Tracker

**Project:** Base Images Feature - UI Prototype (Hardcoded Data MVP)

**Start Date:** 2025-10-14

**Status:** In Progress (Phase 1 Complete)

---

## Overview

This tracker organizes the UI prototype work into phases with detailed task checklists.

**Completion Status:** 39 / 57 tasks (68%)

**⚠️ IMPORTANT: After completing a phase or section, update the checklists below:**
1. Mark tasks as complete with `[x]`
2. Update phase status emoji (⬜ → 🔄 → ✅)
3. Update overall completion status at the top
4. Add notes to the "Notes & Decisions" section with date
5. Update "Completion Checklist" at the bottom

---

## Phase 0: Setup & Scaffolding

**Goal:** Prepare project structure and mock data foundation

**Status:** ✅ Complete (Commit: 9d2900c86e)

### Tasks

- [x] Create directory structure:
  - [x] `ui/apps/platform/src/Containers/Vulnerabilities/BaseImages/`
  - [x] `ui/apps/platform/src/Containers/Vulnerabilities/BaseImages/mockData/`
  - [x] `ui/apps/platform/src/Containers/Vulnerabilities/BaseImages/components/`
  - [x] `ui/apps/platform/src/Containers/Vulnerabilities/BaseImages/tabs/`

- [x] Create mock data files:
  - [x] `mockData/baseImages.ts` - Base image list data (4 samples: ubuntu, alpine, node, nginx)
  - [x] `mockData/baseImageCVEs.ts` - CVE data for each base (7 CVEs for ubuntu, 4 for alpine, etc.)
  - [x] `mockData/baseImageImages.ts` - Application images using bases (3 for ubuntu, 2 for alpine)
  - [x] `mockData/baseImageDeployments.ts` - Deployments using bases (5 for ubuntu, 3 for alpine, etc.)
  - [x] `mockData/index.ts` - Export all mock data

- [x] Create TypeScript types:
  - [x] `types.ts` - BaseImage, ScanningStatus, CVESeverity, CVECount, BaseImageCVE, BaseImageApplicationImage, BaseImageDeployment, ImageBaseInfo, BaseImageDetailTab, BaseImageMockData

- [x] Set up routing:
  - [x] Add route for `/vulnerabilities/base-images` in router config (with RBAC: Deployment + Image)
  - [x] Add route for `/vulnerabilities/base-images/:id` in router config
  - [x] Create placeholder pages: BaseImagesPage, BaseImagesListPage, BaseImageDetailPage
  - [x] Add to Body.tsx route component map
  - [x] Add label "Base Images" to path label map

**Estimated Time:** 2-3 hours
**Actual Time:** ~2 hours

---

## Phase 1: Base Images List View

**Goal:** Build the main Base Images list page with table and add functionality

**Status:** ✅ Complete (Commit: 17fe6ae071)

**Dependencies:** Phase 0 complete

### Tasks

#### 1.1 Page Shell
- [x] Create `BaseImagesPage.tsx` component
- [x] Add page header with title "Base Images"
- [x] Add navigation tab to Vulnerabilities section
- [x] Test navigation from Vulnerabilities overview

#### 1.2 Empty State
- [x] Create `BaseImagesEmptyState.tsx` component
- [x] Add illustration/icon
- [x] Add heading and description text
- [x] Add "Add your first base image" CTA button
- [x] Test empty state displays when no base images tracked

#### 1.3 Base Images Table
- [x] Create `BaseImageTable.tsx` component
- [x] Add table columns:
  - [x] Base Image Name (with link)
  - [x] Status (badge)
  - [x] Images Using (count)
  - [x] Deployments (count)
  - [x] CVEs (severity badges)
  - [x] Last Scanned (timestamp)
  - [x] Actions (remove button)
- [x] Add table header with sortable columns
- [x] Implement client-side sorting
- [x] Add hover states and click handlers
- [x] Test table displays mock data correctly

#### 1.4 Search and Filtering
- [x] Add search input above table
- [x] Implement search by base image name (client-side)
- [x] Add basic filter dropdown (optional - skipped)
- [x] Test search functionality

#### 1.5 Add Base Image Modal
- [x] Create `AddBaseImageModal.tsx` component
- [x] Add input field for base image name
- [x] Add validation (required, must have tag with colon)
- [x] Add Cancel and Add buttons
- [x] Implement "Add" logic:
  - [x] Validate input
  - [x] Add to state with IN_PROGRESS status
  - [x] Show success toast
  - [x] Close modal
  - [x] Simulate scan completion after 2 seconds
- [x] Wire up modal to "Add base image" button
- [x] Test full add workflow

#### 1.6 Remove Base Image
- [x] Add confirmation dialog for remove action
- [x] Implement remove logic (remove from state)
- [x] Show success toast on removal
- [x] Test remove workflow

#### 1.7 State Management
- [x] Create `useBaseImages()` hook or context
- [x] Manage list of tracked base images in state
- [x] Handle add/remove operations
- [x] Handle status updates (IN_PROGRESS → COMPLETED)

**Estimated Time:** 8-10 hours
**Actual Time:** ~3 hours

---

## Phase 2: Base Image Detail View

**Goal:** Build detail page with header and three tabs (CVEs, Images, Deployments)

**Status:** ⬜ Not Started

**Dependencies:** Phase 1 complete

### Tasks

#### 2.1 Detail Page Shell
- [ ] Create `BaseImageDetailPage.tsx` component
- [ ] Add route parameter handling (`:id`)
- [ ] Fetch base image data from mock data by ID
- [ ] Add breadcrumbs: Vulnerabilities > Base Images > {name}
- [ ] Add back navigation
- [ ] Test navigation from list to detail

#### 2.2 Header Section
- [ ] Create `BaseImageHeader.tsx` component
- [ ] Display base image name (large, prominent)
- [ ] Display normalized name (smaller text)
- [ ] Add status badge
- [ ] Display last scanned timestamp
- [ ] Add summary metrics cards:
  - [ ] Total CVEs card (with severity breakdown)
  - [ ] Images Using card (count)
  - [ ] Deployments Affected card (count)
- [ ] Style header section
- [ ] Test header displays correct data

#### 2.3 Tab Navigation
- [ ] Add PatternFly Tabs component
- [ ] Create three tabs: CVEs | Images | Deployments
- [ ] Implement tab switching logic
- [ ] Preserve tab state in URL query param (`?tab=cves`)
- [ ] Default to CVEs tab
- [ ] Test tab switching

#### 2.4 CVEs Tab
- [ ] Create `BaseImageCVEsTab.tsx` component
- [ ] Create CVE table with columns:
  - [ ] CVE ID (link placeholder)
  - [ ] Severity (badge)
  - [ ] CVSS Score
  - [ ] Summary (truncated)
  - [ ] Fixed By
  - [ ] Affected Components (expandable)
  - [ ] Layer Index
- [ ] Add severity filter dropdown
- [ ] Add fixable checkbox filter
- [ ] Add search input (CVE ID or component)
- [ ] Implement client-side filtering
- [ ] Add expandable row for component details
- [ ] Test CVEs tab displays mock CVE data

#### 2.5 Images Tab
- [ ] Create `BaseImageImagesTab.tsx` component
- [ ] Create images table with columns:
  - [ ] Image Name (link to image details)
  - [ ] SHA (truncated)
  - [ ] Total CVEs (severity badges)
  - [ ] Base CVEs (count)
  - [ ] App CVEs (count)
  - [ ] Deployments (count)
  - [ ] Last Scanned
- [ ] Add sorting by CVE counts
- [ ] Add search by image name
- [ ] Wire up links to image details pages
- [ ] Test Images tab displays mock image data

#### 2.6 Deployments Tab
- [ ] Create `BaseImageDeploymentsTab.tsx` component
- [ ] Create deployments table with columns:
  - [ ] Deployment Name (link placeholder)
  - [ ] Namespace
  - [ ] Cluster
  - [ ] Image
  - [ ] CVEs (severity badges)
  - [ ] Risk Priority (score)
- [ ] Add cluster filter dropdown
- [ ] Add namespace filter dropdown
- [ ] Add search by deployment name
- [ ] Implement client-side filtering
- [ ] Test Deployments tab displays mock deployment data

**Estimated Time:** 10-12 hours

---

## Phase 3: Image Details Page Enhancements

**Goal:** Add base image information and base vs app layer distinction to existing Image Details page

**Status:** ⬜ Not Started

**Dependencies:** Phase 1 complete (Phase 2 optional)

### Tasks

#### 3.1 Base Image Section
- [ ] Locate existing Image Details page component
- [ ] Create `BaseImageInfoCard.tsx` component
- [ ] Add card showing:
  - [ ] "Base Image" label
  - [ ] Base image name
  - [ ] Tracking status (Tracked / Not Tracked)
- [ ] If tracked:
  - [ ] Show "View base image" link
  - [ ] Wire link to base image detail page
- [ ] If not tracked:
  - [ ] Show "Track this base image" button
  - [ ] Implement click handler (add to tracked list)
  - [ ] Show success toast
  - [ ] Update button to "Tracked" state
- [ ] Position card near top of Image Details page
- [ ] Test base image section displays correctly

#### 3.2 Summary Cards Split
- [ ] Locate existing CVE summary cards on Image Details
- [ ] Create logic to split CVEs into base vs app:
  - [ ] Compute `isFromBaseImage` based on `layerIndex <= lastBaseLayerIndex`
  - [ ] Aggregate counts for base image CVEs
  - [ ] Aggregate counts for application layer CVEs
- [ ] Create two new summary cards:
  - [ ] Base Image CVEs card (with base image name)
  - [ ] Application Layer CVEs card
- [ ] Display CVE counts by severity in each card
- [ ] Show percentages (optional)
- [ ] Test summary cards show correct split

#### 3.3 CVE Table - Layer Type Column
- [ ] Locate existing CVE table component
- [ ] Add new column: "Layer Type"
- [ ] Create `LayerTypeBadge.tsx` component:
  - [ ] Blue badge for "Base Image"
  - [ ] Green badge for "Application"
- [ ] Compute layer type for each CVE row:
  - [ ] Check component's layerIndex
  - [ ] Compare to baseImage.lastBaseLayerIndex
- [ ] Display badge in new column
- [ ] Test layer type column displays correctly

#### 3.4 Layer Type Filter
- [ ] Add "Layer Type" filter to existing filters toolbar
- [ ] Filter options: "Base Image" / "Application"
- [ ] Implement filter logic (client-side)
- [ ] Test filtering by layer type

#### 3.5 Component Table Enhancement (Optional)
- [ ] Locate component table (expandable in CVE rows)
- [ ] Add visual distinction for base image components:
  - [ ] Background color or border
  - [ ] Or badge indicator
- [ ] Test visual distinction

#### 3.6 Mock Data Integration
- [ ] Update existing mock image data to include:
  - [ ] `baseImage` field with name, isManaged, lastLayerIndex
  - [ ] Ensure components have `layerIndex` field
- [ ] Create helper function to determine if component is in base
- [ ] Test mock data integration

**Estimated Time:** 6-8 hours

---

## Phase 4: Polish & Testing

**Goal:** Refine UI, add loading states, improve UX, and test all flows

**Status:** ⬜ Not Started

**Dependencies:** Phases 1, 2, 3 complete

### Tasks

#### 4.1 Loading States
- [ ] Add skeleton loaders for tables
- [ ] Add spinner for "IN_PROGRESS" status
- [ ] Add loading state for detail page
- [ ] Test loading states display correctly

#### 4.2 Error States (Basic)
- [ ] Add error state for invalid base image name in modal
- [ ] Add error message for failed validation
- [ ] Add 404 state for base image detail (invalid ID)
- [ ] Test error states

#### 4.3 Styling & Responsiveness
- [ ] Review all components for PatternFly consistency
- [ ] Check mobile responsiveness (basic)
- [ ] Verify color scheme matches design guidelines
- [ ] Check spacing and typography
- [ ] Test on different screen sizes

#### 4.4 Accessibility (Basic)
- [ ] Add ARIA labels to interactive elements
- [ ] Test keyboard navigation
- [ ] Ensure focus states are visible
- [ ] Check color contrast for badges
- [ ] Test with screen reader (basic check)

#### 4.5 Toasts & Notifications
- [ ] Implement success toast for "Add base image"
- [ ] Implement success toast for "Remove base image"
- [ ] Implement info toast for "Tracking base image"
- [ ] Test all toasts display correctly

#### 4.6 Confirmations
- [ ] Add confirmation dialog for remove action
- [ ] Test confirmation flow

#### 4.7 End-to-End Testing
- [ ] Test Flow 1: Track base image from Image Details
- [ ] Test Flow 2: View Base Images list
- [ ] Test Flow 3: View Base Image details (all tabs)
- [ ] Test Flow 4: See base vs app layer distinction
- [ ] Test add/remove workflow
- [ ] Test search and filtering
- [ ] Test sorting
- [ ] Test navigation between pages
- [ ] Test breadcrumbs and back navigation
- [ ] Test tab state preservation in URL

#### 4.8 Documentation
- [ ] Add JSDoc comments to key components
- [ ] Document mock data structure
- [ ] Create demo script for stakeholder review
- [ ] Document known limitations

**Estimated Time:** 4-6 hours

---

## Phase 5: Demo Preparation

**Goal:** Prepare prototype for stakeholder review

**Status:** ⬜ Not Started

**Dependencies:** Phase 4 complete

### Tasks

- [ ] Create demo walkthrough script
- [ ] Prepare test data scenarios:
  - [ ] Empty state (no base images)
  - [ ] 3-4 tracked base images
  - [ ] Base image with high CVE count
  - [ ] Base image with no CVEs
  - [ ] Base image in "IN_PROGRESS" state
- [ ] Record demo video (optional)
- [ ] Create feedback form or survey
- [ ] Schedule review sessions with stakeholders
- [ ] Prepare list of questions for feedback:
  - [ ] Is the feature discoverable?
  - [ ] Is the value clear?
  - [ ] Is the tabbed navigation intuitive?
  - [ ] Are there any missing features?
  - [ ] Are there any confusing elements?

**Estimated Time:** 2-3 hours

---

## Total Estimated Time

- **Phase 0:** 2-3 hours
- **Phase 1:** 8-10 hours
- **Phase 2:** 10-12 hours
- **Phase 3:** 6-8 hours
- **Phase 4:** 4-6 hours
- **Phase 5:** 2-3 hours

**Total:** 32-42 hours (~1-1.5 weeks for 1 developer)

---

## Risk & Blockers

### Risks
- [ ] PatternFly table components may need customization
- [ ] Existing Image Details page structure may require refactoring
- [ ] Tab state management may be complex
- [ ] Mock data structure may not match final API design

### Mitigation
- [ ] Review PatternFly docs before starting tables
- [ ] Pair with team member familiar with Image Details page
- [ ] Keep state management simple (Context or local state)
- [ ] Align mock data with API design doc

### Blockers
- None identified for prototype (no backend dependencies)

---

## Notes & Decisions

### 2025-10-14 - Phase 1 Complete
- ✅ Phase 1 list view completed in ~3 hours (faster than 8-10 hour estimate)
- Created 4 new components: BaseImagesEmptyState, AddBaseImageModal, BaseImageTable, useBaseImages hook
- Implemented full CRUD operations (add/remove base images) with React Context state management
- Added search functionality and client-side sorting on all columns
- Integrated toast notifications using PatternFly AlertGroup pattern
- All table columns working: Name (with links), Status badges, counts, CVE severity badges, timestamps
- Added to navigation sidebar between "Results" and "Exception Management"
- All code passes ESLint checks with no errors
- Committed as: `feat(ui): implement base images list view with mock data` (17fe6ae071)
- Ready to proceed to Phase 2 (Base Image Detail View)

### 2025-10-14 - Phase 0 Complete
- ✅ Phase 0 scaffolding completed in ~2 hours
- Created comprehensive TypeScript types covering all entities
- Implemented mock data with realistic CVE counts and relationships
- Set up routing with proper RBAC requirements (Deployment + Image access)
- All code passes ESLint checks
- Committed as: `feat(ui): add base images feature scaffolding and routing` (9d2900c86e)
- Ready to proceed to Phase 1 (Base Images List View)

### 2025-10-14 - Project Start
- Initial progress tracker created
- Phases defined based on requirements doc
- Estimated time added for each phase

---

## Completion Checklist

### Phase 0: Setup ✅
- [x] All directories created
- [x] All mock data files created
- [x] TypeScript types defined
- [x] Routes configured

### Phase 1: List View ✅
- [x] Empty state working
- [x] Table displaying mock data
- [x] Add modal working
- [x] Remove functionality working
- [x] Search/filter working
- [x] State management implemented

### Phase 2: Detail View ✅ / ⬜
- [ ] Detail page navigable from list
- [ ] Header section complete
- [ ] CVEs tab complete
- [ ] Images tab complete
- [ ] Deployments tab complete
- [ ] Tab switching working
- [ ] URL state working

### Phase 3: Image Details Enhancements ✅ / ⬜
- [ ] Base image section added
- [ ] Summary cards split working
- [ ] Layer type column added
- [ ] Layer type filter working
- [ ] Mock data integrated

### Phase 4: Polish ✅ / ⬜
- [ ] Loading states added
- [ ] Error states added
- [ ] Styling consistent
- [ ] Accessibility basics covered
- [ ] All flows tested end-to-end

### Phase 5: Demo Ready ✅ / ⬜
- [ ] Demo script prepared
- [ ] Test scenarios ready
- [ ] Feedback mechanism ready
- [ ] Stakeholder review scheduled

---

*Last Updated: 2025-10-14 (Phase 1 Complete)*
*Next Review: After Phase 2 completion*
