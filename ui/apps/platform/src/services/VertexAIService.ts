import type { BasePolicy, Policy, ClientPolicy, PolicyGroup, PolicySection } from 'types/policy.proto';
import {
    policyCriteriaDescriptors,
    auditLogDescriptor,
} from 'Containers/Policies/Wizard/Step3/policyCriteriaDescriptors';

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
console.log('Vertex AI Config Debug:', {
    projectId: VERTEX_AI_CONFIG.projectId,
    location: VERTEX_AI_CONFIG.location,
    model: VERTEX_AI_CONFIG.model,
    hasAccessToken: !!VERTEX_AI_CONFIG.accessToken,
    accessTokenLength: VERTEX_AI_CONFIG.accessToken?.length || 0,
    envModel: import.meta.env.VITE_VERTEX_MODEL,
    allEnvVars: Object.keys(import.meta.env).filter(k => k.startsWith('VITE_')),
});

// Combine all descriptors for lookup
const allDescriptors = [...policyCriteriaDescriptors, ...auditLogDescriptor];

// Create a map for quick lookup by field name
const descriptorMap = new Map(allDescriptors.map((d) => [d.name, d]));

type GeminiResponse = {
    candidates: Array<{
        content: {
            parts: Array<{
                text: string;
            }>;
        };
        finishReason?: string;
    }>;
};

type ClaudeResponse = {
    content: Array<{
        text: string;
    }>;
    stop_reason: string;
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
                const fieldName = group.fieldName;
                const operator = group.booleanOperator;
                const negate = group.negate ? 'NOT ' : '';
                const values = group.values.map((v) => v.value).join(`, ${operator} `);
                
                // Look up the descriptor for this field
                const descriptor = descriptorMap.get(fieldName);
                // Use shortName (what users see in UI) instead of name (internal field name)
                const displayName = descriptor?.shortName || fieldName;
                const description = descriptor?.description 
                    ? `\n    Description: ${descriptor.description}`
                    : '';
                
                // Flag fields with inverted/negated matching behavior
                const negatedWarning = isNegatedField(fieldName)
                    ? '\n    ⚠️ INVERTED FIELD: This criterion matches when the condition is NOT met (e.g., "Required Label" matches when label is ABSENT)'
                    : '';
                
                return `  ${negate}${displayName}: ${values}${description}${negatedWarning}`;
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
                if (s.cluster) parts.push(`Cluster: ${s.cluster}`);
                if (s.namespace) parts.push(`Namespace: ${s.namespace}`);
                if (s.label) parts.push(`Label: ${s.label.key}=${s.label.value}`);
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

CRITICAL - INVERTED/NEGATED FIELDS: Some fields are marked with "⚠️ INVERTED FIELD" which means they match when the specified condition is NOT met. This is a common source of user confusion:
- "Required Image Label" matches when the label is ABSENT (not present)
- "Disallowed Image Label" matches when the label is PRESENT
- "Image Signature Verified By" matches when the signature CANNOT be verified (is missing or invalid)

When explaining inverted fields, be EXTREMELY CLEAR about the matching behavior. Use phrases like:
- "matches when the label is ABSENT" (not just "requires label")
- "triggers when the image signature CANNOT be verified" (not just "image signature")
- "fires when the annotation is MISSING" (not just "required annotation")

Generate a technical explanation focused ONLY on when this policy triggers violations. Use PLAIN TEXT with simple formatting markers for emphasis.

Start with a one-sentence summary of when the overall policy triggers.

[blank line]

Then explain the trigger conditions:
- If there is ONLY ONE policy section: state it directly as a bullet, emphasizing that ALL listed fields must trigger for the violation to occur
- If there are MULTIPLE policy sections: say "The policy triggers if ANY of the following conditions are met:" (sections are ALWAYS combined with OR), then emphasize for each section that ALL its fields must trigger

IMPORTANT: Make it very clear that within each section, ALL field criteria must trigger simultaneously (AND logic). This is especially critical for inverted fields - users often expect OR behavior but get AND. Use phrases like:
- "The policy triggers when **ALL** of the following are true:"
- "This violation fires only when **BOTH** of these conditions match:"
- "**ALL** these field requirements must trigger simultaneously:"

For inverted fields in particular, explicitly state the matching behavior:
- WRONG: "Required Image Label: app=myapp"
- RIGHT: "Image must be MISSING the label app=myapp (the 'Required Label' field triggers when the label is ABSENT)"

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

Formatting rules:
- Use bullet symbol (•) for main points, dash (-) for sub-items with 4 spaces of indentation
- Use **double asterisks** around key terms for emphasis (field names, important conditions, AND/OR/ALL/ABSENT/PRESENT/CANNOT keywords)
- Add blank lines between major sections
- Include scope/exclusion info if relevant
- For inverted fields, ALWAYS clarify whether you mean "present" or "absent"

Do NOT explain why the policy matters or remediation steps. Keep under 300 words but COMPLETE all sentences.`;
}

/**
 * Calls Vertex AI API to generate policy explanation using OAuth access token
 * Supports both Gemini and Claude models
 */
export async function generatePolicyExplanation(policy: BasePolicy | Policy | ClientPolicy): Promise<string> {
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
    } else {
        return generateWithGemini(projectId, location, model, accessToken, prompt);
    }
}

async function generateWithClaude(
    projectId: string,
    location: string,
    model: string,
    accessToken: string,
    prompt: string
): Promise<string> {
    const endpoint = `https://${location}-aiplatform.googleapis.com/v1/projects/${projectId}/locations/${location}/publishers/anthropic/models/${model}:rawPredict`;

    const requestBody = {
        anthropic_version: 'vertex-2023-10-16',
        messages: [
            {
                role: 'user',
                content: prompt,
            },
        ],
        max_tokens: 2048,
        temperature: 0.7,
        // Note: Claude models don't allow both temperature and top_p
    };

    try {
        const response = await fetch(endpoint, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${accessToken}`,
            },
            body: JSON.stringify(requestBody),
        });

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`Claude API error: ${response.status} - ${errorText}`);
        }

        const data = (await response.json()) as ClaudeResponse;

        if (data.content && data.content.length > 0) {
            const text = data.content[0].text;
            
            if (data.stop_reason && data.stop_reason !== 'end_turn') {
                console.warn('Response may be incomplete. Stop reason:', data.stop_reason);
            }
            
            return text;
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
): Promise<string> {
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
            maxOutputTokens: 2048,
            topP: 0.8,
            topK: 40,
        },
    };

    try {
        const response = await fetch(endpoint, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${accessToken}`,
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
            const text = candidate.content.parts[0].text;
            
            if (candidate.finishReason && candidate.finishReason !== 'STOP') {
                console.warn('Response may be incomplete. Finish reason:', candidate.finishReason);
            }
            
            return text;
        }

        throw new Error('No response generated from Gemini');
    } catch (error) {
        if (error instanceof Error) {
            throw new Error(`Failed to generate policy explanation: ${error.message}`);
        }
        throw new Error('Failed to generate policy explanation: Unknown error');
    }
}

