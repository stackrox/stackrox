import {
    CompoundSearchFilterConfig,
    SearchFilterAttribute,
    SearchFilterAttributeName,
    SearchFilterEntityName,
    compoundSearchEntityNames,
} from '../types';

export function getEntities(config: Partial<CompoundSearchFilterConfig>): SearchFilterEntityName[] {
    const entities = Object.keys(config) as SearchFilterEntityName[];
    return entities;
}

function isSearchFilterEntity(key: string): key is SearchFilterEntityName {
    return compoundSearchEntityNames.includes(key);
}

export function getDefaultEntity(
    config: Partial<CompoundSearchFilterConfig>
): SearchFilterEntityName | undefined {
    const entities = Object.keys(config).filter(isSearchFilterEntity);
    return entities[0];
}

export function getEntityAttributes(
    entity: SearchFilterEntityName,
    config: Partial<CompoundSearchFilterConfig>
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
    config: Partial<CompoundSearchFilterConfig>
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
    return [];
}

export function ensureString(value: unknown): string {
    if (typeof value === 'string') {
        return value;
    }
    return '';
}
