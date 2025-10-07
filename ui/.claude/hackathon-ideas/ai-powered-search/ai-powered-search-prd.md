# AI-Powered Natural Language Search

**Status:** Idea
**Estimated Effort:** 1 day hackathon project
**Category:** UX Enhancement / AI Integration

## Overview

Replace tedious dropdown-based filtering with natural language search queries. Users type what they're looking for in plain English, and AI automatically translates it into structured search filters.

## The Problem

- Current filter system requires multiple dropdown selections (Entity → Attribute → Value)
- Users need to know exact filter categories and field names
- Building complex filters with multiple criteria is time-consuming
- New users struggle to discover available filter options
- Cognitive overhead remembering filter structure across different pages

## The Solution

An AI-powered text input that converts natural language queries into StackRox's existing search filter system. Users describe what they want, and the system figures out the filters.

---

## How It Works

### User Experience Flow

1. **Type natural language query**
   ```
   "Show me critical CVEs in production cluster"
   ```

2. **AI parses and translates**
   - Identifies search intent: CVE severity + cluster filter
   - Maps to filter structure: `{SEVERITY: 'CRITICAL', Cluster: 'production'}`
   - Validates against available filter schema

3. **Updates UI automatically**
   - URL updates: `?s[SEVERITY]=CRITICAL&s[Cluster]=production`
   - FilterChips appear showing active filters
   - Results refresh based on filters

4. **User can refine**
   - Add more criteria in natural language: "and fixable status is true"
   - Or use traditional dropdowns to fine-tune
   - Clear all or individual chips as normal

### AI Processing Pipeline

```typescript
Natural Language Query
    ↓
[AI Parsing Service]
    ↓
{
  intent: "filter",
  entities: {
    severity: "CRITICAL",
    cluster: "production"
  },
  confidence: 0.95
}
    ↓
[Filter Mapper]
    ↓
SearchFilter Object
{
  "SEVERITY": "CRITICAL",
  "Cluster": "production"
}
    ↓
[URL State Update]
    ↓
Backend Query: "SEVERITY:CRITICAL+Cluster:production"
```

---

## Example Queries

### Example 1: CVE Search
**Input:** "critical CVEs discovered in the last 7 days"
**Output:**
```typescript
{
  "SEVERITY": "CRITICAL",
  "CVE Created Time": "2024-09-26" // date 7 days ago
}
```

### Example 2: Deployment Search
**Input:** "deployments with log4j vulnerabilities in default namespace"
**Output:**
```typescript
{
  "CVE": "r/.*log4j.*",
  "Namespace": "default"
}
```

### Example 3: Image Search
**Input:** "show fixable vulnerabilities in nginx images"
**Output:**
```typescript
{
  "FIXABLE": "true",
  "Image": "r/.*nginx.*"
}
```

### Example 4: Complex Multi-Filter
**Input:** "production cluster deployments with critical or high severity CVEs that are fixable"
**Output:**
```typescript
{
  "Cluster": "production",
  "SEVERITY": ["CRITICAL", "HIGH"],
  "FIXABLE": "true"
}
```

### Example 5: Time-Based
**Input:** "CVEs with CVSS score greater than 7 from this month"
**Output:**
```typescript
{
  "CVSS": ">=7",
  "CVE Created Time": ">=2024-09-01" // first day of current month
}
```

---

## Configuration & Environment

### Environment Variables

```bash
# AI Provider Selection (ollama for local testing, anthropic for cloud)
ROX_AI_SEARCH_PROVIDER=ollama|anthropic|openai

# API Keys (only needed for cloud providers)
ROX_AI_SEARCH_ANTHROPIC_KEY=sk-ant-...
ROX_AI_SEARCH_OPENAI_KEY=sk-...

# Ollama Configuration (for local testing)
ROX_AI_SEARCH_OLLAMA_URL=http://localhost:11434
ROX_AI_SEARCH_OLLAMA_MODEL=llama3.2:latest

# Query Limits
ROX_AI_SEARCH_MAX_QUERY_LENGTH=500  # characters
ROX_AI_SEARCH_CONFIDENCE_THRESHOLD=0.7  # 0.0-1.0
```

### Feature Flag

Add to `apps/platform/src/types/featureFlag.ts`:
```typescript
| 'ROX_AI_POWERED_SEARCH'  // Frontend-only AI-powered natural language search
```

### MVP Scope

- **Target Page**: Vulnerability Results page (`WorkloadCvesOverviewPage.tsx`)
- **AI Providers**: Ollama (development/testing) + Anthropic Claude (production)
- **Provider Switching**: Environment variable controlled
- **Caching**: None for MVP (future enhancement)
- **Confidence Score**: Displayed in UI when <0.9

---

## Technical Implementation

### Architecture

```
┌─────────────────────────────────┐
│  NaturalLanguageSearchInput     │
│  - Input sanitization           │
│  - Max length validation        │
│  - Confidence score display     │
└──────────────┬──────────────────┘
               │
               ↓
┌─────────────────────────────────┐
│  AI Provider Router             │
│  - Ollama (local testing)       │
│  - Anthropic Claude (prod)      │
│  - OpenAI GPT-4 (fallback)      │
│  - Error handling & fallback    │
└──────────────┬──────────────────┘
               │
               ↓
┌─────────────────────────────────┐
│  AI Query Parser Service        │
│  - Prompt Engineering           │
│  - Filter Schema Validation     │
│  - Confidence scoring           │
└──────────────┬──────────────────┘
               │
               ↓
┌─────────────────────────────────┐
│  Filter Mapper Utility          │
│  - Maps AI output to            │
│    SearchFilter format          │
│  - Handles edge cases           │
│  - Validates against schema     │
└──────────────┬──────────────────┘
               │
               ↓
┌─────────────────────────────────┐
│  Existing Filter System         │
│  - useURLSearch hook            │
│  - SearchFilterChips            │
│  - AdvancedFiltersToolbar       │
└─────────────────────────────────┘
```

### Tech Stack

**Frontend:**
- React + TypeScript
- PatternFly TextInput, Alert, Label components
- Existing filter infrastructure (no changes needed)
- Feature flag: `ROX_AI_POWERED_SEARCH`

**AI Integration:**
- **Development**: Ollama (local, free, fast iteration)
- **Production**: Anthropic Claude (accurate, reliable)
- **Fallback**: OpenAI GPT-4 (if Anthropic unavailable)
- Structured output for reliable parsing
- Filter schema provided in prompt for accuracy

**Input Validation:**
- Max query length: 500 characters
- Sanitization: Remove special characters, SQL injection prevention
- Rate limiting: Per-user limits (future)

**State Management:**
- Existing `useURLSearch` hook
- No new state management needed
- Integrates seamlessly with current system

### Core Components

#### 1. NaturalLanguageSearchInput.tsx
```typescript
import { useState } from 'react';
import { TextInput, Spinner, Alert, Label } from '@patternfly/react-core';
import { sanitizeInput } from 'services/inputSanitizer';

const MAX_QUERY_LENGTH = 500;
const CONFIDENCE_THRESHOLD = 0.7;

type Props = {
  searchFilterConfig: CompoundSearchFilterConfig;
  onFilterGenerated: (filter: SearchFilter) => void;
}

function NaturalLanguageSearchInput({ searchFilterConfig, onFilterGenerated }) {
  const [query, setQuery] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [confidence, setConfidence] = useState<number | null>(null);
  const [needsClarification, setNeedsClarification] = useState(false);

  const handleSearch = async () => {
    // Input validation
    if (query.length > MAX_QUERY_LENGTH) {
      setError(`Query too long. Maximum ${MAX_QUERY_LENGTH} characters.`);
      return;
    }

    const sanitizedQuery = sanitizeInput(query);
    if (!sanitizedQuery) {
      setError('Invalid query. Please try again.');
      return;
    }

    setIsLoading(true);
    setError(null);
    setNeedsClarification(false);

    try {
      const result = await parseNaturalLanguageQuery(sanitizedQuery, searchFilterConfig);

      setConfidence(result.confidence);

      // Low confidence - ask for clarification
      if (result.confidence < CONFIDENCE_THRESHOLD) {
        setNeedsClarification(true);
        return;
      }

      onFilterGenerated(result.searchFilter);
    } catch (err) {
      setError('Failed to parse query. Please try traditional filters.');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <>
      <div style={{ position: 'relative', display: 'flex', alignItems: 'center', gap: '8px' }}>
        <TextInput
          placeholder="Type what you're looking for (e.g., 'critical CVEs in production')"
          value={query}
          onChange={(_, value) => setQuery(value)}
          onKeyPress={(e) => e.key === 'Enter' && handleSearch()}
          maxLength={MAX_QUERY_LENGTH}
          validated={error ? 'error' : 'default'}
        />
        {isLoading && <Spinner size="md" />}
        {confidence !== null && confidence < 0.9 && (
          <Label color="orange">Confidence: {(confidence * 100).toFixed(0)}%</Label>
        )}
      </div>

      {error && (
        <Alert variant="danger" title="Error" isInline>
          {error}
        </Alert>
      )}

      {needsClarification && (
        <Alert variant="warning" title="Could you clarify?" isInline>
          I'm not quite sure what you're looking for (confidence: {(confidence! * 100).toFixed(0)}%).
          Try being more specific or use the dropdown filters below.
        </Alert>
      )}
    </>
  );
}
```

#### 2. AI Query Parser Service
```typescript
// services/aiSearchParserService.ts

async function parseNaturalLanguageQuery(
  query: string,
  filterConfig: CompoundSearchFilterConfig
): Promise<SearchFilter> {
  const filterSchema = buildFilterSchema(filterConfig);

  const prompt = `
You are a search query parser for a security platform. Convert this natural language query into structured filters.

Available filters:
${JSON.stringify(filterSchema, null, 2)}

User query: "${query}"

Return JSON with this format:
{
  "searchFilter": {
    "FILTER_NAME": "value" or ["value1", "value2"]
  },
  "confidence": 0.0-1.0
}
`;

  const response = await callClaudeAPI(prompt);
  return response.searchFilter;
}

function buildFilterSchema(config: CompoundSearchFilterConfig) {
  // Extract all available filters from config
  return config.flatMap(entity =>
    entity.attributes.map(attr => ({
      displayName: attr.displayName,
      searchTerm: attr.searchTerm,
      inputType: attr.inputType,
      options: attr.inputType === 'select' ? attr.inputProps.options : undefined
    }))
  );
}
```

#### 3. Integration with AdvancedFiltersToolbar

```typescript
// Modify existing AdvancedFiltersToolbar.tsx

function AdvancedFiltersToolbar({ searchFilterConfig, searchFilter, onFilterChange, ... }) {
  const handleNaturalLanguageSearch = (generatedFilter: SearchFilter) => {
    // Merge with existing filters or replace
    onFilterChange({ ...searchFilter, ...generatedFilter });
  };

  return (
    <Toolbar>
      <ToolbarContent>
        <ToolbarGroup variant="filter-group">
          {/* NEW: Natural language search input */}
          <NaturalLanguageSearchInput
            searchFilterConfig={searchFilterConfig}
            onFilterGenerated={handleNaturalLanguageSearch}
          />

          {/* OR divider */}
          <Divider orientation="vertical" />

          {/* Existing CompoundSearchFilter */}
          <CompoundSearchFilter ... />
        </ToolbarGroup>

        {/* Rest of toolbar... */}
      </ToolbarContent>
    </Toolbar>
  );
}
```

### Project Structure

```
apps/platform/src/
├── Components/
│   └── NaturalLanguageSearch/
│       ├── NaturalLanguageSearchInput.tsx
│       ├── NaturalLanguageSearchInput.test.tsx
│       ├── ConfidenceScoreLabel.tsx
│       ├── ClarificationAlert.tsx
│       └── types.ts
├── services/
│   ├── aiProviders/
│   │   ├── aiProviderRouter.ts          # Routes to correct provider
│   │   ├── anthropicProvider.ts         # Claude integration
│   │   ├── openaiProvider.ts            # OpenAI integration
│   │   ├── ollamaProvider.ts            # Ollama integration
│   │   └── types.ts                     # Shared provider types
│   ├── aiSearchParserService.ts         # Main parsing logic
│   ├── aiSearchParserService.test.ts
│   ├── filterSchemaBuilder.ts
│   └── inputSanitizer.ts                # Input validation & sanitization
├── utils/
│   └── aiSearchUtils.ts
└── test-data/
    └── aiSearchTestQueries.ts           # Library of test queries with expected outputs
```

### Prompt Engineering Strategy

The key to accuracy is a well-structured prompt with:

1. **Filter Schema Context**
   - All available filters with types and options
   - Example mappings for common queries
   - Edge case handling instructions

2. **Output Format Specification**
   ```json
   {
     "searchFilter": {
       "CVE": "CVE-2024-1234",
       "SEVERITY": ["CRITICAL", "HIGH"]
     },
     "confidence": 0.95,
     "reasoning": "Mapped 'critical or high' to SEVERITY filter"
   }
   ```

3. **Validation Rules**
   - Only use filters from provided schema
   - Return empty filter if query is unclear
   - Handle dates, ranges, regex patterns correctly
   - Support multiple values for same filter

### Test Query Library

A comprehensive test suite with expected outputs for validation and regression testing:

```typescript
// test-data/aiSearchTestQueries.ts

export const testQueries = [
  {
    query: "critical CVEs in production cluster",
    expected: {
      SEVERITY: "CRITICAL",
      Cluster: "production"
    },
    minConfidence: 0.9
  },
  {
    query: "fixable vulnerabilities in nginx images",
    expected: {
      FIXABLE: "true",
      Image: "r/.*nginx.*"
    },
    minConfidence: 0.85
  },
  {
    query: "CVEs discovered in the last 7 days",
    expected: {
      "CVE Created Time": `>=${getDateDaysAgo(7)}`
    },
    minConfidence: 0.8
  },
  {
    query: "critical or high severity deployments that are fixable",
    expected: {
      SEVERITY: ["CRITICAL", "HIGH"],
      FIXABLE: "true"
    },
    minConfidence: 0.9
  },
  {
    query: "log4j vulnerabilities",
    expected: {
      CVE: "r/.*log4j.*"
    },
    minConfidence: 0.95
  }
  // ... 15+ more test cases
];
```

**Test Coverage Goals:**
- Simple filters (1 criterion): 5+ queries
- Complex filters (2-3 criteria): 5+ queries
- Date/time filters: 3+ queries
- Regex patterns: 3+ queries
- Multi-value filters: 3+ queries
- Edge cases (ambiguous, typos): 5+ queries

**Usage:**
- Automated tests run against Ollama (fast, free, local)
- CI/CD integration to catch prompt regressions
- Manual validation during prompt refinement
- Demo preparation with proven working queries

---

## Day 1 Implementation Plan

### Phase 1: Core Parser (3 hours)
- Set up AI API integration (Claude or OpenAI)
- Build filter schema from searchFilterConfig
- Create prompt template with filter schema
- Implement basic query → filter conversion
- Add confidence scoring

### Phase 2: UI Integration (2 hours)
- Create NaturalLanguageSearchInput component
- Add to AdvancedFiltersToolbar
- Integrate with existing useURLSearch hook
- Show loading state during AI processing
- Display generated filters as chips

### Phase 3: Testing & Refinement (2 hours)
- Test common query patterns
- Refine prompt for better accuracy
- Handle edge cases (ambiguous queries, typos)
- Add error handling and fallbacks
- User feedback for low confidence results

### Phase 4: Polish (1 hour)
- Add example queries placeholder text
- Keyboard shortcuts (Cmd+K to focus)
- Query history (optional)
- Demo preparation

---

## Demo Flow

**Hackathon Presentation Scenario:**

1. **Show current workflow** (30 seconds)
   - Multiple clicks to build filter: Cluster → production, Severity → Critical, Status → Fixable
   - "This takes 6+ clicks and requires knowing exact filter names"

2. **Show AI-powered search** (30 seconds)
   - Type: "critical fixable CVEs in production cluster"
   - Press Enter
   - Watch filters populate automatically
   - Results appear instantly

3. **Show complex query** (30 seconds)
   - Type: "nginx deployments with log4j vulnerabilities from last week"
   - Watch it handle: regex matching, date calculation, multiple entities

4. **Show refinement** (15 seconds)
   - Add: "and severity is critical or high"
   - Filters update incrementally

5. **Wow factor** (15 seconds)
   - "Works across all pages that use filters"
   - "Learns from available filter schema automatically"
   - "No training data needed - just schema-aware prompts"

**Key Demo Points:**
- Speed: Seconds vs minutes
- Accessibility: Natural language vs learning filter names
- Flexibility: Complex queries in one go
- Integration: Works with existing system seamlessly

---

## Success Metrics

### For Hackathon MVP
- ✅ Converts 20+ common query patterns correctly (validated via test library)
- ✅ >80% accuracy on test queries (using Ollama for testing)
- ✅ <2 second response time from Ollama, <5 seconds from cloud APIs
- ✅ Zero changes to existing filter system
- ✅ Works on WorkloadCvesOverviewPage (Vulnerability Results)
- ✅ Confidence score displayed when <0.9
- ✅ Graceful fallback when API unavailable
- ✅ Input validation and sanitization working
- ✅ Feature flag integration complete

### If This Became Real
- Reduced average time to create complex filters (measure: analytics)
- Increased filter usage (more users filtering results)
- Reduced support requests about "how to filter"
- Positive user feedback (NPS surveys)
- Feature adoption rate >30% within first month

---

## Challenges & Mitigations

### Challenge 1: AI Hallucination (inventing filters)
**Mitigation:**
- Provide explicit filter schema in prompt
- Validate output against schema
- Return only filters that exist
- Display confidence score in UI (PatternFly Label with warning color when <0.9)
- Allow manual correction via traditional filters

### Challenge 2: Ambiguous Queries (Low Confidence)
**Mitigation:**
- Show confidence score in UI when <0.9
- Ask clarifying questions via PatternFly Alert component
- Display parsed filters for user verification before applying
- Suggest query refinements: "Did you mean: ..."
- Show example queries in placeholder/help text

### Challenge 3: API Unavailable or Latency
**Mitigation:**
- **Automatic Fallback**: If Anthropic/OpenAI fails → fallback to Ollama (local)
- **Error Handling**: Clear error messages in PatternFly Alert
- **Loading States**: PatternFly Spinner during processing
- **Timeout**: 10 second timeout, then show error + fallback option
- **Graceful Degradation**: Traditional filters always available

### Challenge 4: Cost & Development Speed
**Mitigation:**
- **Ollama for Development**: Free local testing with llama3.2
- **Environment-based Switching**: Easy toggle between providers
- **Debounced Input**: Only trigger on Enter key press
- **No Caching in MVP**: Reduces complexity, add later if needed
- **Rate Limiting**: Future enhancement, not needed for hackathon

### Challenge 5: Input Security & Validation
**Mitigation:**
- **Max Length**: 500 character limit (enforced in UI and service)
- **Sanitization**: Remove/escape special characters before AI processing
- **Injection Prevention**: Validate against filter schema after AI response
- **XSS Protection**: Sanitize all user input before rendering

### Challenge 6: Complex Filter Logic (AND/OR)
**Mitigation:**
- Start simple (AND only for MVP)
- Clearly document limitations in UI help text
- Show combined filter in chips for transparency
- Future: Support advanced syntax like "A AND (B OR C)"

---

## Future Enhancements

### Post-Hackathon Ideas

1. **Query Suggestions/Autocomplete**
   - Show common queries based on page context
   - Learn from user's previous searches
   - "People also searched for..."

2. **Multi-Language Support**
   - Spanish: "mostrar CVEs críticos en producción"
   - French: "afficher les CVE critiques en production"
   - AI naturally handles this with minimal changes

3. **Voice Input**
   - Speak your search query
   - Perfect for accessibility
   - Mobile-friendly

4. **Query Templates**
   - Save favorite queries
   - Share with team
   - "My saved searches" sidebar

5. **Smart Defaults**
   - Context-aware suggestions
   - "Based on your role, you might want to filter by..."
   - Learn from team patterns

6. **Advanced Query Syntax**
   - Support boolean operators: "A AND (B OR C)"
   - Exclusions: "NOT namespace=test"
   - Ranges: "CVSS between 7 and 9"

7. **Query Explanation**
   - Show reasoning: "I interpreted 'recent' as 'last 7 days'"
   - Suggest improvements: "Try 'last 30 days' for more results"
   - Help users learn filter names

---

## Integration with Existing Pages

### Works Out of Box On:
- ✅ Vulnerability pages (CVEs, Images, Deployments, Nodes)
- ✅ Compliance pages
- ✅ Policy violations
- ✅ Cluster management
- ✅ Any page using AdvancedFiltersToolbar

### Requires Minimal Config:
1. Pass `searchFilterConfig` to NaturalLanguageSearch
2. Hook up to existing `onFilterChange` callback
3. That's it!

### Example Integration:

```typescript
// In WorkloadCvesOverviewPage.tsx
<AdvancedFiltersToolbar
  searchFilterConfig={[
    imageCVESearchFilterConfig,
    imageSearchFilterConfig,
    clusterSearchFilterConfig,
    // ... existing configs
  ]}
  searchFilter={searchFilter}
  onFilterChange={(newFilter) => {
    setSearchFilter(newFilter); // Existing logic, no changes!
  }}
  enableNaturalLanguageSearch={true} // NEW: Feature flag
/>
```

---

## Resources Needed

### For Hackathon

**Development Environment:**
- **Ollama** (local AI for development/testing)
  - Install: `brew install ollama` (macOS) or `curl -fsSL https://ollama.com/install.sh | sh` (Linux)
  - Pull model: `ollama pull llama3.2`
  - Free, fast iteration, no API costs during development

**Production/Demo (Optional):**
- **Anthropic Claude API** (recommended for demo)
  - API key from https://console.anthropic.com
  - Budget: ~$5-10 for testing/demo
- **OpenAI API** (fallback option)
  - API key from https://platform.openai.com
  - Budget: ~$5-10 for testing/demo

**Time Investment:**
  - 1 developer for 1 day
  - OR 2 developers for 4 hours each (pair programming)

**Skills Required:**
  - TypeScript/React (existing UI patterns)
  - Prompt engineering (AI experience helpful but not required)
  - Understanding of StackRox filter system (covered in this doc)
  - Basic REST API integration (Ollama/Claude/OpenAI)

### No Additional Infrastructure
- Uses existing filter system
- No new backend services
- No database changes
- No new state management

---

## Comparison: Current vs AI-Powered

| Task | Current System | AI-Powered Search | Time Saved |
|------|---------------|-------------------|------------|
| Simple filter (1 criterion) | 3 clicks + typing | Type + Enter | 50% |
| Complex filter (3+ criteria) | 9+ clicks + typing | Type + Enter | 80% |
| Unknown filter name | Trial & error, docs | Just describe it | 90% |
| Date ranges | Calendar picker clicks | "last week", "this month" | 70% |
| Regex patterns | Know regex syntax | "contains log4j" | 85% |
| Learning curve | Medium-High | Low | N/A |

**Result:** Average 70% reduction in time to create filters

---

## Why This Has Real Value

### 1. **Dramatic UX Improvement**
- Reduces cognitive load (don't need to know filter names)
- Faster workflows (seconds instead of minutes)
- More accessible to new users
- Natural interaction model

### 2. **Increased Filter Adoption**
- Users who avoid filters will try them
- Discover features they didn't know existed
- Power users can go even faster

### 3. **Competitive Differentiation**
- No other security platform has this
- Modern, AI-forward approach
- Shows innovation leadership

### 4. **Scalable Foundation**
- Works across entire product
- Easy to extend to new filter types
- Could expand to other AI features

### 5. **Low Implementation Risk**
- Non-invasive (doesn't change existing code)
- Can be feature-flagged
- Easy to A/B test
- Falls back gracefully if AI unavailable

---

## Conclusion

AI-Powered Natural Language Search transforms StackRox filtering from a tedious multi-click process into a simple, intuitive text-based experience. By leveraging Claude/GPT to parse natural language and map it to existing filter structures, we can dramatically improve UX without changing any backend systems.

**Perfect Hackathon Project Because:**
- ✅ Achievable in 1 day (80% solution)
- ✅ Highly demonstrable (impressive live demo)
- ✅ Uses cutting-edge AI technology
- ✅ Solves real user pain point
- ✅ Could become actual product feature
- ✅ No infrastructure changes needed

**Best Part:** Works across the entire UI wherever filters are used. Build once, benefit everywhere.

**Demo Potential:** Very high - visual, fast, magical feeling when it works.

**Learning Opportunities:** AI integration, prompt engineering, UX innovation, working with existing complex systems.

---

## Next Steps

### To Build This:

1. **Get API access** (Claude or OpenAI)
2. **Set up basic component** (TextInput + loading state)
3. **Build filter schema extractor** from searchFilterConfig
4. **Create prompt template** with schema + examples
5. **Wire up to existing filter system** (useURLSearch hook)
6. **Test & refine** prompt for accuracy
7. **Polish UI** and prepare demo

### To Pitch This:

1. Show current multi-click filter workflow (painfully slow)
2. Demo natural language: "critical CVEs in production" → instant filters
3. Show complex query: "nginx deployments with log4j from last week"
4. Explain: Works everywhere filters exist, zero backend changes
5. Reveal: Built in 1 day, could ship in 1 sprint

**Ready to revolutionize search? Let's make filtering fun! 🔍🤖✨**
