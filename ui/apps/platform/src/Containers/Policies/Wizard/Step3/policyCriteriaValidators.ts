import type { ClientPolicy, ClientPolicyGroup, ClientPolicySection } from 'types/policy.proto';

function policyGroupsHasCriterion(
    policyGroups: ClientPolicyGroup[],
    criterionName: string
): boolean {
    return policyGroups.some((g) => g.fieldName === criterionName);
}

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
        validate: ({ policyGroups }) => {
            // Either a built-in resource type or an API resource identifies the resource.
            const hasResource =
                policyGroupsHasCriterion(policyGroups, 'Kubernetes Resource') ||
                policyGroupsHasCriterion(policyGroups, 'Kubernetes API Resource');
            const hasVerb = policyGroupsHasCriterion(policyGroups, 'Kubernetes API Verb');

            if (!hasResource && !hasVerb) {
                return 'Criteria must be present for audit log policies: Kubernetes resource type (or Kubernetes API resource) and Kubernetes API verb';
            }
            if (!hasResource) {
                return 'Criterion must be present for audit log policies: Kubernetes resource type or Kubernetes API resource';
            }
            if (!hasVerb) {
                return 'Criterion must be present for audit log policies: Kubernetes API verb';
            }
            return undefined;
        },
    },
    {
        name: 'Audit log resource criteria are mutually exclusive',
        appliesTo: (context) => context.eventSource === 'AUDIT_LOG_EVENT',
        validate: ({ policyGroups }) => {
            if (
                policyGroupsHasCriterion(policyGroups, 'Kubernetes Resource') &&
                policyGroupsHasCriterion(policyGroups, 'Kubernetes API Resource')
            ) {
                return 'Criteria cannot be combined: Kubernetes resource type and Kubernetes API resource';
            }
            return undefined;
        },
    },
    {
        name: 'File operation requires file path',
        appliesTo: (context) =>
            context.lifecycleStages.includes('RUNTIME') &&
            (context.eventSource === 'NODE_EVENT' || context.eventSource === 'DEPLOYMENT_EVENT'),
        validate: ({ policyGroups }) => {
            const hasFileOperation = policyGroupsHasCriterion(policyGroups, 'File Operation');
            const hasFilePath = policyGroupsHasCriterion(policyGroups, 'File Path');

            if (hasFileOperation && !hasFilePath) {
                return 'Criterion must be present with at least one value when using File operation: File Path';
            }
            return undefined;
        },
    },
    {
        name: 'Process criteria require file path',
        appliesTo: (context) =>
            context.lifecycleStages.includes('RUNTIME') && context.eventSource === 'NODE_EVENT',
        validate: ({ policyGroups }) => {
            const processCriteria = [
                'Process Name',
                'Process Ancestor',
                'Process Arguments',
                'Process UID',
            ];
            const hasProcessCriterion = processCriteria.some((name) =>
                policyGroupsHasCriterion(policyGroups, name)
            );
            const hasFilePath = policyGroupsHasCriterion(policyGroups, 'File Path');

            if (hasProcessCriterion && !hasFilePath) {
                return 'Criterion must be present when using process criteria: File Path';
            }
            return undefined;
        },
    },
];
