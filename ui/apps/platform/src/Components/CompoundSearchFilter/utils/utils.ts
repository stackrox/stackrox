import {
    PartialCompoundSearchFilterConfig,
    SearchFilterAttribute,
    SearchFilterAttributeName,
    SearchFilterEntityName,
    compoundSearchEntityNames,
} from '../types';

export function getEntities(config: PartialCompoundSearchFilterConfig): SearchFilterEntityName[] {
    const entities = Object.keys(config) as SearchFilterEntityName[];
    return entities;
}

function isSearchFilterEntity(key: string): key is SearchFilterEntityName {
    return compoundSearchEntityNames.includes(key);
}

export function getDefaultEntity(
    config: PartialCompoundSearchFilterConfig
): SearchFilterEntityName | undefined {
    const entities = Object.keys(config).filter(isSearchFilterEntity);
    return entities[0];
}

export function getEntityAttributes(
    entity: SearchFilterEntityName,
    config: PartialCompoundSearchFilterConfig
): SearchFilterAttribute[] {
    const entityConfig = config[entity];
    if (entityConfig && entityConfig.attributes) {
        const attributeValues: SearchFilterAttribute[] = Object.values(entityConfig.attributes);
        return attributeValues;
    }
    return [];
}

export function getDefaultAttribute(
    entity: SearchFilterEntityName,
    config: PartialCompoundSearchFilterConfig
): SearchFilterAttributeName | undefined {
    const entityConfig = config[entity];
    if (entityConfig && entityConfig.attributes) {
        const attributeNames = Object.keys(entityConfig.attributes) as SearchFilterAttributeName[];
        return attributeNames[0];
    }
    return undefined;
}

export function ensureStringArray(value: unknown): string[] {
    if (Array.isArray(value) && value.every((element) => typeof element === 'string')) {
        return value as string[];
    }
    if (value === 'string') {
        return [value];
    }
    return [];
}

export function ensureString(value: unknown): string {
    if (typeof value === 'string') {
        return value;
    }
    return '';
}

export function ensureConditionNumber(value: unknown): { condition: string; number: number } {
    if (
        typeof value === 'object' &&
        value !== null &&
        'condition' in value &&
        'number' in value &&
        typeof value.condition === 'string' &&
        typeof value.number === 'number'
    ) {
        return {
            condition: value.condition,
            number: value.number,
        };
    }
    return {
        condition: '',
        number: 0,
    };
}
