import pluralize from 'pluralize';
import qs from 'qs';
import cloneDeep from 'lodash/cloneDeep';

import {
    policyConfigurationDescriptor,
    auditLogDescriptor,
    imageSigningCriteriaName,
    Descriptor,
} from 'Containers/Policies/Wizard/Step3/policyCriteriaDescriptors';
import { notifierIntegrationsDescriptors } from 'Containers/Integrations/utils/integrationsList';
import { eventSourceLabels, lifecycleStageLabels } from 'messages/common';
import { ClusterScopeObject } from 'services/RolesService';
import { NotifierIntegration } from 'types/notifier.proto';
import {
    EnforcementAction,
    LifecycleStage,
    PolicyEventSource,
    PolicyExcludedDeployment,
    PolicyExclusion,
    Policy,
    ClientPolicy,
    ValueObj,
    ClientPolicyValue,
    PolicyDeploymentExclusion,
    PolicyImageExclusion,
    PolicyScope,
} from 'types/policy.proto';
import { SearchFilter } from 'types/search';
import { ExtendedPageAction } from 'utils/queryStringUtils';
import { checkArrayContainsArray } from 'utils/arrayUtils';

function isValidAction(action: unknown): action is ExtendedPageAction {
    return action === 'clone' || action === 'create' || action === 'edit' || action === 'generate';
}

export const initialPolicy: ClientPolicy = {
    id: '',
    name: '',
    description: '',
    severity: 'LOW_SEVERITY',
    disabled: false,
    lifecycleStages: [],
    notifiers: [],
    lastUpdated: null,
    eventSource: 'NOT_APPLICABLE',
    isDefault: false,
    rationale: '',
    remediation: '',
    categories: [],
    exclusions: [],
    scope: [],
    enforcementActions: [],
    excludedImageNames: [],
    excludedDeploymentScopes: [],
    SORTName: '', // For internal use only.
    SORTLifecycleStage: '', // For internal use only.
    SORTEnforcement: false, // For internal use only.
    policyVersion: '',
    serverPolicySections: [],
    policySections: [
        {
            sectionName: 'Policy Section 1',
            policyGroups: [],
        },
    ],
    mitreAttackVectors: [],
    criteriaLocked: false,
    mitreVectorsLocked: false,
};

export type PoliciesSearch = {
    pageAction?: ExtendedPageAction;
    searchFilter?: SearchFilter;
};

/*
 * Given search query string from location, return validated action string.
 *
 * Examples of search query string:
 * ?action=create
 * ?action=edit
 */
export function parsePoliciesSearchString(search: string): PoliciesSearch {
    const { action } = qs.parse(search, { ignoreQueryPrefix: true });

    return {
        pageAction: isValidAction(action) ? action : undefined,
    };
}

// categories

export function formatCategories(categories: string[]): string {
    return categories.join(', ');
}

// enforcementActions

export const lifecycleStagesToEnforcementActionsMap: Record<LifecycleStage, EnforcementAction[]> = {
    BUILD: ['FAIL_BUILD_ENFORCEMENT'],
    DEPLOY: ['SCALE_TO_ZERO_ENFORCEMENT', 'UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT'],
    RUNTIME: ['KILL_POD_ENFORCEMENT', 'FAIL_KUBE_REQUEST_ENFORCEMENT'],
};

export function hasEnforcementActionForLifecycleStage(
    lifecycleStage: LifecycleStage,
    enforcementActions: EnforcementAction[]
) {
    const enforcementActionsForLifecycleStage =
        lifecycleStagesToEnforcementActionsMap[lifecycleStage];

    return enforcementActions.some((enforcementAction) =>
        enforcementActionsForLifecycleStage.includes(enforcementAction)
    );
}

export function getEnforcementLifecycleStages(
    lifecycleStages: LifecycleStage[],
    enforcementActions: EnforcementAction[]
): LifecycleStage[] {
    return lifecycleStages.filter((lifecycleStage) => {
        return hasEnforcementActionForLifecycleStage(lifecycleStage, enforcementActions);
    });
}

export function formatResponse(enforcementLifecycleStages: LifecycleStage[]): string {
    return enforcementLifecycleStages.length === 0 ? 'Inform' : 'Enforce';
}

export function appendEnforcementActionsForAddedLifecycleStage(
    lifecycleStage: LifecycleStage,
    enforcementActions: EnforcementAction[]
): EnforcementAction[] {
    return [...enforcementActions, ...lifecycleStagesToEnforcementActionsMap[lifecycleStage]];
}

export function filterEnforcementActionsForRemovedLifecycleStage(
    lifecycleStage: LifecycleStage,
    enforcementActions: EnforcementAction[]
): EnforcementAction[] {
    return enforcementActions.filter(
        (enforcementAction) =>
            !lifecycleStagesToEnforcementActionsMap[lifecycleStage].includes(enforcementAction)
    );
}

// eventSource

export function formatEventSource(eventSource: PolicyEventSource): string {
    return eventSourceLabels[eventSource];
}

// exclusions

export function getExcludedDeployments(exclusions: PolicyExclusion[]): PolicyExcludedDeployment[] {
    const excludedDeploymentScopes: PolicyExcludedDeployment[] = [];

    exclusions.forEach(({ deployment }) => {
        if (deployment?.name || deployment?.scope) {
            excludedDeploymentScopes.push(deployment);
        }
    });

    return excludedDeploymentScopes;
}

export function getExcludedImageNames(exclusions: PolicyExclusion[]): string[] {
    const excludedImageNames: string[] = [];

    exclusions.forEach(({ image }) => {
        if (image?.name) {
            excludedImageNames.push(image.name);
        }
    });

    return excludedImageNames;
}

// isDefault

export function formatType(isDefault: boolean): string {
    return isDefault ? 'System default' : 'User generated';
}

// lifecycleStages

export function formatLifecycleStages(lifecycleStages: LifecycleStage[]): string {
    return lifecycleStages.map((lifecycleStage) => lifecycleStageLabels[lifecycleStage]).join(', ');
}

// notifiers

export function getNotifierTypeLabel(type: string): string {
    return notifierIntegrationsDescriptors.find((notifier) => notifier.type === type)?.label ?? '';
}

/*
 * Given array of label-with-ids tuples for notifier and array of notifier ids for a policy,
 * return an array of count-with-label strings:
 * [] if policy does not have notifier ids
 * ['N notifiers'] if policy has notifier ids, but notifiers request does not (yet) have a response
 * ['N Slack'] for example, if notifier ids have the same type
 * ['N Slack', 'N Webhook'] for example, if notifier ids have different types
 */
export function formatNotifierCountsWithLabelStrings(
    labelAndNotifierIdsForTypes: LabelAndNotifierIdsForType[],
    notifierIds: string[]
): string[] {
    const notifierCountsWithLabelStrings: string[] = [];
    let countWithLabel = 0;

    labelAndNotifierIdsForTypes.forEach(([labelForType, notifierIdsForType]) => {
        let countForType = 0;

        notifierIds.forEach((notifierId) => {
            if (notifierIdsForType.includes(notifierId)) {
                countForType += 1;
            }
        });

        if (countForType !== 0) {
            notifierCountsWithLabelStrings.push(`${countForType} ${labelForType}`);
            countWithLabel += countForType;
        }
    });

    const countWithoutLabel = notifierIds.length - countWithLabel;
    if (countWithoutLabel !== 0) {
        notifierCountsWithLabelStrings.push(
            `${countWithoutLabel} ${pluralize('notifier', countWithoutLabel)}`
        );
    }

    return notifierCountsWithLabelStrings;
}

export type LabelAndNotifierIdsForType = [string, string[]];

/*
 * Given notifier integrations, return array of tuples:
 * label for type (for example, 'Slack' for 'slack')
 * notifier ids for type
 */

export function getLabelAndNotifierIdsForTypes(
    notifiers: NotifierIntegration[]
): LabelAndNotifierIdsForType[] {
    return notifierIntegrationsDescriptors.map(({ label, type }) => [
        label,
        notifiers.filter((notifier) => notifier.type === type).map(({ id }) => id),
    ]);
}

// scope

export function getClusterName(clusters: ClusterScopeObject[], clusterId: string): string {
    const cluster = clusters.find(({ id }) => id === clusterId);
    return cluster?.name ?? clusterId;
}

/* PolicyWizard steps */

export type WizardPolicyStep4 = {
    scope: WizardScope[];
    excludedDeploymentScopes: WizardExcludedDeployment[];
    excludedImageNames: string[];
};

export type WizardExcludedDeployment = {
    name?: string;
    scope: WizardScope;
};

/*
 * WizardScope whose label object whose properties have either empty string or undefined values
 * corresponds to PolicyScope label value null.
 */

export type WizardScope = {
    cluster?: string;
    namespace?: string;
    label: WizardScopeLabel | null;
};

export type WizardScopeLabel = {
    key?: string;
    value?: string;
};

export const initialScope: WizardScope = {
    cluster: '',
    namespace: '',
    label: {},
};

export const initialExcludedDeployment: WizardExcludedDeployment = {
    name: '',
    scope: initialScope,
};

// TODO: work with API to update contract for returning number comparison fields
//   until that improves, we short-circuit those fields here
const nonStandardNumberFields = [
    'CVSS',
    'Container CPU Request',
    'Container CPU Limit',
    'Container Memory Request',
    'Container Memory Limit',
    'Replicas',
    'Severity',
];

function isCompoundField(fieldName = '') {
    const compoundValueFields = [
        'Disallowed Annotation',
        'Disallowed Image Label',
        'Dockerfile Line',
        'Environment Variable',
        'Image Component',
        'Required Annotation',
        'Required Image Label',
        'Required Label',
    ];

    return compoundValueFields.includes(fieldName);
}

const numericCompRe =
    /^([><=]+)?\D*(?=.)(([+-]?([0-9]*)(\.([0-9]+))?)|(UNKNOWN|LOW|MODERATE|IMPORTANT|CRITICAL))$/;

export function parseNumericComparisons(str): [string, string] {
    const matches: string[] = str.match(numericCompRe);
    return [matches[1], matches[2]];
}

export function parseValueStr(value, fieldName): ValueObj {
    // TODO: work with API to update contract for returning number comparison fields
    //   until that improves, we short-circuit those fields here

    if (nonStandardNumberFields.includes(fieldName)) {
        const [comparison, num] = parseNumericComparisons(value);
        return comparison
            ? {
                  key: comparison,
                  value: num,
              }
            : {
                  key: '=',
                  value: num,
              };
    }
    if (typeof value === 'string' && isCompoundField(fieldName)) {
        // handle all other string fields
        const valueArr = value.split('=');
        // for nested policy criteria fields
        if (valueArr.length === 2) {
            return {
                key: valueArr[0],
                value: valueArr[1],
            };
        }
        // for the Environment Variable policy criteria
        if (valueArr.length === 3) {
            return {
                source: valueArr[0],
                key: valueArr[1],
                value: valueArr[2],
            };
        }
    }
    return {
        value,
    };
}

function preFormatNestedPolicyFields(policy: Policy): ClientPolicy {
    if (!policy.policySections) {
        // TS2352: Conversion of type 'Policy' to type 'ClientPolicy' may be a mistake because neither type sufficiently overlaps with the other.
        // If this was intentional, convert the expression to 'unknown' first.
        return policy as unknown as ClientPolicy;
    }

    // TS2352: Conversion of type 'Policy' to type 'ClientPolicy' may be a mistake because neither type sufficiently overlaps with the other.
    // If this was intentional, convert the expression to 'unknown' first.
    const clientPolicy = cloneDeep(policy) as unknown as ClientPolicy;
    clientPolicy.serverPolicySections = policy.policySections;
    // itreating through each value in a policy group in a policy section to parse value string
    policy.policySections.forEach((policySection, sectionIdx) => {
        const { policyGroups } = policySection;
        policyGroups.forEach((policyGroup, groupIdx) => {
            const { values, fieldName } = policyGroup;
            values.forEach((value, valueIdx) => {
                clientPolicy.policySections[sectionIdx].policyGroups[groupIdx].values[valueIdx] =
                    parseValueStr(value.value, fieldName) as ClientPolicyValue;
            });
        });
    });
    return clientPolicy;
}

export function formatValueStr(valueObj: ValueObj, fieldName: string): string {
    if (!valueObj) {
        return '';
    }
    const { source, key = '', value = '' } = valueObj;
    let valueStr = value;

    if (nonStandardNumberFields.includes(fieldName)) {
        // TODO: work with API to update contract for returning number comparison fields
        //   until that improves, we short-circuit those fields here
        valueStr = key !== '=' ? `${key}${value}` : `${value}`;
    } else if (source || fieldName === 'Environment Variable') {
        valueStr = `${source || ''}=${key}=${value}`;
    } else if (key) {
        valueStr = `${key}=${value}`;
    }
    return valueStr ?? '';
}

function postFormatNestedPolicyFields(policy: ClientPolicy): Policy {
    if (!policy.policySections) {
        return policy;
    }

    const serverPolicy = cloneDeep(policy) as Policy;
    if (policy.criteriaLocked) {
        serverPolicy.policySections = policy.serverPolicySections;
    } else {
        // itereating through each value in a policy group in a policy section to format to a flat value string
        policy.policySections.forEach((policySection, sectionIdx) => {
            const { policyGroups } = policySection;
            policyGroups.forEach((policyGroup, groupIdx) => {
                const { values } = policyGroup;
                values.forEach((value, valueIdx) => {
                    serverPolicy.policySections[sectionIdx].policyGroups[groupIdx].values[
                        valueIdx
                    ] = {
                        value: formatValueStr(value as ValueObj, policyGroup.fieldName),
                    };
                });
                // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                // @ts-ignore
                delete serverPolicy.policySections[sectionIdx].policyGroups[groupIdx].fieldKey;
            });
        });
    }
    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
    // @ts-ignore
    delete serverPolicy.serverPolicySections;
    return serverPolicy;
}

/*
 * Split server exclusions property into client-wizard excludedDeploymentScopes and excludedImageNames properties.
 */
function preFormatExclusionField(policy: Policy): ClientPolicy {
    const { exclusions } = policy;
    const clientPolicy = { ...policy } as ClientPolicy;

    clientPolicy.excludedImageNames = [];

    const excludedImageNames = exclusions.filter((o) => !!o.image?.name) as PolicyImageExclusion[];
    if (excludedImageNames.length > 0) {
        clientPolicy.excludedImageNames = excludedImageNames.map((o) => o.image.name);
    }

    clientPolicy.excludedDeploymentScopes = exclusions
        .filter((o) => !!o.deployment?.name || !!o.deployment?.scope)
        .map((o) => o.deployment as PolicyExcludedDeployment);

    return clientPolicy;
}

/*
 * Merge client-wizard excludedDeploymentScopes and excludedImageNames properties into server exclusions property.
 */
export function postFormatExclusionField(policy: ClientPolicy): Policy {
    const serverPolicy = { ...policy } as Policy;
    serverPolicy.exclusions = [];

    const { excludedDeploymentScopes } = policy;
    if (excludedDeploymentScopes && excludedDeploymentScopes.length) {
        serverPolicy.exclusions = serverPolicy.exclusions.concat(
            excludedDeploymentScopes.map(
                (deployment) => ({ deployment }) as PolicyDeploymentExclusion
            )
        );
    }

    const { excludedImageNames } = policy;
    if (excludedImageNames && excludedImageNames.length > 0) {
        serverPolicy.exclusions = serverPolicy.exclusions.concat(
            excludedImageNames.map((name) => ({ image: { name } }) as PolicyImageExclusion)
        );
    }

    return serverPolicy;
}

export function preFormatImageSigningPolicyGroup(policy: Policy): ClientPolicy {
    if (!policy.policySections) {
        return policy as ClientPolicy;
    }

    const clientPolicy = cloneDeep(policy) as ClientPolicy;
    policy.policySections.forEach((policySection, sectionIdx) => {
        const { policyGroups } = policySection;
        policyGroups.forEach((policyGroup, groupIdx) => {
            const { values, fieldName } = policyGroup;
            if (fieldName === imageSigningCriteriaName) {
                const arrayValue = values.map((v) => v.value);
                clientPolicy.policySections[sectionIdx].policyGroups[groupIdx].values = [
                    {
                        arrayValue,
                    },
                ];
            }
        });
    });

    return clientPolicy;
}

export function postFormatImageSigningPolicyGroup(policy: ClientPolicy): Policy {
    if (!policy.policySections) {
        return policy;
    }

    const serverPolicy = cloneDeep(policy) as Policy;
    policy.policySections.forEach((policySection, sectionIdx) => {
        const { policyGroups } = policySection;
        policyGroups.forEach((policyGroup, groupIdx) => {
            const { values } = policyGroup;
            if (policyGroup.fieldName === imageSigningCriteriaName) {
                const { arrayValue } = values[0];
                arrayValue?.forEach((value, valueIdx) => {
                    serverPolicy.policySections[sectionIdx].policyGroups[groupIdx].values[
                        valueIdx
                    ] = {
                        value,
                    };
                });
            }
        });
    });

    return serverPolicy;
}

// Impure function assumes caller has cloned the scope!
function trimPolicyScope(scope: PolicyScope) {
    /* eslint-disable no-param-reassign */
    if (typeof scope.cluster === 'string') {
        scope.cluster = scope.cluster.trim();
    }

    if (typeof scope.namespace === 'string') {
        scope.namespace = scope.namespace.trim();
    }

    // TODO label key and value: make sure about empty string versus undefined.
    /*
    if (scope.label) {
        if (typeof scope.label.key === 'string') {
            scope.label.key = scope.label.key.trim();
        }

        if (typeof scope.label.value === 'string') {
            scope.label.value = scope.label.value.trim();
        }
    }
    */
    /* eslint-enable no-param-reassign */

    return scope;
}

function trimClientWizardPolicy(policyUntrimmed: ClientPolicy): ClientPolicy {
    const policy = cloneDeep(policyUntrimmed);

    // Policy details

    policy.name = policy.name.trim();
    policy.description = policy.description.trim();
    policy.rationale = policy.rationale.trim();
    policy.remediation = policy.remediation.trim();

    // Policy criteria

    if (Array.isArray(policy.policySections)) {
        // for instead of forEach to work around no-param-reassign lint error.
        for (let iSection = 0; iSection !== policy.policySections.length; iSection += 1) {
            const policySection = policy.policySections[iSection];

            policySection.sectionName = policySection.sectionName.trim();

            // TODO value: make sure about empty string versus undefined.
            /*
            if (Array.isArray(policySection.policyGroups)) {
                for (let iGroup = 0; iGroup !== policySection.policyGroups.length; iGroup += 1) {
                    const policyGroup = policySection.policyGroups[iGroup];

                    if (Array.isArray(policyGroup.values)) {
                        for (let iValue = 0; iValue !== policyGroup.values.length; iValue += 1) {
                            const valueObject = policyGroup.values[iValue];

                            if (typeof valueObject.value === 'string') {
                                // TODO Investigate ValueObj for ClientPolicyValue.
                                // TS2339 Property does not exist on type never.
                                valueObject.value = valueObject.value.trim();
                            }
                        }
                    }
                }
            }
            */
        }
    }

    // Policy scope

    if (Array.isArray(policy.scope)) {
        // for instead of forEach to work around no-param-reassign lint error.
        for (let i = 0; i !== policy.scope.length; i += 1) {
            trimPolicyScope(policy.scope[i]);
        }
    }

    if (Array.isArray(policy.excludedDeploymentScopes)) {
        // for instead of forEach to work around no-param-reassign lint error.
        for (let i = 0; i !== policy.excludedDeploymentScopes.length; i += 1) {
            const excludedDeploymentScope = policy.excludedDeploymentScopes[i];

            if (excludedDeploymentScope.scope) {
                trimPolicyScope(excludedDeploymentScope.scope);
            }

            if (typeof excludedDeploymentScope.name === 'string') {
                excludedDeploymentScope.name = excludedDeploymentScope.name.trim();
            }
        }
    }

    if (Array.isArray(policy.excludedImageNames)) {
        policy.excludedImageNames = policy.excludedImageNames.map((excludedImageName) =>
            excludedImageName.trim()
        );
    }

    return policy;
}

export function getClientWizardPolicy(policy: Policy): ClientPolicy {
    let formattedPolicy = preFormatExclusionField(policy);
    formattedPolicy = preFormatNestedPolicyFields(formattedPolicy);
    formattedPolicy = preFormatImageSigningPolicyGroup(formattedPolicy);
    return formattedPolicy;
}

// Called before POST dryrunjob request and before POST or PUT policies request for Save.
export function getServerPolicy(policyUntrimmed: ClientPolicy): Policy {
    const policy = trimClientWizardPolicy(policyUntrimmed);
    let serverPolicy = postFormatExclusionField(policy);
    serverPolicy = postFormatImageSigningPolicyGroup(serverPolicy as ClientPolicy);
    serverPolicy = postFormatNestedPolicyFields(serverPolicy as ClientPolicy);
    return serverPolicy;
}

export function getLifeCyclesUpdates(
    values: ClientPolicy,
    lifecycleStage: LifecycleStage,
    isChecked: boolean
) {
    /*
     * Set all changed values at once, because separate setFieldValue calls
     * for lifecycleStages and eventSource cause inconsistent incorrect validation.
     */
    const changedValues = cloneDeep(values);
    if (isChecked) {
        changedValues.lifecycleStages = [...values.lifecycleStages, lifecycleStage];
    } else {
        changedValues.lifecycleStages = values.lifecycleStages.filter(
            (stage) => stage !== lifecycleStage
        );
        if (lifecycleStage === 'RUNTIME') {
            changedValues.eventSource = 'NOT_APPLICABLE';
        }
        if (lifecycleStage === 'BUILD') {
            changedValues.excludedImageNames = [];
        }
        changedValues.enforcementActions = filterEnforcementActionsForRemovedLifecycleStage(
            lifecycleStage,
            values.enforcementActions
        );
    }
    return changedValues;
}

export function getPolicyDescriptors(
    isFeatureFlagEnabled: (string) => boolean,
    eventSource: PolicyEventSource,
    lifecycleStages: LifecycleStage[]
) {
    const unfilteredDescriptors =
        eventSource === 'AUDIT_LOG_EVENT' ? auditLogDescriptor : policyConfigurationDescriptor;

    const descriptors = unfilteredDescriptors.filter((unfilteredDescriptor) => {
        if (typeof unfilteredDescriptor.featureFlagDependency === 'string') {
            return isFeatureFlagEnabled(unfilteredDescriptor.featureFlagDependency);
        }
        return true;
    });

    const descriptorsFilteredByLifecycle = getCriteriaAllowedByLifecycle(
        descriptors,
        lifecycleStages
    );

    return descriptorsFilteredByLifecycle;
}

export function getCriteriaAllowedByLifecycle(
    criteria: Descriptor[],
    lifecycleStages: LifecycleStage[]
) {
    const filteredCriteria = criteria.filter((criterion) =>
        checkArrayContainsArray(criterion.lifecycleStages, lifecycleStages)
    );

    return filteredCriteria;
}

export function getEmptyPolicyFieldCard(fieldKey) {
    const defaultValue = fieldKey.defaultValue !== undefined ? fieldKey.defaultValue : '';
    return {
        fieldName: fieldKey.name,
        booleanOperator: 'OR',
        values: [
            {
                value: defaultValue,
            },
        ],
        negate: false,
        fieldKey,
    };
}
