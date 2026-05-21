import cloneDeep from 'lodash/cloneDeep';

import type {
    EntityScopeRule,
    RuleValue,
    ScopeEntity,
    ScopeField,
} from 'services/ReportsService.types';
import type { SearchFilter } from 'types/search';
import type { SearchFieldLabel } from 'types/searchOptions';
import {
    getValueByCaseInsensitiveKey,
    isQuotedString,
    searchValueAsArray,
    wrapInQuotes,
} from 'utils/searchUtils';

type EntityScopeSearchFieldLabelForCluster = Extract<
    SearchFieldLabel,
    'Cluster ID' | 'Cluster' | 'Cluster Label'
>;

type EntityScopeSearchFieldLabelForClusterNamespace = Extract<
    SearchFieldLabel,
    | 'Cluster ID'
    | 'Cluster'
    | 'Cluster Label'
    | 'Namespace ID'
    | 'Namespace'
    | 'Namespace Label'
    | 'Namespace Annotation'
>;

type EntityScopeSearchFieldLabelForClusterNamespaceDeployment = Extract<
    SearchFieldLabel,
    | 'Cluster ID'
    | 'Cluster'
    | 'Cluster Label'
    | 'Namespace ID'
    | 'Namespace'
    | 'Namespace Label'
    | 'Namespace Annotation'
    | 'Deployment ID'
    | 'Deployment'
    | 'Deployment Label'
    | 'Deployment Annotation'
>;

type EntityScopeRuleWithoutValues = {
    entity: Exclude<ScopeEntity, 'SCOPE_ENTITY_UNSET'>;
    field: Exclude<ScopeField, 'FIELD_UNSET'>;
};

const searchFieldLabelMapForCluster: Record<
    EntityScopeSearchFieldLabelForCluster,
    EntityScopeRuleWithoutValues
> = {
    'Cluster ID': {
        entity: 'SCOPE_ENTITY_CLUSTER',
        field: 'FIELD_ID',
    },
    Cluster: {
        entity: 'SCOPE_ENTITY_CLUSTER',
        field: 'FIELD_NAME',
    },
    'Cluster Label': {
        entity: 'SCOPE_ENTITY_CLUSTER',
        field: 'FIELD_LABEL',
    },
    // 'Cluster Annotation' is not a search filter
} as const;

const searchFieldLabelMapForClusterNamespace: Record<
    EntityScopeSearchFieldLabelForClusterNamespace,
    EntityScopeRuleWithoutValues
> = {
    ...searchFieldLabelMapForCluster,
    'Namespace ID': {
        entity: 'SCOPE_ENTITY_NAMESPACE',
        field: 'FIELD_ID',
    },
    Namespace: {
        entity: 'SCOPE_ENTITY_NAMESPACE',
        field: 'FIELD_NAME',
    },
    'Namespace Label': {
        entity: 'SCOPE_ENTITY_NAMESPACE',
        field: 'FIELD_LABEL',
    },
    'Namespace Annotation': {
        entity: 'SCOPE_ENTITY_NAMESPACE',
        field: 'FIELD_ANNOTATION',
    },
} as const;

export const searchFieldLabelMapForClusterNamespaceDeployment: Record<
    EntityScopeSearchFieldLabelForClusterNamespaceDeployment,
    EntityScopeRuleWithoutValues
> = {
    ...searchFieldLabelMapForClusterNamespace,
    'Deployment ID': {
        entity: 'SCOPE_ENTITY_DEPLOYMENT',
        field: 'FIELD_ID',
    },
    Deployment: {
        entity: 'SCOPE_ENTITY_DEPLOYMENT',
        field: 'FIELD_NAME',
    },
    'Deployment Label': {
        entity: 'SCOPE_ENTITY_DEPLOYMENT',
        field: 'FIELD_LABEL',
    },
    'Deployment Annotation': {
        entity: 'SCOPE_ENTITY_DEPLOYMENT',
        field: 'FIELD_ANNOTATION',
    },
} as const;

// One size omits all of cluster, namespace, deployment, for simplicity.
// For initial query string when ?action=createFromFilters
export function getSearchFilterWithoutEntityScope(
    searchFilterWithEntityScopeRules: SearchFilter
): SearchFilter {
    const searchFilterWithoutEntityScopeRules: SearchFilter = cloneDeep(
        searchFilterWithEntityScopeRules
    );

    Object.entries(searchFilterWithEntityScopeRules).forEach(([searchFieldLabel]) => {
        if (
            getValueByCaseInsensitiveKey(
                searchFieldLabelMapForClusterNamespaceDeployment,
                searchFieldLabel
            )
        ) {
            delete searchFilterWithoutEntityScopeRules[searchFieldLabel];
        }
    });

    return searchFilterWithoutEntityScopeRules;
}

function isLabelOrAnnotationField(field: string): boolean {
    return field === 'FIELD_LABEL' || field === 'FIELD_ANNOTATION';
}

export const searchValueToRuleValue = (value: string): RuleValue =>
    isQuotedString(value)
        ? { matchType: 'EXACT', value: value.slice(1, -1) }
        : { matchType: 'REGEX', value };

// For label/annotation fields, equal-less values expand to two rule values
// so the backend queries both the key and value sides of the map field.
export function searchValueToMapRuleValues(value: string): RuleValue[] {
    const raw = isQuotedString(value) ? value.slice(1, -1) : value;
    const matchType = isQuotedString(value) ? 'EXACT' : 'REGEX';

    if (raw.includes('=')) {
        return [{ matchType, value: raw }];
    }

    return [
        { matchType: 'REGEX', value: `${raw}=.*` },
        { matchType: 'REGEX', value: `.*=${raw}` },
    ];
}

/**
 * Return initial entity scope rules for corresponding search fields
 * when user creates scheduled report configuration from results page.
 */
function getEntityScopeRulesFromSearchFilter(
    searchFilter: SearchFilter,
    searchFieldLabelMap: Record<string, EntityScopeRuleWithoutValues>
) {
    const rules: EntityScopeRule[] = [];

    Object.entries(searchFilter).forEach(([searchFieldLabel, searchFieldValue]) => {
        const ruleWithoutValues = getValueByCaseInsensitiveKey(
            searchFieldLabelMap,
            searchFieldLabel
        );
        const searchFieldValues = searchValueAsArray(searchFieldValue);

        if (ruleWithoutValues && searchFieldValues.length !== 0) {
            rules.push({
                ...ruleWithoutValues,
                values: searchFieldValues.flatMap((v) =>
                    isLabelOrAnnotationField(ruleWithoutValues.field)
                        ? searchValueToMapRuleValues(v)
                        : [searchValueToRuleValue(v)]
                ),
            });
        }
    });

    return rules;
}

export const ruleValueToSearchValue = ({ matchType, value }: RuleValue): string =>
    matchType === 'EXACT' ? wrapInQuotes(value) : value;

// Collapses complementary equal-less pairs ([`value=.*`, `.*=value`]) back to a single
// search value. Inverse of searchValueToMapRuleValues.
export function collapseMapRuleValues(ruleValues: RuleValue[]): string[] {
    const pairKeys = new Set(ruleValues.map(({ matchType, value }) => `${matchType}:${value}`));

    return ruleValues.flatMap(({ matchType, value }) => {
        if (value.startsWith('.*=')) {
            if (pairKeys.has(`${matchType}:${value.slice(3)}=.*`)) {
                return [];
            }
        }

        if (value.endsWith('=.*') && !value.startsWith('.*=')) {
            const raw = value.slice(0, -3);
            if (pairKeys.has(`${matchType}:.*=${raw}`)) {
                return [raw];
            }
        }

        return [ruleValueToSearchValue({ matchType, value })];
    });
}

/**
 * Return search filter in EntityScopeCompoundSearchFilter component.
 */
export function getSearchFilterFromEntityScopeRules(
    rules: EntityScopeRule[],
    searchFieldLabelMap: Record<string, EntityScopeRuleWithoutValues>
) {
    const searchFilter: SearchFilter = {};

    rules.forEach((rule) => {
        const found = Object.entries(searchFieldLabelMap).find(
            ([, { entity, field }]) => entity === rule.entity && field === rule.field
        );

        if (found) {
            const [searchFieldLabel] = found;
            const searchFilterValue = getValueByCaseInsensitiveKey(searchFilter, searchFieldLabel);
            const searchFilterValues = searchValueAsArray(searchFilterValue);
            const ruleValues = isLabelOrAnnotationField(rule.field)
                ? collapseMapRuleValues(rule.values)
                : rule.values.map(ruleValueToSearchValue);
            searchFilter[searchFieldLabel] = [...searchFilterValues, ...ruleValues];
        }
    });

    return searchFilter;
}

/**
 * Return initial entity scope rules for corresponding search fields
 * when user creates node vulnerability report configuration from results page.
 */
export function getEntityScopeRulesFromSearchFilterForCluster(searchFilter: SearchFilter) {
    return getEntityScopeRulesFromSearchFilter(searchFilter, searchFieldLabelMapForCluster);
}

export function getSearchFilterFromEntityScopeRulesForCluster(rules: EntityScopeRule[]) {
    return getSearchFilterFromEntityScopeRules(rules, searchFieldLabelMapForCluster);
}

/**
 * Return initial entity scope rules for corresponding search fields
 * when user creates either virtual machine vulnerability report configuration from results page.
 */
export function getEntityScopeRulesFromSearchFilterForClusterNamespace(searchFilter: SearchFilter) {
    return getEntityScopeRulesFromSearchFilter(
        searchFilter,
        searchFieldLabelMapForClusterNamespace
    );
}

export function getSearchFilterFromEntityScopeRulesForClusterNamespace(rules: EntityScopeRule[]) {
    return getSearchFilterFromEntityScopeRules(rules, searchFieldLabelMapForClusterNamespace);
}

/**
 * Return initial entity scope rules for corresponding search fields
 * when user creates either violation or image vulnerability report configuration from results page.
 */
export function getEntityScopeRulesFromSearchFilterForClusterNamespaceDeployment(
    searchFilter: SearchFilter
) {
    return getEntityScopeRulesFromSearchFilter(
        searchFilter,
        searchFieldLabelMapForClusterNamespaceDeployment
    );
}

export function getSearchFilterFromEntityScopeRulesForClusterNamespaceDeployment(
    rules: EntityScopeRule[]
) {
    return getSearchFilterFromEntityScopeRules(
        rules,
        searchFieldLabelMapForClusterNamespaceDeployment
    );
}
