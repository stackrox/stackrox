import isPlainObject from 'lodash/isPlainObject';
import qs, { ParsedQs } from 'qs';

import { eventSourceLabels, lifecycleStageLabels } from 'messages/common';
import { Cluster } from 'types/cluster.proto';
import {
    EnforcementAction,
    LifecycleStage,
    PolicyEventSource,
    PolicyExcludedDeployment,
    PolicyExclusion,
} from 'types/policy.proto';
import { SearchFilter } from 'types/search';

export type PageAction = 'clone' | 'create' | 'edit';

function isValidAction(action: unknown): action is PageAction {
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
    pageAction?: PageAction;
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

export function getEnforcementLifecycleStages(
    lifecycleStages: LifecycleStage[],
    enforcementActions: EnforcementAction[]
): LifecycleStage[] {
    return lifecycleStages.filter((lifecycleStage) => {
        const enforcementActionsForLifecycleStage =
            lifecycleStagesToEnforcementActionsMap[lifecycleStage];

        return enforcementActions.some((enforcementAction) =>
            enforcementActionsForLifecycleStage.includes(enforcementAction)
        );
    });
}

export function formatResponse(enforcementLifecycleStages: LifecycleStage[]): string {
    return enforcementLifecycleStages.length === 0 ? 'Inform' : 'Enforce';
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

// scope

export function getClusterName(clusters: Cluster[], clusterId: string): string {
    const cluster = clusters.find(({ id }) => id === clusterId);
    return cluster?.name ?? clusterId;
}
