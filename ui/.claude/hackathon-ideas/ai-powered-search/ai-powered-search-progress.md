# AI-Powered Natural Language Search - Progress Tracker

**Status:** Not Started
**Target Date:** TBD
**MVP Target Page:** WorkloadCvesOverviewPage (Vulnerability Results)

---

## Progress Overview

| Phase | Status | Completion |
|-------|--------|------------|
| Phase 1: Foundation & Setup | ⬜ Not Started | 0% |
| Phase 2: AI Provider Integration | ⬜ Not Started | 0% |
| Phase 3: Core Search Parsing | ⬜ Not Started | 0% |
| Phase 4: UI Components | ⬜ Not Started | 0% |
| Phase 5: Integration & Testing | ⬜ Not Started | 0% |
| Phase 6: Polish & Demo | ⬜ Not Started | 0% |

**Overall Progress:** 0/87 tasks complete (0%)

---

## Phase 1: Foundation & Setup (15 tasks)

**Goal:** Establish the basic infrastructure, environment configuration, and project structure.

**Status:** ⬜ Not Started | **Progress:** 0/15

### Environment Configuration
- [ ] Install Ollama locally (`brew install ollama` or equivalent)
- [ ] Pull llama3.2 model (`ollama pull llama3.2`)
- [ ] Test Ollama is running (`curl http://localhost:11434/api/generate`)
- [ ] (Optional) Get Anthropic Claude API key for production demo
- [ ] Create `.env.local` with environment variables:
  - `ROX_AI_SEARCH_PROVIDER=ollama`
  - `ROX_AI_SEARCH_OLLAMA_URL=http://localhost:11434`
  - `ROX_AI_SEARCH_OLLAMA_MODEL=llama3.2:latest`
  - `ROX_AI_SEARCH_MAX_QUERY_LENGTH=500`
  - `ROX_AI_SEARCH_CONFIDENCE_THRESHOLD=0.7`

### Feature Flag Setup
- [ ] Add `ROX_AI_POWERED_SEARCH` to `apps/platform/src/types/featureFlag.ts`
- [ ] Add feature flag check utility/hook if needed
- [ ] Document feature flag usage in code comments

### Project Structure
- [ ] Create directory: `apps/platform/src/Components/NaturalLanguageSearch/`
- [ ] Create directory: `apps/platform/src/services/aiProviders/`
- [ ] Create directory: `apps/platform/src/test-data/`
- [ ] Create base type definitions file: `apps/platform/src/Components/NaturalLanguageSearch/types.ts`
- [ ] Create provider types file: `apps/platform/src/services/aiProviders/types.ts`

### Documentation
- [ ] Add inline code comments explaining AI provider architecture
- [ ] Document environment variable usage in project README or setup guide

---

## Phase 2: AI Provider Integration (18 tasks)

**Goal:** Build the AI provider abstraction layer with support for Ollama, Anthropic, and OpenAI, with fallback logic.

**Status:** ⬜ Not Started | **Progress:** 0/18

### Provider Types & Interfaces
- [ ] Define `AIProvider` interface in `services/aiProviders/types.ts`
  - `generateCompletion(prompt: string): Promise<AIResponse>`
  - `isAvailable(): Promise<boolean>`
  - `getName(): string`
- [ ] Define `AIResponse` type with `content`, `confidence`, `reasoning` fields
- [ ] Define provider configuration types

### Ollama Provider
- [ ] Create `services/aiProviders/ollamaProvider.ts`
- [ ] Implement `generateCompletion()` using Ollama REST API
- [ ] Add error handling for connection failures
- [ ] Add timeout handling (10 seconds)
- [ ] Test basic query: "What is 2+2?" → verify JSON response

### Anthropic Provider (Optional for Demo)
- [ ] Create `services/aiProviders/anthropicProvider.ts`
- [ ] Implement `generateCompletion()` using Anthropic Messages API
- [ ] Add API key validation
- [ ] Add error handling for rate limits, auth errors
- [ ] Test basic query if API key available

### OpenAI Provider (Optional Fallback)
- [ ] Create `services/aiProviders/openaiProvider.ts`
- [ ] Implement `generateCompletion()` using OpenAI Chat API
- [ ] Add API key validation and error handling

### Provider Router
- [ ] Create `services/aiProviders/aiProviderRouter.ts`
- [ ] Implement provider selection logic based on `ROX_AI_SEARCH_PROVIDER` env var
- [ ] Implement fallback cascade: Cloud → Ollama → Error
- [ ] Add logging for provider selection and fallback events
- [ ] Test provider switching via environment variable

---

## Phase 3: Core Search Parsing (21 tasks)

**Goal:** Build the AI query parsing service with filter schema extraction, prompt engineering, and input validation.

**Status:** ⬜ Not Started | **Progress:** 0/21

### Input Validation & Sanitization
- [ ] Create `services/inputSanitizer.ts`
- [ ] Implement max length validation (500 chars)
- [ ] Implement XSS prevention (strip HTML tags, escape special chars)
- [ ] Implement SQL injection prevention for filter values
- [ ] Add unit tests for sanitization edge cases

### Filter Schema Builder
- [ ] Create `services/filterSchemaBuilder.ts`
- [ ] Implement `buildFilterSchema(config: CompoundSearchFilterConfig)`
- [ ] Extract filter names, types, and available options
- [ ] Format schema for AI prompt (JSON structure)
- [ ] Test with WorkloadCvesOverviewPage filter config

### Prompt Engineering
- [ ] Design base prompt template in `services/aiSearchParserService.ts`
- [ ] Include filter schema in prompt
- [ ] Add example query→filter mappings (3-5 examples)
- [ ] Define strict JSON output format requirements
- [ ] Add instructions for confidence scoring
- [ ] Add instructions for handling ambiguous queries
- [ ] Add validation rules (only use provided filters, etc.)

### AI Query Parser Service
- [ ] Create `services/aiSearchParserService.ts`
- [ ] Implement `parseNaturalLanguageQuery(query, filterConfig): Promise<ParseResult>`
- [ ] Integrate with AI provider router
- [ ] Parse AI response and extract `searchFilter`, `confidence`, `reasoning`
- [ ] Validate AI output against filter schema
- [ ] Handle malformed AI responses gracefully
- [ ] Add comprehensive error handling

### Test Query Library
- [ ] Create `test-data/aiSearchTestQueries.ts`
- [ ] Add 5+ simple filter test queries
- [ ] Add 5+ complex multi-filter test queries
- [ ] Add 3+ date/time-based test queries
- [ ] Add 3+ regex pattern test queries

---

## Phase 4: UI Components (16 tasks)

**Goal:** Build user-facing React components with PatternFly, including input, loading states, confidence display, and error handling.

**Status:** ⬜ Not Started | **Progress:** 0/16

### Component Types
- [ ] Define component prop types in `Components/NaturalLanguageSearch/types.ts`
- [ ] Define state types for component internal state

### NaturalLanguageSearchInput Component
- [ ] Create `Components/NaturalLanguageSearch/NaturalLanguageSearchInput.tsx`
- [ ] Implement TextInput with placeholder and max length
- [ ] Add loading state with PatternFly Spinner
- [ ] Add Enter key handler to trigger search
- [ ] Integrate `sanitizeInput()` before sending query
- [ ] Integrate `parseNaturalLanguageQuery()` on search

### Confidence Score Display
- [ ] Create `Components/NaturalLanguageSearch/ConfidenceScoreLabel.tsx`
- [ ] Display confidence as percentage when <0.9
- [ ] Use PatternFly Label with orange color for low confidence
- [ ] Integrate into NaturalLanguageSearchInput

### Error & Clarification Handling
- [ ] Create `Components/NaturalLanguageSearch/ClarificationAlert.tsx`
- [ ] Display PatternFly Alert when confidence <0.7
- [ ] Show helpful clarification message
- [ ] Add error Alert for API failures
- [ ] Add error Alert for input validation failures

### Component Testing
- [ ] Create `Components/NaturalLanguageSearch/NaturalLanguageSearchInput.test.tsx`
- [ ] Test input validation (max length, sanitization)
- [ ] Test loading state display
- [ ] Test error handling

---

## Phase 5: Integration & Testing (14 tasks)

**Goal:** Integrate the NaturalLanguageSearchInput into WorkloadCvesOverviewPage, test end-to-end flows, and refine based on results.

**Status:** ⬜ Not Started | **Progress:** 0/14

### WorkloadCvesOverviewPage Integration
- [ ] Find WorkloadCvesOverviewPage component file
- [ ] Identify AdvancedFiltersToolbar usage
- [ ] Add feature flag check for `ROX_AI_POWERED_SEARCH`
- [ ] Import NaturalLanguageSearchInput component
- [ ] Pass `searchFilterConfig` prop
- [ ] Implement `onFilterGenerated` callback to update URL search params
- [ ] Test that existing filters still work

### End-to-End Testing
- [ ] Test simple query: "critical CVEs" → verify SEVERITY filter applied
- [ ] Test complex query: "critical fixable CVEs in production" → verify multiple filters
- [ ] Test date query: "CVEs from last 7 days" → verify date filter
- [ ] Test regex query: "log4j vulnerabilities" → verify CVE regex filter
- [ ] Test low confidence scenario → verify clarification alert appears
- [ ] Test API failure scenario → verify error handling and fallback

### Refinement & Edge Cases
- [ ] Run all test queries from test library and measure accuracy
- [ ] Refine prompt based on failed queries
- [ ] Add edge case handling for typos and ambiguous terms
- [ ] Verify performance: <2s for Ollama, <5s for cloud APIs

---

## Phase 6: Polish & Demo (3 tasks)

**Goal:** Final polish, demo preparation, and documentation.

**Status:** ⬜ Not Started | **Progress:** 0/3

### UI Polish
- [ ] Add keyboard shortcut (Cmd/Ctrl+K) to focus search input (optional)
- [ ] Add helpful placeholder examples
- [ ] Ensure accessibility (ARIA labels, screen reader support)
- [ ] Test responsive layout on different screen sizes

### Demo Preparation
- [ ] Prepare 3-5 impressive demo queries
- [ ] Test demo queries on real StackRox instance
- [ ] Record demo video or prepare live demo environment
- [ ] Create demo talking points (speed, accuracy, ease of use)

### Documentation
- [ ] Update project README with AI search feature usage
- [ ] Document environment variable setup for other developers
- [ ] Add inline code comments for complex logic
- [ ] Create quick start guide for enabling feature flag

---

## Critical Path (Minimum Viable Demo)

For fastest path to a working demo, prioritize these tasks in order:

1. **Setup Ollama** (Phase 1: 3 tasks)
   - Install Ollama, pull model, test connection

2. **Basic Types** (Phase 1: 2 tasks)
   - Create type definition files

3. **Ollama Provider Only** (Phase 2: 5 tasks)
   - Build just Ollama provider, skip others for now

4. **Core Parser** (Phase 3: 10 tasks)
   - Input sanitization, filter schema, prompt, parser service, basic tests

5. **Basic UI Component** (Phase 4: 6 tasks)
   - NaturalLanguageSearchInput with basic features, skip polish

6. **Integration** (Phase 5: 7 tasks)
   - Connect to WorkloadCvesOverviewPage, basic E2E tests

7. **Demo Prep** (Phase 6: 1 task)
   - Prepare demo queries and talking points

**Critical Path Total:** 34 tasks (achievable in 1 day)

---

## Notes

- **Ollama First:** Start with Ollama to avoid API costs and get fast iteration
- **Feature Flag:** Keep feature behind flag for easy rollback
- **Incremental Testing:** Test each phase before moving to next
- **Prompt Iteration:** Expect to refine prompt multiple times based on test results
- **Fallback Always:** Ensure traditional filters always work as fallback

---

## Completed Features

_This section will be updated as features are completed._

---

## Known Issues / Future Work

_To be filled in during development._
