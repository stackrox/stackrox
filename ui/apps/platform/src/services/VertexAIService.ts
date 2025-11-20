import type {
    BasePolicy,
    ClientPolicy,
    Policy,
    PolicyGroup,
    PolicySection,
} from 'types/policy.proto';
import {
    auditLogDescriptor,
    policyCriteriaDescriptors,
} from 'Containers/Policies/Wizard/Step3/policyCriteriaDescriptors';

// Extended type for ValueObj that includes arrayValue (used at runtime but not in official types)
type ExtendedValueObj = {
    value?: string;
    arrayValue?: string[];
};

// Configuration - For PoC, these can be set here or via environment variables
// Note: Using import.meta.env for Vite (not process.env)
const VERTEX_AI_CONFIG = {
    projectId: import.meta.env.VITE_VERTEX_PROJECT_ID || 'itpc-gcp-hybrid-pe-eng-claude',
    location: import.meta.env.VITE_VERTEX_LOCATION || 'us-east5',
    // Can use either Gemini (gemini-1.5-flash) or Claude (claude-3-5-sonnet@20240620)
    model: import.meta.env.VITE_VERTEX_MODEL || 'claude-3-5-sonnet@20240620',
    // For Vertex AI, use OAuth access token instead of API key
    accessToken: import.meta.env.VITE_VERTEX_ACCESS_TOKEN || '',
};

// Debug logging to verify configuration is loaded (remove after testing)
// Commented out to avoid console statement linting errors
// console.log('Vertex AI Config Debug:', {
//     projectId: VERTEX_AI_CONFIG.projectId,
//     location: VERTEX_AI_CONFIG.location,
//     model: VERTEX_AI_CONFIG.model,
//     hasAccessToken: !!VERTEX_AI_CONFIG.accessToken,
//     accessTokenLength: VERTEX_AI_CONFIG.accessToken?.length || 0,
//     envModel: import.meta.env.VITE_VERTEX_MODEL,
//     allEnvVars: Object.keys(import.meta.env).filter((k) => k.startsWith('VITE_')),
// });

// Combine all descriptors for lookup
const allDescriptors = [...policyCriteriaDescriptors, ...auditLogDescriptor];

// Create a map for quick lookup by field name
const descriptorMap = new Map(allDescriptors.map((d) => [d.name, d]));

type GeminiResponse = {
    candidates: {
        content: {
            parts: {
                text: string;
            }[];
        };
        finishReason?: string;
    }[];
};

type ClaudeResponse = {
    content: {
        text: string;
    }[];
    stop_reason: string;
};

export type PolicyExplanationResult = {
    explanation: string;
    warning?: string;
};

// Fields that have "inverted" matching behavior (match when condition is NOT met)
// These commonly confuse users as they trigger the opposite of what the name suggests
const NEGATED_FIELDS = new Set([
    'Required Label',
    'Required Annotation',
    'Required Image Label',
    'Required Image User',
    'Image Signature Verified By', // Matches when signature CANNOT be verified
    'Trusted Image Signers', // Matches when NOT signed by trusted signer
]);

/**
 * Determines if a field has inverted/negated matching behavior
 */
function isNegatedField(fieldName: string): boolean {
    return NEGATED_FIELDS.has(fieldName);
}

/**
 * Formats policy criteria from policy sections into human-readable text with descriptions
 */
function formatPolicyCriteria(policySections: PolicySection[] | undefined): string {
    if (!policySections || policySections.length === 0) {
        return 'No specific criteria defined';
    }

    const formattedSections = policySections.map((section) => {
        const sectionName = section.sectionName || 'Policy Section';
        const groups = section.policyGroups
            .map((group: PolicyGroup) => {
                const { fieldName } = group;
                const operator = group.booleanOperator;
                const negate = group.negate ? 'NOT ' : '';

                // Format values with explicit comparison operator explanation
                const formattedValues = group.values
                    .map((v) => {
                        // Handle arrayValue (e.g., image signature fields)
                        // TypeScript doesn't officially include arrayValue in ValueObj but it's used at runtime
                        const vExtended = v as ExtendedValueObj;
                        if (vExtended.arrayValue && Array.isArray(vExtended.arrayValue)) {
                            return vExtended.arrayValue.length > 0
                                ? `${vExtended.arrayValue.length} integration(s) selected`
                                : 'No integrations selected';
                        }

                        const val = v.value;
                        // Handle undefined/null values
                        if (!val) {
                            return '';
                        }
                        // Check if value starts with a comparison operator
                        if (typeof val === 'string' && val.match(/^(>|>=|<|<=|=)/)) {
                            return val; // Keep as-is, we'll add note below
                        }
                        return val;
                    })
                    .join(`, ${operator} `);

                const values = formattedValues;

                // Look up the descriptor for this field
                const descriptor = descriptorMap.get(fieldName);
                // Use shortName (what users see in UI) instead of name (internal field name)
                const displayName = descriptor?.shortName || fieldName;
                const description = descriptor?.description
                    ? `\n    Description: ${descriptor.description}`
                    : '';

                // Flag fields with inverted/negated matching behavior
                const negatedWarning = isNegatedField(fieldName)
                    ? '\n    ⚠️ INVERTED FIELD: This criterion matches when the condition is NOT met (inverted matching behavior)'
                    : '';

                // Add note about comparison operators if present
                const hasComparisonOperators = group.values.some((v) => {
                    // Skip arrayValue entries
                    const vExtended = v as ExtendedValueObj;
                    if (vExtended.arrayValue) {
                        return false;
                    }
                    // Check if value has comparison operator
                    return v.value && typeof v.value === 'string' && v.value.match(/^(>|>=|<|<=)/);
                });
                const comparisonNote = hasComparisonOperators
                    ? '\n    ⚠️ COMPARISON OPERATORS: The values contain comparison operators (>, >=, <, <=). For example, ">8" means "greater than 8", not "exactly 8".'
                    : '';

                return `  ${negate}${displayName}: ${values}${description}${negatedWarning}${comparisonNote}`;
            })
            .join('\n\n');
        return `${sectionName}:\n${groups}`;
    });

    return formattedSections.join('\n\n');
}

/**
 * Formats scope and exclusions into readable text
 */
function formatScopeInfo(policy: BasePolicy): string {
    const scopeParts: string[] = [];

    if (policy.scope && policy.scope.length > 0) {
        const scopeDesc = policy.scope
            .map((s) => {
                const parts: string[] = [];
                if (s.cluster) {
                    parts.push(`Cluster: ${s.cluster}`);
                }
                if (s.namespace) {
                    parts.push(`Namespace: ${s.namespace}`);
                }
                if (s.label) {
                    parts.push(`Label: ${s.label.key}=${s.label.value}`);
                }
                return parts.join(', ');
            })
            .join('; ');
        scopeParts.push(`Scope: ${scopeDesc}`);
    }

    if (policy.exclusions && policy.exclusions.length > 0) {
        const exclusionDesc = policy.exclusions
            .map((e) => {
                if (e.deployment) {
                    return `Deployment: ${e.deployment.name}`;
                }
                if (e.image) {
                    return `Image: ${e.image.name}`;
                }
                return '';
            })
            .filter(Boolean)
            .join('; ');
        if (exclusionDesc) {
            scopeParts.push(`Exclusions: ${exclusionDesc}`);
        }
    }

    return scopeParts.length > 0 ? scopeParts.join('\n') : 'Applies to all resources';
}

/**
 * Builds the prompt for the LLM based on policy data
 */
function buildPrompt(policy: BasePolicy | Policy | ClientPolicy): string {
    const policyCriteria = formatPolicyCriteria(
        (policy as Policy).policySections || (policy as ClientPolicy).serverPolicySections
    );
    const scopeInfo = formatScopeInfo(policy);

    return `You are a Kubernetes security expert. Generate a clear, human-readable explanation of this security policy for ACS (Advanced Cluster Security).

===
CRITICAL - UNDERSTAND HOW ACS POLICIES WORK:

In ACS, a policy defines the conditions when a violation should be raised. The conditions are defined using a set of criteria to be matched; when a match is found, a violation is generated.

Note: this differs from an alternative approach where policies would define the security posture, raising violations when deviations occur. That is not the case in ACS.

Instead, in ACS the policy criteria define conditions when a violation should be raised.

EXAMPLE TO PREVENT CONFUSION:
Policy with Field A (inverted field) + Field B (normal field)
- This TRIGGERS A VIOLATION when: Field A condition is NOT met AND Field B condition IS met
- This ENFORCES the requirement that: when Field B matches, Field A must also be satisfied
- How does it enforce this? By raising violations when Field A is NOT satisfied while Field B matches
- WRONG interpretation: "enforces that Field A must NOT be satisfied"
- CORRECT interpretation: "triggers when Field A is NOT satisfied while Field B matches, thereby REQUIRING Field A to be satisfied when Field B is present"

Always remember: TRIGGER CONDITION ≠ DESIRED STATE. Policies trigger on BAD states to enforce GOOD states.
===

Policy Metadata:
Policy Name: ${policy.name}
Severity: ${policy.severity}
Description: ${policy.description || 'No description provided'}
Rationale: ${policy.rationale || 'Not specified'}
Categories: ${policy.categories?.join(', ') || 'None'}
Lifecycle Stages: ${policy.lifecycleStages?.join(', ') || 'Not specified'}
Enforcement Actions: ${policy.enforcementActions?.join(', ') || 'Alert only'}

Policy Criteria (with detailed field descriptions):
${policyCriteria}

Scope/Exclusions: 
${scopeInfo}

IMPORTANT: The "Description" fields under each policy criterion provide the authoritative explanation of what that criterion does, when it triggers, and how it should be used. Use these descriptions as your primary source of truth.

CRITICAL - NEGATED FIELDS (NOT prefix): When a field name starts with "NOT ", it means the field is negated. A negated field triggers when it does NOT match the specified value(s).

EXAMPLES:
- "NOT Field X: value1" triggers when Field X is NOT "value1" (triggers on "value2", "value3", etc., but NOT on "value1")
- "Field X: value1" triggers when Field X IS "value1"
- "NOT Field Y: value2" triggers when Field Y is NOT "value2" (triggers on "value3", "value4", etc., but NOT on "value2")

When explaining fields with the "NOT " prefix, be crystal clear:
- WRONG: "NOT Field X: value1" → "triggers when Field X is value1"
- CORRECT: "NOT Field X: value1" → "triggers when Field X is NOT value1"
- WRONG: "NOT Field Y: value2" → "triggers when Field Y is value2"
- CORRECT: "NOT Field Y: value2" → "triggers when Field Y is NOT value2"

CRITICAL - COMPARISON OPERATORS: Some fields use comparison operators in their values (marked with "⚠️ COMPARISON OPERATORS"). When you see values like ">8" or "<5", these are NOT exact values - they are comparisons:
- ">8" means "greater than 8" (not "exactly 8")
- ">=8" means "greater than or equal to 8"
- "<5" means "less than 5" (not "exactly 5")
- "<=5" means "less than or equal to 5"
- "=5" means "exactly equal to 5"

When explaining these, use clear language like:
- WRONG: "Field value of 8 or 5"
- RIGHT: "Field value greater than 8 OR less than 5"
- WRONG: "exactly 8"
- RIGHT: "more than 8" (for ">8")

CRITICAL - INVERTED/NEGATED FIELDS: Some fields are marked with "⚠️ INVERTED FIELD" which means they INHERENTLY match when the specified condition is NOT met. This is DIFFERENT from the "NOT " prefix and is a common source of user confusion:
- Fields named "Required X" typically match when X is ABSENT (not present)
- Fields named "Disallowed X" typically match when X is PRESENT
- Fields related to verification typically match when verification CANNOT be completed (is missing or invalid)

When explaining inverted fields, be EXTREMELY CLEAR about the matching behavior. Use phrases like:
- "matches when X is ABSENT" (not just "requires X")
- "triggers when verification CANNOT be completed" (not just "verification field")
- "fires when X is MISSING" (not just "required X")

Generate a technical explanation focused ONLY on when this policy triggers violations. Use PLAIN TEXT with simple formatting markers for emphasis.

REMINDER: You are explaining TRIGGER CONDITIONS (when violations occur), not the enforcement goal. A policy that triggers when "X is NOT present" is ENFORCING that "X MUST be present". Keep this distinction clear in your explanation.

Start with a one-sentence summary of when the overall policy triggers (the BAD state that causes a violation).

[blank line]

Then explain the trigger conditions:
- If there is ONLY ONE policy section: state it directly as a bullet, emphasizing that ALL listed fields must trigger for the violation to occur
- If there are MULTIPLE policy sections: say "The policy triggers if ANY of the following conditions are met:" (sections are ALWAYS combined with OR), then emphasize for each section that ALL its fields must trigger

IMPORTANT: Make it very clear that within each section, ALL field criteria must trigger simultaneously (AND logic). This is especially critical for inverted fields - users often expect OR behavior but get AND. Use phrases like:
- "The policy triggers when **ALL** of the following are true:"
- "This violation fires only when **BOTH** of these conditions match:"
- "**ALL** these field requirements must trigger simultaneously:"

For inverted fields in particular, explicitly state the matching behavior:
- WRONG: "Field X: value Y"
- RIGHT: "X must be MISSING/ABSENT value Y (the inverted field triggers when the condition is NOT met)"

Format the conditions as:

• Section condition: [brief description]
    ALL of the following must trigger:
    - Field requirement A: [specific detail with explicit match behavior for inverted fields]
    - Field requirement B: [specific detail with explicit match behavior for inverted fields]

[blank line]

• Another main condition (if multiple exist)
    - Sub-requirement X: [specific detail]

[blank line]

EXAMPLES:
• [Specific example 1 with exact values and explicit presence/absence statements for inverted fields]
• [Specific example 2 with exact values and explicit presence/absence statements for inverted fields]

[blank line]

TRUTH TABLE:
After the examples, generate a truth table showing all possible combinations of the policy field conditions and the resulting policy outcome.

For each policy section, create a table with:
- One column for each field in the section (use short field names)
- One final column labeled "Result" showing the outcome for that combination
- Rows showing all possible value combinations for the fields
- Use ACTUAL CONCRETE VALUES in cells, NOT abstract T/F notation
- In the "Result" column, use "Violation" if policy triggers, "No Violation" if it does not

**CRITICAL - USE CONCRETE VALUES, NOT T/F:**
- For boolean fields: use "true"/"false" or "yes"/"no"
- For comparison fields: use actual example values like "6 units" (matches) vs "4 units" (doesn't match)
- For verification fields: use "Verified"/"Not Verified" or "Valid"/"Invalid"
- For text fields: use actual realistic values that make sense for that field type
- For inverted fields: use "Absent"/"Present" or "Missing"/"Has X"
- For dropdown/enum fields: use actual option values
- Keep values concise but realistic

Format the table using pipe characters (|) and dashes for borders. Example format:

**Truth Table for Section 1:**

| Field A    | Field B        | Field C     | Result       |
|------------|----------------|-------------|--------------|
| value1     | value2         | value3      | Violation    |
| value1     | value2         | value4      | No Violation |
| value1     | value5         | value3      | No Violation |
| value1     | value5         | value4      | No Violation |
| value6     | value2         | value3      | No Violation |
| value6     | value2         | value4      | No Violation |
| value6     | value5         | value3      | No Violation |
| value6     | value5         | value4      | No Violation |

IMPORTANT TRUTH TABLE GUIDELINES:
- Within a section, ALL fields must match their trigger conditions (AND logic) for the policy to trigger
- If there are multiple sections, show a table for EACH section
- After individual section tables, add a note: "Policy triggers if **ANY** section results in a Violation"
- For policies with 6+ fields in a section, you may show a representative sample of key combinations instead of all 2^n rows, but clearly note this
- Use realistic, concrete values that make sense for each field type
- For comparison operators, show values on both sides of the threshold
- For inverted fields, use clear terms like "Absent"/"Present" or "Missing"/"exists"

Formatting rules:
- Use bullet symbol (•) for main points, dash (-) for sub-items with 4 spaces of indentation
- Use **double asterisks** around key terms for emphasis (field names, important conditions, AND/OR/ALL/ABSENT/PRESENT/CANNOT keywords)
- Add blank lines between major sections
- For inverted fields, ALWAYS clarify whether you mean "present" or "absent"
- Use monospace-friendly table formatting with consistent column widths

Do NOT include these sections (they are redundant or shown elsewhere in the UI):
- Legend or explanation of what table columns mean
- Enforcement behavior or what the policy enforces
- Scope/exclusion information
- Enforcement action information

Do NOT explain why the policy matters or remediation steps. You may exceed 300 words if needed to show complete truth tables, but be concise in other sections.

---

CRITICAL - SANITY CHECK:

After generating the explanation, evaluate whether this policy is likely to be useful in practice or if it appears to be a user mistake.

**FIRST, try to come up with a realistic use case for this policy:**
- Can you think of a practical scenario where an organization would want this policy?
- Is there a legitimate security concern this policy addresses?
- Would a real security team or DevOps engineer actually deploy this?

REMINDER: The policy TRIGGERS on bad/unwanted states to ENFORCE good/desired states. Don't confuse the trigger condition with what the policy enforces.
- A policy that triggers when X is NOT present is ENFORCING that X MUST be present
- A policy that triggers when Y is enabled is ENFORCING that Y must NOT be enabled

If you CANNOT come up with a realistic use case, OR if the only use case you can imagine is highly contrived/artificial, this is a strong signal the policy may be a mistake.

**THEN, also consider these specific issues:**

IMPORTANT: Be conservative with warnings. Only flag issues when you are VERY CONFIDENT there is a problem. Many policy combinations that seem unusual may have legitimate use cases. When in doubt, DO NOT warn.

**Valid Use Cases to NOT Warn About:**
- **Conditional Requirements**: Combining inverted and normal fields to enforce conditional rules is VALID. Example: an inverted field (matches when X is NOT met) + a normal field (matches when Y IS met) creates a conditional rule meaning "when Y is present, X must also be satisfied" - this is a legitimate security pattern.
- **Scoped Restrictions**: Using multiple criteria to narrow down which resources need specific security controls.
- **Label/Annotation-based Enforcement**: Enforcing different security standards based on labels or annotations.

**Only warn for these CLEAR problems:**

1. **No Realistic Use Case**: You cannot come up with a practical, realistic scenario where this policy would be useful, OR the only use case is highly contrived/artificial
2. **Overly Broad**: Does this policy trigger on nearly EVERY deployment without meaningful filtering? (e.g., single criterion with a very permissive threshold with no other filters - but if combined with other criteria for conditional logic, this is valid)
3. **Logically Impossible**: Are there field combinations that make it mathematically impossible to trigger? (e.g., same field checked for X AND NOT X simultaneously, or mutually exclusive values for the same field)
4. **Clear Misunderstanding**: Is there overwhelming evidence the user misunderstood inverted fields? (e.g., ONLY an inverted field with no other criteria, or description explicitly contradicts the behavior)
5. **Contradicts Explicit Purpose**: Does the policy name/description/rationale EXPLICITLY state the opposite of what the criteria actually do?
6. **Always True/False**: Are there conditions that are ALWAYS true or ALWAYS false with no useful filtering? (e.g., checking for a field that is never empty - nearly always true)

**DO NOT warn for:**
- Narrow policies targeting specific scenarios (they may be intentionally specific)
- Complex AND logic across multiple fields (legitimate security requirements)
- Policies with multiple criteria that create conditional enforcement patterns

If you identify a CLEAR problem from the list above AND you are VERY CONFIDENT it's an error, output a WARNING section BEFORE your explanation:

--- WARNING ---
[Brief, specific warning about the likely mistake. Be direct and actionable. Use correct "trigger" vs "enforce" language. Examples:
- "This policy has no realistic use case - it's unclear why an organization would want these trigger conditions."
- "This policy will trigger on nearly every deployment because it uses a very permissive criterion that most resources satisfy. Consider making the criteria more specific."
- "This policy appears to misunderstand inverted fields - the trigger conditions contradict the name/description. Remember: policies trigger on BAD states to enforce GOOD states."]
--- END WARNING ---

[Your normal explanation follows here]

If the policy appears reasonable, has ANY plausible use case, or you are uncertain, do NOT include a WARNING section - just output the explanation as normal.

The WARNING section should be brief (1-3 sentences), specific about the issue, and suggest what the user might want to reconsider.
---

CRITICAL - COHERENCE:
After you have crafted your response, read it again thoroughly and ensure all the parts are coherent with each other and with the previous instructions. Pay special attention to whether the explanation of each field aligns with the explanation of the overall policy, the warning (if any) and the examples/truth table.

`;
}

/**
 * Parses LLM response to extract warning (if present) and explanation
 */
function parseResponse(text: string): PolicyExplanationResult {
    const warningStartMarker = '--- WARNING ---';
    const warningEndMarker = '--- END WARNING ---';

    const warningStart = text.indexOf(warningStartMarker);

    if (warningStart === -1) {
        // No warning present
        return { explanation: text.trim() };
    }

    const warningEnd = text.indexOf(warningEndMarker, warningStart);

    if (warningEnd === -1) {
        // Malformed warning section, treat entire text as explanation
        return { explanation: text.trim() };
    }

    // Extract warning text (between markers)
    const warningText = text.substring(warningStart + warningStartMarker.length, warningEnd).trim();

    // Extract explanation (everything after end marker)
    const explanation = text.substring(warningEnd + warningEndMarker.length).trim();

    return {
        explanation,
        warning: warningText || undefined,
    };
}

/**
 * Calls Vertex AI API to generate policy explanation using OAuth access token
 * Supports both Gemini and Claude models
 */
export async function generatePolicyExplanation(
    policy: BasePolicy | Policy | ClientPolicy
): Promise<PolicyExplanationResult> {
    const { projectId, location, model, accessToken } = VERTEX_AI_CONFIG;

    if (!accessToken) {
        throw new Error(
            'Vertex AI access token not configured. Set VITE_VERTEX_ACCESS_TOKEN environment variable. Generate token with: gcloud auth print-access-token'
        );
    }

    const prompt = buildPrompt(policy);
    const isClaude = model.includes('claude');

    if (isClaude) {
        return generateWithClaude(projectId, location, model, accessToken, prompt);
    }
    return generateWithGemini(projectId, location, model, accessToken, prompt);
}

async function generateWithClaude(
    projectId: string,
    location: string,
    model: string,
    accessToken: string,
    prompt: string
): Promise<PolicyExplanationResult> {
    const endpoint = `https://${location}-aiplatform.googleapis.com/v1/projects/${projectId}/locations/${location}/publishers/anthropic/models/${model}:rawPredict`;

    const requestBody = {
        anthropic_version: 'vertex-2023-10-16',
        messages: [
            {
                role: 'user',
                content: prompt,
            },
        ],
        max_tokens: 4096,
        temperature: 0.7,
        // Note: Claude models don't allow both temperature and top_p
    };

    try {
        const response = await fetch(endpoint, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                Authorization: `Bearer ${accessToken}`,
            },
            body: JSON.stringify(requestBody),
        });

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`Claude API error: ${response.status} - ${errorText}`);
        }

        const data = (await response.json()) as ClaudeResponse;

        if (data.content && data.content.length > 0) {
            const { text } = data.content[0];

            if (data.stop_reason && data.stop_reason !== 'end_turn') {
                // Log warning about incomplete response (commented to avoid linting errors)
                // console.warn('Response may be incomplete. Stop reason:', data.stop_reason);
            }

            return parseResponse(text);
        }

        throw new Error('No response generated from Claude');
    } catch (error) {
        if (error instanceof Error) {
            throw new Error(`Failed to generate policy explanation: ${error.message}`);
        }
        throw new Error('Failed to generate policy explanation: Unknown error');
    }
}

async function generateWithGemini(
    projectId: string,
    location: string,
    model: string,
    accessToken: string,
    prompt: string
): Promise<PolicyExplanationResult> {
    const endpoint = `https://${location}-aiplatform.googleapis.com/v1/projects/${projectId}/locations/${location}/publishers/google/models/${model}:generateContent`;

    const requestBody = {
        contents: [
            {
                role: 'user',
                parts: [{ text: prompt }],
            },
        ],
        generationConfig: {
            temperature: 0.7,
            maxOutputTokens: 4096,
            topP: 0.8,
            topK: 40,
        },
    };

    try {
        const response = await fetch(endpoint, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                Authorization: `Bearer ${accessToken}`,
            },
            body: JSON.stringify(requestBody),
        });

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`Gemini API error: ${response.status} - ${errorText}`);
        }

        const data = (await response.json()) as GeminiResponse;

        if (data.candidates && data.candidates.length > 0) {
            const candidate = data.candidates[0];
            const { text } = candidate.content.parts[0];

            if (candidate.finishReason && candidate.finishReason !== 'STOP') {
                // Log warning about incomplete response (commented to avoid linting errors)
                // console.warn('Response may be incomplete. Finish reason:', candidate.finishReason);
            }

            return parseResponse(text);
        }

        throw new Error('No response generated from Gemini');
    } catch (error) {
        if (error instanceof Error) {
            throw new Error(`Failed to generate policy explanation: ${error.message}`);
        }
        throw new Error('Failed to generate policy explanation: Unknown error');
    }
}
