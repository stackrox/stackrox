import type {
    EntityScopeRule,
    RuleValue,
    ScopeEntity,
    ScopeField,
} from 'services/ReportsService.types';
import type { SearchFilter } from 'types/search';
import type { SearchFieldLabel } from 'types/searchOptions';
import { isQuotedString, searchValueAsArray } from 'utils/searchUtils';

type EntityScopeSearchFieldLabelForCluster = Extract<
    SearchFieldLabel,
    'Cluster ID' | 'Cluster' | 'Cluster Label'
>;

type EntityScopeSearchFieldLabelForWorkload = Extract<
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

const searchFieldLabelMapForWorkload: Record<
    EntityScopeSearchFieldLabelForWorkload,
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
        const ruleWithoutValues = searchFieldLabelMap[searchFieldLabel];
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

/**
 * Return initial entity scope rules for corresponding search fields
 * when user creates node vulnerability report configuration from results page.
 */
export function getEntityScopeRulesFromSearchFilterForCluster(searchFilter: SearchFilter) {
    return getEntityScopeRulesFromSearchFilter(searchFilter, searchFieldLabelMapForCluster);
}

/**
 * Return initial entity scope rules for corresponding search fields
 * when user creates either violation or image vulnerability report configuration from results page.
 */
export function getEntityScopeRulesFromSearchFilterForCWorkload(searchFilter: SearchFilter) {
    return getEntityScopeRulesFromSearchFilter(searchFilter, searchFieldLabelMapForWorkload);
}
