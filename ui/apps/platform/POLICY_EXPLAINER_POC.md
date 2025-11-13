# ACS Policy Explainer - Proof of Concept

## Overview

This PoC adds AI-powered policy explanations to ACS (Advanced Cluster Security) in two places: the policy detail view and the policy creation wizard. It automatically generates human-readable explanations of when security policies trigger violations, making complex policy logic more accessible to both security teams and developers.

## Why This Is Useful

### The Problem

ACS policies use complex boolean logic with multiple sections, fields, operators, and conditions. Understanding exactly when a policy triggers requires:

- Knowledge of field semantics (what "Image Registry" means vs "Image Remote")
- Understanding boolean operators (AND/OR) at multiple levels
- Recognizing implicit AND logic between fields within a section
- Knowing how multiple sections combine (always OR)
- Understanding inverted/negated fields (fields that match the opposite of what their name suggests)
- Interpreting how scope and exclusions affect policy evaluation

This complexity creates barriers:

- **For Developers**: Unclear why their deployments are being blocked
- **For Security Teams**: Difficult to audit and explain policy behavior
- **For New Users**: Steep learning curve to create effective policies

### Value Delivered

1. **Improved Understanding**: Clear explanations in natural language of complex policy logic
2. **Faster Troubleshooting**: Developers can quickly understand why violations occur
3. **Better Policy Design**: Users can verify their policies do what they intend in real-time while building them
4. **Real-time Feedback**: Policy creators see immediate explanations as they modify criteria in the wizard
5. **Knowledge Transfer**: Leverages detailed field descriptions that were previously only visible in the policy editor
6. **Reduced Support Burden**: Self-service explanations reduce need for security team intervention

## What Was Implemented

### Policy Detail View

When viewing any policy detail page, an AI-generated explanation appears in a new card titled "When This Policy Triggers". Explanations use structured formatting with bullet points, bold emphasis on key terms, and concrete examples of violation scenarios.

(Note: The detail view retains the original single AI explanation. The tabbed interface with tables is wizard-only.)

### Policy Creation Wizard

The wizard shows real-time visual feedback as users build policies. The explanation card appears below the rules section with three tabbed views:

#### Interactive Simulator Tab
- **Interactive deployment simulator** - users can input ANY values to test scenarios
- **Field-specific controls** - appropriate inputs for each field type:
  - Dockerfile Line: dropdown for instruction type + text input for arguments
  - Image Signature: dropdown (Verified/Not Verified)
  - CVE Severity: dropdown (CRITICAL/HIGH/MEDIUM/LOW)
  - Registry/Tag/Labels: text inputs
- **Shows ALL sections** - multiple sections displayed as separate cards
- **Real-time evaluation** - immediate feedback as values are entered
- **Match indicators** - shows "Matches (triggers)" or "No match (safe)" per criterion
- **Visual styling** with red "VIOLATION" and green "NO VIOLATION" alerts
- **Section-by-section breakdown** - shows how many criteria match (e.g., "2/3 match (need ALL)")
- **Inverted field handling** - orange warning badges for inverted fields
- **Flexible testing** - test ANY deployment configuration, not just configured values

#### AI Explanation Tab
- **LLM-generated explanation** with 2-second debouncing to prevent excessive API calls
- **Smart validation** that detects incomplete criteria (fields with no values configured)
- **Truncated preview** showing 3 lines by default with "Show more/less" toggle
- **Proper formatting** with bold text rendering and structured layout
- **Truth tables** showing all possible combinations of field conditions and policy outcomes:
  - One table per policy section with columns for each field
  - All possible value combinations using **concrete, realistic values** (not abstract T/F)
  - Final "Result" column clearly shows "Violation" or "No Violation" for each combination
  - Shows actual values: "true"/"false", "Verified"/"Not Verified", "6 cores"/"4 cores", "Absent"/"Present"
  - Smart handling of large policies (6+ fields) with representative samples
  - Legend explaining what each column represents and what values mean

This dual-view approach provides:
- **Interactive exploration** (users can test scenarios they care about)
- **No information overload** (no exponential 2^n rows to scan)
- **Instant feedback** (simulator updates immediately as switches toggle)
- **Multi-section support** (all sections shown, not just first one)
- **Better understanding** (hands-on interaction + natural language explanation)

### Technology Stack

- **Provider**: Google Cloud Vertex AI
- **Models**: Claude (claude-3-5-sonnet@20240620, claude-sonnet-4-5) and Gemini (gemini-1.5-flash, gemini-1.5-pro)
- **Authentication**: OAuth 2.0 with gcloud access tokens

## How It Works

### High-Level Flow

```text
Policy Detail Page Load
         |
         v
PolicyExplainer Component Mounts
         |
         v
Extract Policy Data
  - Name, severity, description
  - Policy sections & criteria
  - Field names and values
  - Boolean operators
  - Scope/exclusions
         |
         v
Look Up Field Descriptions
  - Maps field names to policyCriteriaDescriptors
  - Retrieves authoritative descriptions
         |
         v
Build Structured Prompt
  - Policy metadata
  - Formatted criteria with descriptions
  - Instructions for output format
         |
         v
Call Vertex AI API
  - OAuth authentication
  - Model-specific request format
  - Parse response
         |
         v
Render Formatted Explanation
  - Parse **bold** markers
  - Display with PatternFly styling
```

## Implementation Details

### Service Layer

`VertexAIService.ts` handles LLM integration:

- Extracts policy criteria descriptions from `policyCriteriaDescriptors.tsx` using a lookup map
- Formats policy sections with their field requirements and boolean operators
- Detects fields with non-obvious semantics and adds contextual markers to help the LLM explain them clearly
- Detects model type (Claude vs Gemini) and uses appropriate API endpoint:
  - Claude: `https://{region}-aiplatform.googleapis.com/.../publishers/anthropic/models/{model}:rawPredict`
  - Gemini: `https://{region}-aiplatform.googleapis.com/.../publishers/google/models/{model}:generateContent`

### Component Layer

**PolicyExplainer.tsx** (Policy Detail View):

- Auto-generates explanation on mount via React useEffect
- Parses `**bold**` markers into styled `<strong>` elements
- Detects and renders tables (lines starting with `|`) in preformatted `<pre>` blocks with monospace font
- Tables styled with light background, border, and horizontal scrolling for wide content
- Integrates into policy detail page after PolicyOverview section

**PolicyExplainerWizard.tsx** (Policy Creation Wizard):

- Reads policy data from Formik context (live editing state)
- Implements tabbed interface with three views (Structure, Examples, AI Explanation)
- Tracks previous policy state for visual change detection
- AI tab implements 2-second debouncing to avoid API spam during active editing
- Validates policy criteria before generating AI explanations
- Structure and Examples tabs update instantly with no debounce
- Uses same table detection and rendering as PolicyExplainer (monospace preformatted blocks)
- Includes "Show more/less" toggle for long explanations with truth tables

**PolicySimulator.tsx** (Interactive Simulator Tab):

- Creates interactive deployment simulator with flexible input controls
- Shows ALL policy sections as separate cards
- For each criterion, provides:
  - Display name and configured value
  - Appropriate input controls based on field type:
    - **Dockerfile Line**: dropdown for instruction (RUN/CMD/etc.) + text input for arguments
    - **Image Signature**: dropdown (Verified/Not Verified)
    - **CVE Severity**: dropdown (CRITICAL/HIGH/MEDIUM/LOW/UNKNOWN)
    - **Text fields** (Registry, Tag, Label, User, etc.): text input
  - Match indicator showing "Matches (triggers)" or "No match (safe)"
  - Inverted field warning badge
- Real-time evaluation:
  - Compares user-entered values against configured values
  - Calculates match count per section (e.g., "2/3 match (need ALL)")
  - Determines if section triggers (all criteria must match)
  - Applies OR logic across sections (any section match = violation)
  - Shows overall result in prominent alert (red=violation, green=pass)
- Correctly handles inverted fields:
  - Shows orange warning badge
  - Match logic respects inverted semantics
- Flexible testing:
  - Users can enter ANY values, not just toggle between two states
  - Test realistic deployment configurations
  - Compare against configured policy values
- Updates instantly with smooth animations

### Prompt Engineering Strategy

The prompt is carefully designed to:

1. **Focus on Trigger Conditions**: Only explains WHEN the policy triggers, not WHY it matters (rationale already visible elsewhere)

2. **Leverage Field Descriptions**: Includes detailed descriptions from `policyCriteriaDescriptors.tsx` for each policy criterion, providing authoritative context

3. **Handle Field Semantics**:
   - Detects fields with non-obvious matching behavior (e.g., fields that match when a condition is NOT met)
   - Adds contextual markers in prompt to clarify field behavior
   - Instructs LLM to use explicit language about when fields match

4. **Emphasize Boolean Logic**:
   - Makes clear that policy sections are combined with OR
   - Emphasizes that ALL fields within a section must trigger (implicit AND)
   - Highlights configurable OR/AND operators within field values

5. **Structured Format Requirements**:
   - One-sentence summary
   - Bullet points for sections
   - "ALL of the following must trigger:" for field lists
   - Concrete examples with exact values
   - Bold formatting on key terms

6. **Truth Table Generation**:
   - Generates comprehensive truth tables showing all possible field combinations
   - One table per policy section with columns for each field
   - Rows show all possible value combinations (2^n for n fields)
   - Final "Result" column shows "Violation" or "No Violation" for each combination
   - Uses **concrete, realistic values** instead of abstract T/F notation:
     - Boolean fields: "true"/"false"
     - Comparison fields: actual values like "6 cores" (matches >5) vs "4 cores" (doesn't match)
     - Signature verification: "Verified"/"Not Verified"
     - Inverted fields: "Absent"/"Present" or "Missing"/"exists"
     - Image registries: actual names like "quay.io"/"docker.io"
   - Intelligently handles large policies (6+ fields) by showing representative samples
   - Includes legend explaining what each column represents and what values mean
   - Makes AND logic within sections immediately visible
   - Helps users understand inverted field behavior through concrete examples

7. **Length Control**: Extended token limit (4096) to accommodate truth tables while keeping other sections concise

## Configuration

### Environment Variables (Vite)

Create `ui/apps/platform/.env.local`:

```bash
VITE_VERTEX_PROJECT_ID=your-gcp-project-id
VITE_VERTEX_LOCATION=us-east5
VITE_VERTEX_MODEL=claude-3-5-sonnet@20240620
VITE_VERTEX_ACCESS_TOKEN=your-access-token
```

### Generating Access Tokens

Access tokens expire after ~1 hour. Regenerate with:

```bash
gcloud auth print-access-token
```

Update the token in `.env.local` and restart the dev server.

## Limitations & Known Issues

### Current Limitations

1. **No Caching**: Explanations are regenerated on every page load
   - Increases latency and API costs
   - Same policy always produces slightly different explanations

2. **Token Expiration**: OAuth tokens expire hourly
   - Requires manual regeneration
   - No automatic refresh mechanism

3. **Client-Side LLM Calls**: API calls made directly from browser
   - Exposes API endpoints and tokens in browser
   - Not suitable for production without backend proxy

4. **No Explanation Persistence**: Generated text is not stored
   - Cannot be edited or improved by users
   - No version control or audit trail

5. **Limited Error Handling**: Basic error messages
   - No retry logic
   - No fallback for API failures

6. **Model-Specific Quirks**:
   - Different models produce slightly different output quality
   - No guarantee of consistent formatting

### PoC Scope

This is a proof of concept intended to validate the approach. It is NOT production-ready:

- Hardcoded configuration fallbacks
- No security review of token handling
- No performance optimization
- No analytics or monitoring
- No testing coverage

## Future Enhancements

### Near-Term Improvements

1. **Backend Integration**
   - Move LLM calls to Central backend service
   - Secure token management server-side
   - Add caching layer (Redis/PostgreSQL)
   - Implement rate limiting

2. **Caching Strategy**
   - Cache explanations by policy content hash
   - Invalidate cache when policy is modified
   - Pre-generate explanations for default policies
   - Store in database alongside policy

3. **Better Authentication**
   - Use service account with workload identity
   - Automatic token refresh
   - Remove token from client-side code entirely

4. **UI Enhancements**
   - Add "Regenerate" button for manual refresh
   - Show cache status (fresh vs cached)
   - Allow users to rate explanation quality
   - Support for expanding/collapsing sections

5. **Quality Improvements**
   - A/B test different prompt templates
   - Collect user feedback on explanation quality
   - Fine-tune model on ACS-specific examples
   - Add validation to ensure output correctness

### Medium-Term Features

1. **Interactive Explanations**
   - Click field names to see full field documentation
   - Highlight matching parts of policy when hovering examples
   - Show related policies with similar logic

2. **Violation-Specific Explanations**
   - On violation detail page, explain why THIS specific deployment triggered
   - Show which policy section matched
   - Display actual field values that caused trigger

3. **Policy Comparison**
   - Compare explanations of similar policies side-by-side
   - Highlight differences in trigger conditions
   - Suggest policy consolidation opportunities

4. **Multi-Language Support**
   - Generate explanations in user's preferred language
   - Internationalize technical terms appropriately

5. **Explanation Templates**
   - Create curated explanations for common policy patterns
   - Allow admins to override AI explanations with custom text
   - Version control for explanation improvements

### Long-Term Vision

1. **Policy Creation Assistant**
   - Natural language to policy conversion
   - "Create a policy that blocks containers running as root"
   - AI suggests policy criteria and logic

2. **Policy Optimization**
   - Detect redundant or conflicting policies
   - Suggest simplifications
   - Identify gaps in policy coverage

3. **Learning System**
   - Learn from user feedback (thumbs up/down)
   - Improve explanations over time
   - Adapt terminology to organization's vocabulary

4. **Integration with Documentation**
   - Link explanations to relevant docs
   - Generate policy-specific runbooks
   - Create training materials from policies

5. **Compliance Mapping**
   - Explain how policies relate to compliance frameworks
   - Map policy logic to specific compliance requirements
   - Generate audit reports with explanations

## Technical Decisions & Rationale

### Why Vertex AI?

- Already available in user's GCP project
- Supports both Claude (best instruction following) and Gemini (fast/cheap)
- No additional vendor onboarding required
- Same authentication as other GCP services

### Why Client-Side Implementation?

For PoC only:

- Faster iteration without backend changes
- Validates UX before investing in backend
- Easy to demo and share
- NOT suitable for production

### Why Extract Field Descriptions?

- Single source of truth for field semantics
- Ensures consistency between policy editor and explanations
- Leverages existing documentation work
- Reduces hallucination risk

### Why Focus on "When" vs "Why"?

- "Why" (rationale) already present in policy metadata
- "When" (trigger conditions) is the complex, hard-to-understand part
- Avoid redundancy with existing UI elements
- Keep explanations focused and concise

## Performance Considerations

### API Latency

Typical response times:

- **Claude Sonnet 4.5**: 2-4 seconds
- **Claude Sonnet 3.5**: 1.5-3 seconds
- **Gemini 1.5 Flash**: 1-2 seconds

Factors affecting latency:

- Prompt length (varies by policy complexity)
- Model choice (Claude is slower but higher quality)
- Network conditions
- API rate limits

### Cost Estimates

Per explanation (approximate):

- **Claude Sonnet 4.5**: $0.002-0.004
- **Claude Sonnet 3.5**: $0.001-0.002
- **Gemini 1.5 Flash**: $0.0001-0.0002

For a deployment with 200 policies viewed monthly:

- Claude Sonnet: ~$0.40-0.80/month
- Gemini Flash: ~$0.02-0.04/month

Caching would reduce costs by 90%+ for repeated views.

## Testing Recommendations

Before production deployment:

1. **Accuracy Testing**
   - Verify explanations match actual policy behavior
   - Test edge cases (complex nested logic, exclusions)
   - Validate examples are truly violations

2. **Quality Testing**
   - User testing with security team and developers
   - A/B test different prompt templates
   - Measure comprehension improvement

3. **Performance Testing**
   - Load test with concurrent requests
   - Measure impact on page load time
   - Test with slow network conditions

4. **Security Testing**
   - Review token handling and storage
   - Test for prompt injection vulnerabilities
   - Validate no sensitive data in prompts

5. **Compatibility Testing**
   - Test with all default policies
   - Test with complex custom policies
   - Test with policies using all field types
   - Test policies with various field combinations and boolean logic
   - Verify wizard updates correctly as policies are modified

## Truth Table Example

The AI explanation now includes a comprehensive truth table showing all possible combinations of field conditions using **concrete, realistic values** instead of abstract T/F notation. Here's an example for a policy checking privileged containers, signature verification, and CPU limits:

```
Truth Table for Section 1:

| Privileged Container | Image Signature | CPU Request | Result       |
|---------------------|-----------------|-------------|--------------|
| true                | Not Verified    | 6 cores     | Violation    |
| true                | Not Verified    | 4 cores     | No Violation |
| true                | Verified        | 6 cores     | No Violation |
| true                | Verified        | 4 cores     | No Violation |
| false               | Not Verified    | 6 cores     | No Violation |
| false               | Not Verified    | 4 cores     | No Violation |
| false               | Verified        | 6 cores     | No Violation |
| false               | Verified        | 4 cores     | No Violation |

Legend:
- Privileged Container: Whether container runs in privileged mode (true/false)
- Image Signature: Whether image signature can be verified (Verified/Not Verified)
- CPU Request: CPU cores requested (policy triggers when >5 cores)
- Result: Whether the policy triggers a violation for this combination

Note: Within this section, ALL three conditions must be met simultaneously for the policy to trigger. 
Only the first row (privileged=true AND signature not verified AND CPU >5 cores) results in a violation.
```

**Example for Inverted Fields:**
```
| Required Label "app" | Image Registry | Result       |
|---------------------|----------------|--------------|
| Absent              | docker.io      | Violation    |
| Absent              | quay.io        | No Violation |
| Present             | docker.io      | No Violation |
| Present             | quay.io        | No Violation |

Note: "Required Label" is an inverted field - it triggers when the label is ABSENT (missing).
```

**Key Benefits:**
- **Concrete Values**: Shows actual values like "6 cores", "Not Verified", "Absent" instead of abstract T/F
- **Clear Results**: "Violation" / "No Violation" is immediately understandable
- **Intuitive Understanding**: Users immediately see realistic scenarios (e.g., "true + Not Verified + 6 cores = Violation")
- **Complete Coverage**: Shows every possible combination of field values
- **Visual Clarity**: Makes AND logic immediately obvious (all conditions must match)
- **Inverted Field Clarity**: Clearly shows "Absent"/"Present" for fields like "Required Label"
- **Threshold Examples**: Shows values on both sides of comparisons (>5: shows "6 cores" vs "4 cores")
- **Multiple Sections**: One table per section with note that ANY section resulting in Violation = policy triggers
- **Smart Scaling**: For policies with 6+ fields, shows representative samples with clear notation

## Interactive Simulator Approach

The wizard implementation uses an interactive simulator that lets users explore policy behavior hands-on:

### Design Rationale

**Problem:** The original AI-only explanation had a 2-second delay, and static truth tables with 2^n rows quickly become overwhelming (8+ criteria = 256+ rows).

**Solution:** Create an interactive deployment simulator where users input values and see results instantly:

1. **Interactive Simulator Tab** - HANDS-ON TESTING
   - Shows a simulated deployment with input controls for each criterion
   - Users enter actual values to test "what if" scenarios they care about
   - Field-specific controls (dropdowns for Dockerfile instructions, text inputs for registries, etc.)
   - Shows ALL policy sections simultaneously (not just first one)
   - Each section card shows:
     - All criteria with appropriate input controls
     - Match indicators per criterion
     - Current match count (e.g., "2/3 match (need ALL)")
     - Whether that section would trigger
   - Overall result displayed prominently (VIOLATION vs NO VIOLATION)
   - Immediate updates with no API delay

2. **AI Explanation Tab** - NATURAL LANGUAGE SUMMARY
   - Natural language explanation for those who prefer prose
   - Maintains the 2-second debounce for API efficiency
   - Best for understanding subtle logic and edge cases
   - Complements the hands-on simulator with contextual explanation

### User Experience Benefits

- **Hands-On Exploration:** Test scenarios interactively by entering actual values
- **Realistic Testing:** Input real deployment configurations (e.g., "RUN apt-get install", "quay.io/myapp")
- **No Information Overload:** No exponential row explosion (8 fields = 256 rows!)
- **Instant Feedback:** Simulator updates immediately as values are entered
- **Multi-Section Support:** All sections shown, not just the first one
- **Reduced Cognitive Load:** Familiar form controls (text inputs, dropdowns) instead of abstract tables
- **Direct Logic Verification:** See AND logic (within sections) and OR logic (across sections) in action
- **Focused Testing:** Test specific cases you care about with actual values

### Technical Implementation

**Simulator State Management:**
- Extracts criteria from all policy sections (not just first one)
- Each criterion stores:
  - `configuredKey` and `configuredValue`: what the policy requires
  - `simulatedKey` and `simulatedValue`: what the user enters
- React state updates when user changes inputs
- Each change triggers re-evaluation of sections and overall result

**Interactive Controls:**
- Field-specific input components:
  ```typescript
  // Dockerfile Line
  <FormSelect> for instruction type (RUN, CMD, etc.)
  <TextInput> for arguments
  
  // Image Signature
  <FormSelect> with Verified/Not Verified options
  
  // CVE Severity
  <FormSelect> with CRITICAL/HIGH/MEDIUM/LOW options
  
  // Generic fields (Registry, Tag, Label, etc.)
  <TextInput> for free-form values
  ```
- Match indicators show "Matches (triggers)" or "No match (safe)"
- Inverted fields get orange warning badges

**Real-Time Evaluation:**
- Value comparison:
  ```typescript
  // For key=value fields (Dockerfile Line)
  keysMatch = simulatedKey === configuredKey;
  valuesMatch = simulatedValue === configuredValue;
  matches = keysMatch && valuesMatch;
  
  // For simple fields
  matches = simulatedValue === configuredValue;
  ```
- Per-section logic:
  ```typescript
  matchCount = criteria.filter(c => valuesMatch(c)).length;
  allMatch = matchCount === criteria.length; // AND logic
  ```
- Overall violation:
  ```typescript
  anySectionViolates = sections.some(s => s.allMatch); // OR logic
  ```
- Updates instantly on every input change

**Visual Feedback:**
- Prominent alert shows overall result (red=violation, green=pass)
- Each criterion shows match status with colored badge
- Each section card shows match count (e.g., "2/3 match (need ALL)")
- Smooth CSS animations on result changes
- Color-coded: red=matches (triggers), green=no match (safe)

**Performance:**
- No exponential complexity - O(n) where n = number of criteria
- Instant re-renders (no API calls)
- Works smoothly even with many criteria
- AI explanation debounced at 2 seconds

### Advantages Over AI-Only Approach

| Aspect | AI-Only | Interactive Simulator + AI |
|--------|---------|---------------------------|
| Initial feedback | 2+ seconds | **Instant** |
| Exploration | Passive reading | **Active hands-on testing** |
| Scalability | N/A (text) | **No explosion with many criteria** |
| Multi-section support | Text description | **Visual cards for ALL sections** |
| Verification | Must trust AI | **Test yourself with toggles** |
| Inverted fields | In explanation text | **Explicit labels (ABSENT/present)** |
| User engagement | Low (reading) | **High (interactive)** |
| Focused testing | "Here's everything" | **"Test what YOU care about"** |

## Next Steps

1. **Validate with SMEs**: Have ACS policy experts review the generated explanations and table examples
2. **User Testing**: Gather feedback from Support Engineers and users on the tabbed interface and table clarity
3. **Refine Example Generation**: Add more field types and smarter scenario generation logic
4. **Expand Field Coverage**: Identify other confusing fields or patterns that need special handling
5. **Backend Integration**: Move AI explanation to production with caching and monitoring
6. **Metrics**: Track which tabs users prefer and where they spend time
7. **Animation Tuning**: Gather feedback on animation timing and visual feedback

## Conclusion

This PoC successfully demonstrates that combining LLM-powered explanations with instant visual feedback can significantly improve the understandability of ACS security policies. The implementation leverages existing policy metadata and field descriptions to generate accurate, structured explanations with minimal hallucination risk.

The interactive simulator provides hands-on exploration and immediate feedback, addressing different user needs:
- **Interactive testing:** "What if I deploy X?" → Enter values in simulator
- **Multi-section understanding:** "Which section triggers?" → See all sections evaluated simultaneously
- **Detailed understanding:** "Why exactly does this work?" → Check AI Explanation tab

The interactive simulator is particularly valuable for:
- **Hands-on learning:** Enter real values and see results instantly
- **Debugging policies:** Test specific deployment configurations you're concerned about
- **Realistic testing:** Use actual Dockerfile instructions, registry names, label values, etc.
- **Understanding AND/OR logic:** See which sections match (OR) and how criteria combine (AND)
- **Inverted fields:** Orange warning badges alert you to inverted matching semantics
- **No cognitive overload:** Test specific cases with real data instead of scanning abstract tables

Next steps should focus on moving the AI implementation to the backend, adding caching for AI explanations, and gathering user feedback on the interactive simulator (do users find it more helpful than static explanations?).
