import isPlainObject from 'lodash/isPlainObject';
import pluralize from 'pluralize';
import qs, { ParsedQs } from 'qs';

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
} from 'types/policy.proto';
import { SearchFilter } from 'types/search';
import { ExtendedPageAction } from 'utils/queryStringUtils';

function isValidAction(action: unknown): action is ExtendedPageAction {
    return action === 'clone' || action === 'create' || action === 'edit';
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

/*
 * Return request query string for search filter. Omit filter criterion:
 * If option does not have value.
 */
export function getRequestQueryStringForSearchFilter(searchFilter: SearchFilter): string {
    return Object.entries(searchFilter)
        .filter(([, value]) => value.length !== 0)
        .map(([key, value]) => `${key}:${Array.isArray(value) ? value.join(',') : value}`)
        .join('+');
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
    const excludedDeployments: PolicyExcludedDeployment[] = [];

    exclusions.forEach(({ deployment }) => {
        if (deployment?.name || deployment?.scope) {
            excludedDeployments.push(deployment);
        }
    });

    return excludedDeployments;
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

export function getClientWizardPolicy(policy): Policy {
    return preFormatExclusionField(policy);
}

export function getServerPolicy(policy): Policy {
    return postFormatExclusionField(policy);
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
function postFormatExclusionField(policy): Policy {
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
