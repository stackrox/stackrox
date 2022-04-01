import isPlainObject from 'lodash/isPlainObject';
import pluralize from 'pluralize';
import qs, { ParsedQs } from 'qs';
import cloneDeep from 'lodash/cloneDeep';

import integrationsList from 'Containers/Integrations/utils/integrationsList';
import { eventSourceLabels, lifecycleStageLabels } from 'messages/common';
import { Cluster } from 'types/cluster.proto';
import { NotifierIntegration } from 'types/notifier.proto';
import {
    EnforcementAction,
    LifecycleStage,
    PolicyEventSource,
    PolicyExcludedDeployment,
    PolicyExclusion,
    Policy,
    ValueObj,
    PolicyValue,
} from 'types/policy.proto';
import { SearchFilter } from 'types/search';
import { ExtendedPageAction } from 'utils/queryStringUtils';
import { imageSigningCriteriaName } from '../Wizard/Form/descriptors';

function isValidAction(action: unknown): action is ExtendedPageAction {
    return action === 'clone' || action === 'create' || action === 'edit' || action === 'generate';
}

function isParsedQs(s: unknown): s is ParsedQs {
    return isPlainObject(s);
}

function isValidFilterValue(value: unknown): value is string | string[] {
    if (typeof value === 'string') {
        return true;
    }

    if (Array.isArray(value) && value.every((item) => typeof item === 'string')) {
        return true;
    }

    return false;
}

function isValidFilter(s: unknown): s is SearchFilter {
    return isParsedQs(s) && Object.values(s).every((value) => isValidFilterValue(value));
}

export type PoliciesSearch = {
    pageAction?: ExtendedPageAction;
    searchFilter?: SearchFilter;
};

/*
 * Given search query string from location, return validated action string and filter object.
 *
 * Examples of search query string:
 * ?action=create
 * ?action=edit
 * ?s[Lifecycle Stage]=BUILD
 * ?s[Lifecycle Stage]=BUILD&s[Lifecycle State]=DEPLOY
 * ?s[Lifecycle State]=RUNTIME&s[Severity]=CRITICAL_SEVERITY
 */
export function parsePoliciesSearchString(search: string): PoliciesSearch {
    const { action, s } = qs.parse(search, { ignoreQueryPrefix: true });

    return {
        pageAction: isValidAction(action) ? action : undefined,
        searchFilter: isValidFilter(s) ? s : undefined,
    };
}

export function getSearchStringForFilter(s: SearchFilter): string {
    return qs.stringify(
        { s },
        {
            arrayFormat: 'repeat',
            encodeValuesOnly: true,
        }
    );
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
    return integrationsList.notifiers.find((notifier) => notifier.type === type)?.label ?? '';
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
    return integrationsList.notifiers.map(({ label, type }) => [
        label,
        notifiers.filter((notifier) => notifier.type === type).map(({ id }) => id),
    ]);
}

// scope

export function getClusterName(clusters: Cluster[], clusterId: string): string {
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

function preFormatNestedPolicyFields(policy: Policy): Policy {
    if (!policy.policySections) {
        return policy;
    }

    const clientPolicy = cloneDeep(policy);
    clientPolicy.serverPolicySections = policy.policySections;
    // itreating through each value in a policy group in a policy section to parse value string
    policy.policySections.forEach((policySection, sectionIdx) => {
        const { policyGroups } = policySection;
        policyGroups.forEach((policyGroup, groupIdx) => {
            const { values, fieldName } = policyGroup;
            values.forEach((value, valueIdx) => {
                clientPolicy.policySections[sectionIdx].policyGroups[groupIdx].values[valueIdx] =
                    parseValueStr(value.value, fieldName) as PolicyValue;
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

function postFormatNestedPolicyFields(policy: Policy): Policy {
    if (!policy.policySections) {
        return policy;
    }

    const serverPolicy = cloneDeep(policy);
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
function preFormatExclusionField(policy): Policy {
    const { exclusions } = policy;
    const clientPolicy: Policy = { ...policy };

    clientPolicy.excludedImageNames =
        exclusions.filter((o) => !!o.image?.name).map((o) => o.image.name as string) ?? [];

    clientPolicy.excludedDeploymentScopes = exclusions
        .filter((o) => !!o.deployment?.name || !!o.deployment?.scope)
        .map((o) => o.deployment as PolicyExcludedDeployment);

    return clientPolicy;
}

/*
 * Merge client-wizard excludedDeploymentScopes and excludedImageNames properties into server exclusions property.
 */
export function postFormatExclusionField(policy): Policy {
    const serverPolicy: Policy = { ...policy };
    serverPolicy.exclusions = [];

    const { excludedDeploymentScopes } = policy;
    if (excludedDeploymentScopes && excludedDeploymentScopes.length) {
        serverPolicy.exclusions = serverPolicy.exclusions.concat(
            excludedDeploymentScopes.map((deployment) => ({ deployment }))
        );
    }

    const { excludedImageNames } = policy;
    if (excludedImageNames && excludedImageNames.length > 0) {
        serverPolicy.exclusions = serverPolicy.exclusions.concat(
            excludedImageNames.map((name) => ({ image: { name } }))
        );
    }

    return serverPolicy;
}

export function postFormatImageSigningPolicyGroup(policy: Policy): Policy {
    if (!policy.policySections) {
        return policy;
    }

    const serverPolicy = cloneDeep(policy);
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

export function preFormatImageSigningPolicyGroup(policy: Policy): Policy {
    if (!policy.policySections) {
        return policy;
    }

    const clientPolicy = cloneDeep(policy);
    policy.policySections.forEach((policySection, sectionIdx) => {
        const { policyGroups } = policySection;
        policyGroups.forEach((policyGroup, groupIdx) => {
            const { values, fieldName } = policyGroup;
            if (fieldName === imageSigningCriteriaName) {
                const arrayValue = values.map((v) => v.value as string);
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

export function getClientWizardPolicy(policy): Policy {
    let formattedPolicy = preFormatExclusionField(policy);
    formattedPolicy = preFormatNestedPolicyFields(formattedPolicy);
    formattedPolicy = preFormatImageSigningPolicyGroup(formattedPolicy);
    return formattedPolicy;
}

export function getServerPolicy(policy): Policy {
    let serverPolicy = postFormatExclusionField(policy);
    serverPolicy = postFormatImageSigningPolicyGroup(serverPolicy);
    serverPolicy = postFormatNestedPolicyFields(serverPolicy);
    return serverPolicy;
}
