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

const searchFieldLabelMapForClusterNamespaceDeployment: Record<
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

export const searchFieldValueMapper = (value: string): RuleValue =>
    isQuotedString(value)
        ? { matchType: 'EXACT', value: value.slice(1, -1) }
        : { matchType: 'REGEX', value };

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
                values: searchFieldValues.map(searchFieldValueMapper),
            });
        }
    });

    return rules;
}

export const ruleFieldValueMapper = ({ matchType, value }: RuleValue): string =>
    matchType === 'EXACT' ? wrapInQuotes(value) : `r/${value}`;

/**
 * Return search filter in EntityScopeCompoundSearchFilter component.
 */
function getSearchFilterFromEntityScopeRules(
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
            const ruleValues = rule.values.map(ruleFieldValueMapper);
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
