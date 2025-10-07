# StackRox Search Architecture
## Frontend-Focused Guide for AI-Powered Search Implementation

**Last Updated:** 2025-01-07
**Purpose:** Reference guide for implementing AI-powered natural language search in the StackRox UI

---

## Table of Contents

1. [Overview](#overview)
2. [Frontend Search System](#frontend-search-system)
3. [Search State Management](#search-state-management)
4. [Backend Query Format](#backend-query-format)
5. [Available Search Fields](#available-search-fields)
6. [Integration Points for AI Search](#integration-points-for-ai-search)
7. [Practical Examples](#practical-examples)
8. [Key Files Reference](#key-files-reference)

---

## Overview

StackRox search is built on a **string-based query system** where filters are encoded in URL parameters and sent to the backend as structured query strings.

### High-Level Flow

```
User Action → SearchFilter Object → URL Query String → Backend Parser → Database Query → Results
```

**Example:**
- User selects: `Severity = Critical` and `Cluster = production`
- Creates SearchFilter: `{ SEVERITY: "CRITICAL", "Cluster": "production" }`
- Becomes URL: `?s[SEVERITY]=CRITICAL&s[Cluster]=production`
- Backend receives: `"SEVERITY:CRITICAL+Cluster:production"`
- AI Goal: Skip steps 1-2, generate SearchFilter directly from natural language

---

## Frontend Search System

### 1. CompoundSearchFilter Structure

The UI uses a **declarative configuration system** to define available filters.

#### Type Definitions

**Location:** `apps/platform/src/Components/CompoundSearchFilter/types.ts`

```typescript
// Core types
type CompoundSearchFilterConfig = CompoundSearchFilterEntity[];

type CompoundSearchFilterEntity = {
    displayName: string;              // "CVE", "Image", "Cluster"
    searchCategory: SearchCategory;   // Backend category enum
    attributes: CompoundSearchFilterAttribute[];
};

type CompoundSearchFilterAttribute = {
    displayName: string;      // "Severity", "Fixable", "CVE Created Time"
    filterChipLabel: string;  // Display name for filter chips
    searchTerm: string;       // Backend field name (e.g., "SEVERITY", "FIXABLE")
    inputType: InputType;     // 'select' | 'text' | 'autocomplete' | 'date-picker' | etc.
    inputProps?: {            // Type-specific props
        options?: { label: string; value: string }[];  // For 'select' type
    };
};
```

#### Input Types

| Input Type | Use Case | Example |
|------------|----------|---------|
| `select` | Predefined options | Severity (Critical, High, Medium, Low) |
| `text` | Free text input | CVE name search |
| `autocomplete` | Backend-provided suggestions | Cluster names, Namespace names |
| `date-picker` | Date selection | CVE Created Time |
| `condition-number` | Numeric comparisons | CVSS > 7, Image CVE Count >= 5 |
| `condition-text` | Text with conditions | Component Version |

### 2. SearchFilterConfig Examples

**Location:** `apps/platform/src/Containers/Vulnerabilities/searchFilterConfig.ts`

#### CVE Search Filter
```typescript
export const imageCVESearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'CVE',
    searchCategory: 'IMAGE_VULNERABILITIES_V2',
    attributes: [
        {
            displayName: 'CVE',
            filterChipLabel: 'CVE',
            searchTerm: 'CVE',
            inputType: 'autocomplete',
        },
        {
            displayName: 'Severity',
            filterChipLabel: 'CVE severity',
            searchTerm: 'SEVERITY',
            inputType: 'select',
            inputProps: {
                options: [
                    { label: 'Critical', value: 'CRITICAL_VULNERABILITY_SEVERITY' },
                    { label: 'Important', value: 'IMPORTANT_VULNERABILITY_SEVERITY' },
                    { label: 'Moderate', value: 'MODERATE_VULNERABILITY_SEVERITY' },
                    { label: 'Low', value: 'LOW_VULNERABILITY_SEVERITY' },
                ],
            },
        },
        {
            displayName: 'Fixable',
            filterChipLabel: 'CVE fixable',
            searchTerm: 'FIXABLE',
            inputType: 'select',
            inputProps: {
                options: [
                    { label: 'true', value: 'true' },
                    { label: 'false', value: 'false' },
                ],
            },
        },
        {
            displayName: 'CVE Created Time',
            filterChipLabel: 'CVE created time',
            searchTerm: 'CVE Created Time',
            inputType: 'date-picker',
        },
        // ... more attributes
    ],
};
```

#### Image Search Filter
```typescript
export const imageSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Image',
    searchCategory: 'IMAGES',
    attributes: [
        {
            displayName: 'Image',
            filterChipLabel: 'Image',
            searchTerm: 'Image',
            inputType: 'autocomplete',
        },
        {
            displayName: 'Image registry',
            filterChipLabel: 'Image registry',
            searchTerm: 'Image Registry',
            inputType: 'autocomplete',
        },
        {
            displayName: 'Image tag',
            filterChipLabel: 'Image tag',
            searchTerm: 'Image Tag',
            inputType: 'text',
        },
        // ... more attributes
    ],
};
```

#### Cluster Search Filter
```typescript
export const clusterSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Cluster',
    searchCategory: 'CLUSTERS',
    attributes: [
        {
            displayName: 'Cluster',
            filterChipLabel: 'Cluster',
            searchTerm: 'Cluster',
            inputType: 'autocomplete',
        },
        {
            displayName: 'Cluster type',
            filterChipLabel: 'Cluster type',
            searchTerm: 'Cluster Type',
            inputType: 'select',
            inputProps: {
                options: [
                    { label: 'Kubernetes', value: 'KUBERNETES_CLUSTER' },
                    { label: 'OpenShift 3', value: 'OPENSHIFT_CLUSTER' },
                    { label: 'OpenShift 4', value: 'OPENSHIFT4_CLUSTER' },
                ],
            },
        },
        // ... more attributes
    ],
};
```

### 3. Complete Config for Vulnerability Page

**Location:** `apps/platform/src/Containers/Vulnerabilities/WorkloadCves/Overview/WorkloadCvesOverviewPage.tsx`

```typescript
const searchFilterConfig: CompoundSearchFilterConfig = [
    imageCVESearchFilterConfig,      // CVE filters
    imageSearchFilterConfig,         // Image filters
    imageComponentSearchFilterConfig, // Component filters
    deploymentSearchFilterConfig,    // Deployment filters
    namespaceSearchFilterConfig,     // Namespace filters
    clusterSearchFilterConfig,       // Cluster filters
];
```

**This is the config AI will need to understand to generate valid filters!**

---

## Search State Management

### 1. SearchFilter Type

**Location:** `apps/platform/src/types/search.ts`

```typescript
// SearchFilter is a Record (object) mapping field names to values
export type SearchFilter = Record<string, string | string[]>;

// Examples:
const exampleFilter1: SearchFilter = {
    SEVERITY: "CRITICAL_VULNERABILITY_SEVERITY",
    Cluster: "production"
};

const exampleFilter2: SearchFilter = {
    SEVERITY: ["CRITICAL_VULNERABILITY_SEVERITY", "IMPORTANT_VULNERABILITY_SEVERITY"],
    FIXABLE: "true",
    "CVE Created Time": ">=2024-09-26"
};
```

### 2. useURLSearch Hook

**Location:** `apps/platform/src/hooks/useURLSearch.ts`

The primary hook for managing search state via URL parameters.

```typescript
function useURLSearch(): {
    searchFilter: SearchFilter;
    setSearchFilter: (filter: SearchFilter) => void;
} {
    // Reads from URL: ?s[SEVERITY]=CRITICAL&s[Cluster]=prod
    // Returns: { SEVERITY: "CRITICAL", Cluster: "prod" }

    // Updates URL when filter changes
    // Triggers re-render and data fetching
}
```

**Usage Example:**
```typescript
function MyComponent() {
    const { searchFilter, setSearchFilter } = useURLSearch();

    // Add a filter
    const addFilter = (field: string, value: string) => {
        setSearchFilter({
            ...searchFilter,
            [field]: value
        });
    };

    // Remove a filter
    const removeFilter = (field: string) => {
        const { [field]: _, ...rest } = searchFilter;
        setSearchFilter(rest);
    };

    return <div>Current filters: {JSON.stringify(searchFilter)}</div>;
}
```

### 3. URL Format

**Pattern:** `?s[FIELD_NAME]=VALUE&s[ANOTHER_FIELD]=VALUE`

**Examples:**
```
# Single value
?s[SEVERITY]=CRITICAL_VULNERABILITY_SEVERITY

# Multiple fields
?s[SEVERITY]=CRITICAL_VULNERABILITY_SEVERITY&s[Cluster]=production

# Multiple values for same field
?s[SEVERITY]=CRITICAL_VULNERABILITY_SEVERITY&s[SEVERITY]=IMPORTANT_VULNERABILITY_SEVERITY

# Date filter
?s[CVE%20Created%20Time]=>=2024-01-01

# Regex pattern
?s[Image]=r/.*nginx.*
```

---

## Backend Query Format

### 1. Query String Syntax

The backend expects a **concatenated string** with specific syntax:

**Format:** `FIELD:VALUE+FIELD2:VALUE2`

**Examples:**
```
# Simple query
SEVERITY:CRITICAL

# Multiple filters (AND logic)
SEVERITY:CRITICAL+Cluster:production

# Multiple values for same field (OR logic)
SEVERITY:CRITICAL,IMPORTANT

# Numeric conditions
CVSS:>=7

# Date conditions
CVE Created Time:>=2024-09-26

# Regex patterns
Image:r/.*nginx.*
CVE:r/.*log4j.*

# Boolean values
FIXABLE:true

# Negation
!Namespace:kube-system
```

### 2. Query Parsing

**Location:** `pkg/search/parser.go`

```go
// Backend parses queries like this:
func ParseQuery(query string) (*v1.Query, error) {
    // Splits on "+"
    pairs := strings.Split(query, "+")

    // Each pair is "FIELD:VALUE"
    for _, pair := range pairs {
        parts := strings.SplitN(pair, ":", 2)
        field := parts[0]  // "SEVERITY"
        value := parts[1]  // "CRITICAL"

        // Comma-separated values create OR condition
        values := strings.Split(value, ",")
    }
}
```

### 3. Special Operators

| Operator | Meaning | Example |
|----------|---------|---------|
| `:` | Field-value separator | `SEVERITY:CRITICAL` |
| `+` | AND (combines multiple filters) | `SEVERITY:CRITICAL+FIXABLE:true` |
| `,` | OR (multiple values for same field) | `SEVERITY:CRITICAL,IMPORTANT` |
| `!` | NOT (negation) | `!Namespace:kube-system` |
| `>=` | Greater than or equal | `CVSS:>=7` |
| `>` | Greater than | `CVE Created Time:>2024-01-01` |
| `<=` | Less than or equal | `CVSS:<=3` |
| `<` | Less than | `CVE Created Time:<2024-12-31` |
| `r/` | Regex pattern | `Image:r/.*nginx.*` |
| `=` | Exact match (default) | `SEVERITY:=CRITICAL` |

---

## Available Search Fields

### 1. Field Labels (Backend)

**Location:** `pkg/search/options.go`

The backend defines **hundreds** of searchable fields as constants:

```go
// CVE-related fields
CVE                = newFieldLabel("CVE")
CVEID              = newFieldLabel("CVE ID")
CVEType            = newFieldLabel("CVE Type")
CVEPublishedOn     = newFieldLabel("CVE Published On")
CVECreatedTime     = newFieldLabel("CVE Created Time")
CVESuppressed      = newFieldLabel("CVE Snoozed")
CVSS               = newFieldLabel("CVSS")
NVDCVSS            = newFieldLabel("NVD CVSS")
Severity           = newFieldLabel("Severity")
Fixable            = newFieldLabel("Fixable")
FixedBy            = newFieldLabel("Fixed By")

// Image-related fields
ImageName          = newFieldLabel("Image")
ImageSHA           = newFieldLabel("Image Sha")
ImageRegistry      = newFieldLabel("Image Registry")
ImageRemote        = newFieldLabel("Image Remote")
ImageTag           = newFieldLabel("Image Tag")
ImageOS            = newFieldLabel("Image OS")
ImageScanTime      = newFieldLabel("Image Scan Time")

// Cluster-related fields
Cluster            = newFieldLabel("Cluster")
ClusterID          = newFieldLabel("Cluster ID")
ClusterType        = newFieldLabel("Cluster Type")
ClusterPlatformType = newFieldLabel("Cluster Platform Type")

// Deployment-related fields
DeploymentName     = newFieldLabel("Deployment")
DeploymentID       = newFieldLabel("Deployment ID")
DeploymentLabel    = newFieldLabel("Deployment Label")

// Namespace-related fields
Namespace          = newFieldLabel("Namespace")
NamespaceID        = newFieldLabel("Namespace ID")

// Component-related fields
Component          = newFieldLabel("Component")
ComponentVersion   = newFieldLabel("Component Version")
ComponentSource    = newFieldLabel("Component Source")
```

### 2. Field Types

**Location:** `generated/api/v1/search_service.proto`

Each field has a **data type** that determines how it's queried:

```protobuf
enum SearchDataType {
    SEARCH_STRING = 0;      // Text search, supports regex
    SEARCH_BOOL = 1;        // true/false
    SEARCH_NUMERIC = 2;     // Supports <, >, >=, <=
    SEARCH_ENUM = 3;        // Predefined set of values
    SEARCH_DATETIME = 4;    // Date/time with range operators
    SEARCH_MAP = 5;         // Key-value pairs (labels, annotations)
}
```

### 3. Search Categories

**Location:** `generated/api/v1/search_service.proto`

Categories group related entities:

```protobuf
enum SearchCategory {
    ALERTS = 1;
    IMAGES = 2;
    IMAGE_COMPONENTS = 20;
    IMAGE_VULNERABILITIES = 35;
    IMAGE_VULNERABILITIES_V2 = 50;  // Flat data model (current)
    DEPLOYMENTS = 4;
    CLUSTERS = 8;
    NAMESPACES = 9;
    NODES = 10;
    NODE_COMPONENTS = 38;
    NODE_VULNERABILITIES = 36;
    SECRETS = 5;
    POLICIES = 3;
    // ... many more
}
```

**Important:** The frontend `searchCategory` in `CompoundSearchFilterEntity` must match these backend categories!

---

## Integration Points for AI Search

### 1. Where to Add AI Component

**Target Component:** `AdvancedFiltersToolbar`

**Location:** `apps/platform/src/Containers/Vulnerabilities/components/AdvancedFiltersToolbar.tsx`

**Current Structure:**
```tsx
function AdvancedFiltersToolbar({
    searchFilterConfig,
    searchFilter,
    onFilterChange,
    // ... other props
}) {
    return (
        <Toolbar>
            <ToolbarContent>
                <ToolbarGroup variant="filter-group">
                    {/* Existing CompoundSearchFilter */}
                    <CompoundSearchFilter
                        config={searchFilterConfig}
                        searchFilter={searchFilter}
                        onSearch={(payload) => {
                            // Updates searchFilter
                            onFilterChange(newFilter);
                        }}
                    />
                </ToolbarGroup>
                {/* Filter chips display here */}
            </ToolbarContent>
        </Toolbar>
    );
}
```

**AI Integration:**
```tsx
function AdvancedFiltersToolbar({
    searchFilterConfig,
    searchFilter,
    onFilterChange,
}) {
    return (
        <Toolbar>
            <ToolbarContent>
                <ToolbarGroup variant="filter-group">
                    {/* NEW: Natural Language Search */}
                    <NaturalLanguageSearchInput
                        searchFilterConfig={searchFilterConfig}
                        onFilterGenerated={(generatedFilter) => {
                            // Merge with existing filters or replace
                            onFilterChange({
                                ...searchFilter,
                                ...generatedFilter
                            });
                        }}
                    />

                    {/* OR divider */}
                    <Divider orientation="vertical" />

                    {/* Existing dropdown filters */}
                    <CompoundSearchFilter
                        config={searchFilterConfig}
                        searchFilter={searchFilter}
                        onSearch={onFilterChange}
                    />
                </ToolbarGroup>
            </ToolbarContent>
        </Toolbar>
    );
}
```

### 2. AI Component Interface

**Proposed API:**

```typescript
type NaturalLanguageSearchInputProps = {
    // Provides AI context about available filters
    searchFilterConfig: CompoundSearchFilterConfig;

    // Callback when AI generates a valid filter
    onFilterGenerated: (filter: SearchFilter) => void;

    // Optional: Initial query for pre-filling
    initialQuery?: string;

    // Optional: Confidence threshold (default 0.7)
    confidenceThreshold?: number;
};

function NaturalLanguageSearchInput({
    searchFilterConfig,
    onFilterGenerated,
    initialQuery,
    confidenceThreshold = 0.7,
}: NaturalLanguageSearchInputProps) {
    const [query, setQuery] = useState(initialQuery || '');
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const handleSearch = async () => {
        setIsLoading(true);

        try {
            // Call AI service
            const result = await parseNaturalLanguageQuery(
                query,
                searchFilterConfig
            );

            // Check confidence
            if (result.confidence < confidenceThreshold) {
                setError(`Low confidence (${result.confidence}). Please clarify.`);
                return;
            }

            // Generate SearchFilter object
            onFilterGenerated(result.searchFilter);
        } catch (err) {
            setError('Failed to parse query');
        } finally {
            setIsLoading(false);
        }
    };

    return (
        <TextInput
            placeholder="Type what you're looking for (e.g., 'critical CVEs in production')"
            value={query}
            onChange={(_, value) => setQuery(value)}
            onKeyPress={(e) => e.key === 'Enter' && handleSearch()}
        />
    );
}
```

### 3. AI Service Function

**Purpose:** Convert natural language → SearchFilter object

```typescript
type AIParseResult = {
    searchFilter: SearchFilter;
    confidence: number;
    reasoning?: string;
};

async function parseNaturalLanguageQuery(
    query: string,
    filterConfig: CompoundSearchFilterConfig
): Promise<AIParseResult> {
    // Build filter schema from config
    const filterSchema = buildFilterSchema(filterConfig);

    // Call AI API (Claude, OpenAI, or Ollama)
    const prompt = `
You are a search query parser for a security platform. Convert this natural language query into structured filters.

Available filters:
${JSON.stringify(filterSchema, null, 2)}

User query: "${query}"

Return JSON with this format:
{
  "searchFilter": {
    "FIELD_NAME": "value" or ["value1", "value2"]
  },
  "confidence": 0.0-1.0,
  "reasoning": "Explanation of interpretation"
}

Rules:
- Only use filters from the provided schema
- Use exact field names (searchTerm values)
- For severity, use full enum values like "CRITICAL_VULNERABILITY_SEVERITY"
- For dates, calculate relative dates (e.g., "last 7 days" → "${getDateDaysAgo(7)}")
- Multiple values for same field means OR logic
- Return confidence < 0.7 if query is ambiguous
`;

    const response = await callAIAPI(prompt);
    return JSON.parse(response);
}

function buildFilterSchema(config: CompoundSearchFilterConfig) {
    return config.flatMap(entity =>
        entity.attributes.map(attr => ({
            displayName: attr.displayName,
            searchTerm: attr.searchTerm,
            inputType: attr.inputType,
            options: attr.inputType === 'select'
                ? attr.inputProps.options
                : undefined,
        }))
    );
}
```

**Example AI Output:**

Input: `"critical CVEs in production cluster"`

```json
{
  "searchFilter": {
    "SEVERITY": "CRITICAL_VULNERABILITY_SEVERITY",
    "Cluster": "production"
  },
  "confidence": 0.95,
  "reasoning": "Mapped 'critical' to SEVERITY:CRITICAL and 'production cluster' to Cluster:production"
}
```

---

## Practical Examples

### Example 1: Simple Severity Filter

**User selects:** Severity = Critical

**Frontend:**
```typescript
const searchFilter: SearchFilter = {
    SEVERITY: "CRITICAL_VULNERABILITY_SEVERITY"
};
```

**URL:**
```
?s[SEVERITY]=CRITICAL_VULNERABILITY_SEVERITY
```

**Backend query string:**
```
SEVERITY:CRITICAL_VULNERABILITY_SEVERITY
```

### Example 2: Multiple Filters (AND)

**User selects:** Severity = Critical **AND** Fixable = true **AND** Cluster = production

**Frontend:**
```typescript
const searchFilter: SearchFilter = {
    SEVERITY: "CRITICAL_VULNERABILITY_SEVERITY",
    FIXABLE: "true",
    Cluster: "production"
};
```

**URL:**
```
?s[SEVERITY]=CRITICAL_VULNERABILITY_SEVERITY&s[FIXABLE]=true&s[Cluster]=production
```

**Backend query string:**
```
SEVERITY:CRITICAL_VULNERABILITY_SEVERITY+FIXABLE:true+Cluster:production
```

### Example 3: Multiple Values (OR)

**User selects:** Severity = Critical **OR** Important

**Frontend:**
```typescript
const searchFilter: SearchFilter = {
    SEVERITY: ["CRITICAL_VULNERABILITY_SEVERITY", "IMPORTANT_VULNERABILITY_SEVERITY"]
};
```

**URL:**
```
?s[SEVERITY]=CRITICAL_VULNERABILITY_SEVERITY&s[SEVERITY]=IMPORTANT_VULNERABILITY_SEVERITY
```

**Backend query string:**
```
SEVERITY:CRITICAL_VULNERABILITY_SEVERITY,IMPORTANT_VULNERABILITY_SEVERITY
```

### Example 4: Date Filter

**User selects:** CVE Created Time **After** 2024-09-26

**Frontend:**
```typescript
const searchFilter: SearchFilter = {
    "CVE Created Time": ">2024-09-26"
};
```

**URL:**
```
?s[CVE%20Created%20Time]=>2024-09-26
```

**Backend query string:**
```
CVE Created Time:>2024-09-26
```

### Example 5: Regex Pattern

**User types:** Image contains "nginx"

**Frontend:**
```typescript
const searchFilter: SearchFilter = {
    Image: "r/.*nginx.*"
};
```

**URL:**
```
?s[Image]=r/.*nginx.*
```

**Backend query string:**
```
Image:r/.*nginx.*
```

### Example 6: AI-Generated Complex Query

**User types:** "critical or high severity CVEs discovered in the last 7 days in production cluster that are fixable"

**AI generates:**
```typescript
const searchFilter: SearchFilter = {
    SEVERITY: ["CRITICAL_VULNERABILITY_SEVERITY", "IMPORTANT_VULNERABILITY_SEVERITY"],
    "CVE Created Time": `>=${getDateDaysAgo(7)}`,  // e.g., ">=2024-12-31"
    Cluster: "production",
    FIXABLE: "true"
};
```

**Resulting backend query:**
```
SEVERITY:CRITICAL_VULNERABILITY_SEVERITY,IMPORTANT_VULNERABILITY_SEVERITY+CVE Created Time:>=2024-12-31+Cluster:production+FIXABLE:true
```

---

## Key Files Reference

### Frontend Files

| File | Purpose | Key Exports |
|------|---------|-------------|
| `Components/CompoundSearchFilter/types.ts` | Type definitions | `CompoundSearchFilterConfig`, `CompoundSearchFilterEntity`, `CompoundSearchFilterAttribute` |
| `Components/CompoundSearchFilter/utils/utils.tsx` | Helper utilities | `makeFilterChipDescriptors`, `getAttribute`, `getEntity` |
| `Components/CompoundSearchFilter/components/CompoundSearchFilter.tsx` | Main filter component | `CompoundSearchFilter` component |
| `Containers/Vulnerabilities/searchFilterConfig.ts` | Filter configurations | All search filter configs for vulnerability views |
| `Containers/Vulnerabilities/components/AdvancedFiltersToolbar.tsx` | Toolbar component | Where AI component should be added |
| `hooks/useURLSearch.ts` | URL state management | `useURLSearch()` hook |
| `types/search.ts` | Search types | `SearchFilter` type |

### Backend Files (Reference Only)

| File | Purpose |
|------|---------|
| `pkg/search/options.go` | All searchable field definitions (FieldLabel constants) |
| `pkg/search/parser.go` | Query string parsing logic |
| `pkg/search/general_query_parser.go` | Converts query string to structured Query object |
| `pkg/search/fields.go` | Field metadata (type, category, etc.) |
| `central/search/service/service_impl.go` | Search service implementation |
| `generated/api/v1/search_service.proto` | Protobuf definitions for search API |

---

## Quick Start Checklist for AI Implementation

- [ ] **Understand searchFilterConfig** - Review vulnerability page configs
- [ ] **Study SearchFilter type** - Object mapping field names to values
- [ ] **Explore useURLSearch** - How filter state is managed
- [ ] **Examine backend query format** - String syntax with `:` and `+`
- [ ] **Review available fields** - Check `pkg/search/options.go` for valid field names
- [ ] **Create NaturalLanguageSearchInput component** - TextInput with AI integration
- [ ] **Build AI prompt** - Include filter schema for accurate parsing
- [ ] **Implement parseNaturalLanguageQuery** - Service function to call AI API
- [ ] **Integrate with AdvancedFiltersToolbar** - Add to existing toolbar
- [ ] **Test with common queries** - Validate AI output against expected SearchFilter objects

---

## Summary

The StackRox search system is built on:

1. **Declarative filter configs** (`CompoundSearchFilterConfig`) that define available filters
2. **SearchFilter objects** that represent active filters as key-value pairs
3. **URL-based state** managed by `useURLSearch` hook
4. **String-based query format** sent to backend (`FIELD:VALUE+FIELD2:VALUE2`)
5. **Extensive field catalog** defined in backend (`pkg/search/options.go`)

**For AI integration:**
- Extract filter schema from `searchFilterConfig`
- Feed schema to AI along with user query
- AI returns `SearchFilter` object
- Pass to `setSearchFilter()` to update UI and trigger search
- Existing infrastructure handles URL updates, chip display, and backend communication

**The AI's job is simple:** Natural language → SearchFilter object. Everything else already works!
