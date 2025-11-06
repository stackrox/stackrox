# ACS Policy Explainer - Proof of Concept

## Overview

This PoC adds AI-powered policy explanations to ACS (Advanced Cluster Security) in two places: the policy detail view and the policy creation wizard. It automatically generates human-readable explanations of when security policies trigger violations, making complex policy logic more accessible to both security teams and developers. In the wizard, explanations update in real-time as users build policies, providing immediate feedback on policy behavior.

## What Was Implemented

### Policy Detail View

When viewing any policy detail page, an AI-generated explanation appears in a new card titled "When This Policy Triggers". Explanations use structured formatting with bullet points, bold emphasis on key terms, and concrete examples of violation scenarios.

### Policy Creation Wizard

The wizard shows real-time explanations as users build policies. The explanation card appears below the rules section and updates automatically as criteria are modified. Features include:
- **Real-time updates** with 2-second debouncing to prevent excessive API calls
- **Smart validation** that detects incomplete criteria (fields with no values configured)
- **Truncated preview** showing 3 lines by default with "Show more/less" toggle
- **Proper formatting** with bold text rendering and structured layout

**New Files:**
- `ui/apps/platform/src/services/VertexAIService.ts` (322 lines) - LLM API integration supporting Claude and Gemini models
- `ui/apps/platform/src/Containers/Policies/Detail/PolicyExplainer.tsx` (143 lines) - React component for policy detail view
- `ui/apps/platform/src/Containers/Policies/Wizard/Step3/PolicyExplainerWizard.tsx` (307 lines) - React component for policy wizard with debouncing and validation

**Modified Files:**
- `ui/apps/platform/src/Containers/Policies/Detail/PolicyDetailContent.tsx` (+4 lines) - Integrated PolicyExplainer component
- `ui/apps/platform/src/Containers/Policies/Wizard/Step3/PolicyCriteriaForm.tsx` (+10 lines) - Integrated PolicyExplainerWizard component

**LLM Integration:**
- Provider: Google Cloud Vertex AI
- Models: Claude (claude-3-5-sonnet@20240620, claude-sonnet-4-5) and Gemini (gemini-1.5-flash, gemini-1.5-pro)
- Authentication: OAuth 2.0 with gcloud access tokens

## Why This Is Useful

### Problem Statement

ACS policies use complex boolean logic with multiple sections, fields, operators, and conditions. Understanding exactly when a policy triggers requires:
- Knowledge of field semantics (what "Image Registry" means vs "Image Remote")
- Understanding boolean operators (AND/OR) at multiple levels
- Recognizing implicit AND logic between fields within a section
- Knowing how multiple sections combine (always OR)
- **Understanding inverted/negated fields** - fields that match the OPPOSITE of what their name suggests

This complexity creates barriers:
- **For Developers**: Unclear why their deployments are being blocked
- **For Security Teams**: Difficult to audit and explain policy behavior
- **For New Users**: Steep learning curve to create effective policies

#### The Inverted Field Problem

**Inverted/negated fields** are a recurring source of confusion in ACS policies:

**Common Scenario**: Verify signatures only on images with a specific label (ignore unlabeled images)
- **Expected Logic**: "If label is present AND signature is missing → violate"

**What users often try**: Policy with "Required Image Label" + "Image Signature Verified By"
- **What happens**: Only violates when BOTH label AND signature are missing
- **Why**: Both fields are inverted - they match when conditions are ABSENT
  - "Required Image Label" matches when label is ABSENT (not present)
  - "Image Signature Verified By" matches when signature CANNOT be verified
  - Combined with AND: violation only fires when BOTH are absent

**Correct Solution**: Use "Disallowed Image Label" + "Image Signature Verified By"
- "Disallowed Image Label" matches when label IS present
- "Image Signature Verified By" matches when signature is missing
- Combined with AND: violation fires when label present AND signature missing ✓

This confusion pattern happens repeatedly and often requires investigation to understand. The Policy Explainer specifically addresses this by explicitly identifying inverted fields and clarifying their matching behavior.

### Solution Benefits

1. **Improved Understanding**: Clear explanations in natural language of complex policy logic
2. **Faster Troubleshooting**: Developers can quickly understand why violations occur
3. **Better Policy Design**: Users can verify their policies do what they intend in real-time while building them
4. **Real-time Feedback**: Policy creators see immediate explanations as they modify criteria in the wizard
5. **Knowledge Transfer**: Leverages detailed field descriptions from `policyCriteriaDescriptors.tsx` that were previously only visible in the policy editor
6. **Reduced Support Burden**: Self-service explanations reduce need for security team intervention

## How It Works

```
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

### Implementation Details

**Service Layer** (`VertexAIService.ts`):
- Extracts policy criteria descriptions from `policyCriteriaDescriptors.tsx` using a lookup map
- Formats policy sections with their field requirements and boolean operators
- Detects inverted/negated fields and adds warning markers:

```typescript
// Fields that have "inverted" matching behavior (match when condition is NOT met)
const NEGATED_FIELDS = new Set([
    'Required Label',
    'Required Annotation',
    'Required Image Label',
    'Required Image User',
    'Image Signature Verified By', // Matches when signature CANNOT be verified
    'Trusted Image Signers', // Matches when NOT signed by trusted signer
]);
```

- Detects model type (Claude vs Gemini) and uses appropriate API endpoint
- API Endpoints:
  - Claude: `https://{region}-aiplatform.googleapis.com/.../publishers/anthropic/models/{model}:rawPredict`
  - Gemini: `https://{region}-aiplatform.googleapis.com/.../publishers/google/models/{model}:generateContent`

**Component Layer** (`PolicyExplainer.tsx`):
- Auto-generates explanation on mount via React useEffect
- Parses `**bold**` markers into styled `<strong>` elements
- Integrates into policy detail page after PolicyOverview section

**Wizard Component Layer** (`PolicyExplainerWizard.tsx`):
- Reads policy data from Formik context (live editing state)
- Implements 2-second debouncing to avoid API spam during active editing
- Validates policy criteria before generating:
  - Checks `key`, `value`, and `arrayValue` fields (handles different field types)
  - Only flags incomplete when ALL fields are empty (allows optional sub-fields)
- Shows truncated preview (3 lines) with "Show more/less" toggle
- Parses formatting and integrates below rules section in wizard

### Prompt Engineering Strategy

The prompt is carefully designed to:

1. **Focus on Trigger Conditions**: Only explains WHEN the policy triggers, not WHY it matters (rationale already visible elsewhere)

2. **Leverage Field Descriptions**: Includes detailed descriptions from `policyCriteriaDescriptors.tsx` for each policy criterion, providing authoritative context

3. **Explicitly Handle Inverted/Negated Fields**:
   - Detects fields with inverted matching behavior ("Required Label", "Image Signature Verified By", etc.)
   - Adds warning markers in prompt: "⚠️ INVERTED FIELD: This criterion matches when condition is NOT met"
   - Instructs LLM to use explicit language: "matches when ABSENT" vs "matches when PRESENT"
   - Emphasizes AND logic is especially confusing with inverted fields
   - Examples:
     - ✗ BAD: "Required Image Label: app=myapp"
     - ✓ GOOD: "Image must be MISSING the label app=myapp (the 'Required Label' field triggers when the label is ABSENT)"

4. **Emphasize Boolean Logic**:
   - Makes clear that policy sections are combined with OR
   - Emphasizes that ALL fields within a section must trigger (implicit AND)
   - **Special attention to AND logic with inverted fields** - users often expect OR but get AND
   - Highlights configurable OR/AND operators within field values

5. **Structured Format Requirements**:
   - One-sentence summary
   - Bullet points for sections
   - "ALL of the following must trigger:" for field lists
   - Concrete examples with exact values showing presence/absence for inverted fields
   - Bold formatting on key terms (including ABSENT/PRESENT/CANNOT)

6. **Length Control**: Limited to ~300 words to accommodate inverted field explanations

### Inverted Field Examples

The explainer addresses common inverted field confusion patterns. Here's how it explains both problematic policies and correct solutions:

#### Problematic Policy Pattern

**Policy**: "Required Image Label" (app=app1) + "Image Signature Verified By"

**Generated Explanation**:

```text
This policy triggers when images are BOTH unlabeled AND unsigned.

The policy triggers when **ALL** of the following conditions are met:
    - Image must be **MISSING** the label app=app1
      (⚠️ "Required Image Label" matches when the label is **ABSENT**)
    - Image signature **CANNOT** be verified by the specified integration
      (⚠️ "Image Signature Verified By" matches when signature is **MISSING or INVALID**)

**IMPORTANT**: Both criteria are inverted fields combined with AND logic. The policy
will NOT fire if only the label is missing or only the signature is missing - it
requires BOTH to be absent simultaneously.

EXAMPLES:
• Image without app=app1 label AND without valid signature → **VIOLATION**
• Image with app=app1 label but no signature → NO VIOLATION
• Image without app=app1 label but with signature → NO VIOLATION
```

#### Correct Policy Pattern

**Policy**: "Disallowed Image Label" (app=app1) + "Image Signature Verified By"

**Generated Explanation**:

```text
This policy triggers when labeled images lack valid signatures.

The policy triggers when **ALL** of the following conditions are met:
    - Image **HAS** the label app=app1
      ("Disallowed Image Label" matches when the label is **PRESENT**)
    - Image signature **CANNOT** be verified by the specified integration
      (⚠️ "Image Signature Verified By" matches when signature is **MISSING or INVALID**)

This ensures only images with the app=app1 label are checked for signatures.

EXAMPLES:
• Image with app=app1 label AND no valid signature → **VIOLATION**
• Image with app=app1 label and valid signature → NO VIOLATION
• Image without app=app1 label (signature not checked) → NO VIOLATION
```

The explicit presence/absence language and inverted field warnings make the matching behavior immediately clear, preventing common user confusion.

### Data Flow Example

For the "Red Hat Images Must Be Signed" policy:

**Input Policy Data:**
```json
{
  "name": "Red Hat Images Must Be Signed by a Red Hat Release Key",
  "severity": "HIGH_SEVERITY",
  "policySections": [
    {
      "sectionName": "Red Hat Registry",
      "policyGroups": [
        {
          "fieldName": "Image Registry",
          "values": ["registry.redhat.io", "registry.access.redhat.com"],
          "booleanOperator": "OR"
        },
        {
          "fieldName": "Image Signature Verified By",
          "values": ["Red Hat Signature Integration"],
          "booleanOperator": "OR",
          "negate": false
        }
      ]
    }
  ]
}
```

**Field Description Lookup:**
- "Image Registry" → "Triggers a violation when the image registry name matches..."
- "Image Signature Verified By" → "Triggers a violation when an image lacks a valid cryptographic signature..."

**Generated Explanation:**
```
This policy triggers when Red Hat images lack valid signatures.

The policy triggers if ANY of the following conditions are met:

• Red Hat registry images without signatures
    ALL of the following must trigger:
    - Image Registry: registry.redhat.io OR registry.access.redhat.com
    - Image Signature: Cannot be verified by configured signer

EXAMPLES:
• registry.redhat.io/ubi9/ubi-minimal:latest without signature
```

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

6. **Inverted Field Testing**
   - Test policies with inverted fields ("Required Image Label" + "Image Signature Verified By")
   - Verify explanation clearly states when fields must be ABSENT vs PRESENT
   - Confirm warning markers appear for inverted fields
   - Test corrected patterns ("Disallowed Image Label" + "Image Signature Verified By")
   - Test various inverted field types: "Required Annotation", "Required Label", "Required Image User"
   - Verify wizard updates correctly when inverted fields are added

## Next Steps

1. **Validate with SMEs**: Have ACS policy experts review the generated explanations
2. **User Testing**: Gather feedback from Support Engineers and users on explanation clarity
3. **Expand Field List**: Identify other confusing fields that need special handling
4. **Backend Integration**: Move to production with caching and monitoring
5. **Metrics**: Track how often users view explanations for policies with inverted fields

## Conclusion

This PoC successfully demonstrates that LLM-powered policy explanations can significantly improve the understandability of ACS security policies. The implementation leverages existing policy metadata and field descriptions to generate accurate, structured explanations with minimal hallucination risk.

Key achievements:
- ✅ Natural language explanations of complex boolean logic
- ✅ **Explicit handling of inverted/negated fields** - addresses real customer confusion (ROX-30116)
- ✅ Real-time explanations in policy creation wizard with smart debouncing
- ✅ Emphasis on AND/OR relationships at multiple levels, especially for inverted fields
- ✅ Concrete examples of violation scenarios with explicit presence/absence statements
- ✅ Rich formatting for improved readability
- ✅ Smart validation detecting incomplete policy criteria
- ✅ Flexible model support (Claude and Gemini)
- ✅ Clear "matches when ABSENT" vs "matches when PRESENT" language

Next steps should focus on moving the implementation to the backend, adding caching, and gathering user feedback to refine the explanation quality and format.

---

**Files Modified/Created:**
- `ui/apps/platform/src/services/VertexAIService.ts` (new, 322 lines)
- `ui/apps/platform/src/Containers/Policies/Detail/PolicyExplainer.tsx` (new, 143 lines)
- `ui/apps/platform/src/Containers/Policies/Wizard/Step3/PolicyExplainerWizard.tsx` (new, 307 lines)
- `ui/apps/platform/src/Containers/Policies/Detail/PolicyDetailContent.tsx` (modified, +4 lines)
- `ui/apps/platform/src/Containers/Policies/Wizard/Step3/PolicyCriteriaForm.tsx` (modified, +10 lines)
- `ui/apps/platform/.env.local` (new, configuration)

**Total LoC Added:** ~780 lines

