import type {
    EntityScopeRule,
    MatchType,
    RuleValue,
    ScopeEntity,
    ScopeField,
} from 'services/ReportsService.types';

const entityTextMap: Record<ScopeEntity, string> = {
    SCOPE_ENTITY_UNSET: '',
    SCOPE_ENTITY_DEPLOYMENT: 'Deployment',
    SCOPE_ENTITY_NAMESPACE: 'Namespace',
    SCOPE_ENTITY_CLUSTER: 'Cluster',
} as const;

const fieldTextMap: Record<ScopeField, string> = {
    FIELD_UNSET: '',
    FIELD_ID: 'ID',
    FIELD_NAME: 'name',
    FIELD_LABEL: 'label',
    FIELD_ANNOTATION: 'annotation',
} as const;

export function ruleEntityFieldText({ entity, field }: EntityScopeRule) {
    const entityText = entityTextMap[entity] ?? '';
    const fieldText = fieldTextMap[field] ?? '';

    return entityText && fieldText ? `${entityText} ${fieldText}` : 'Resource or field unset';
}

const matchTypeSuffixMap: Record<MatchType, string> = {
    EXACT: ' (exact match)',
    REGEX: ' (regex match)',
} as const;

export function ruleValueText({ value, matchType }: RuleValue) {
    return `${value}${matchTypeSuffixMap[matchType] ?? ''}`;
}
