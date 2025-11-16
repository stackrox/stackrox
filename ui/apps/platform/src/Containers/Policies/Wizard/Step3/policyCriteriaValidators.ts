import type { ClientPolicy, ClientPolicySection } from 'types/policy.proto';

export type PolicyContext = Pick<ClientPolicy, 'eventSource' | 'lifecycleStages'>;

export type PolicySectionValidator = {
    name: string;
    appliesTo: (context: PolicyContext) => boolean;
    validate: (section: ClientPolicySection, context: PolicyContext) => string | undefined;
};

/**
 * Registry of section-level validators for policy criteria.
 * Each validator:
 * - Has a descriptive name for debugging
 * - Defines which contexts it applies to (based on event source, lifecycle stages, etc.)
 * - Validates a policy section and returns an error message upon failure, or undefined if successful
 */
export const policySectionValidators: PolicySectionValidator[] = [
    {
        name: 'Audit log required fields',
        appliesTo: (context) => context.eventSource === 'AUDIT_LOG_EVENT',
        validate: (section) => {
            const hasResource = section.policyGroups.some(
                (g) => g.fieldName === 'Kubernetes Resource'
            );
            const hasVerb = section.policyGroups.some((g) => g.fieldName === 'Kubernetes API Verb');

            if (!hasResource && !hasVerb) {
                return 'Criteria must be present for audit log policies: Kubernetes resource type and Kubernetes API verb';
            }
            if (!hasResource) {
                return 'Criterion must be present for audit log policies: Kubernetes resource type';
            }
            if (!hasVerb) {
                return 'Criterion must be present for audit log policies: Kubernetes API verb';
            }
            return undefined;
        },
    },
];
