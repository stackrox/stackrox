import {
    CompoundSearchFilterConfig,
    SearchFilterAttribute,
    SearchFilterEntityName,
} from '../types';

export function getEntities(config: Partial<CompoundSearchFilterConfig>): SearchFilterEntityName[] {
    const entities = Object.keys(config) as SearchFilterEntityName[];
    return entities;
}

export function getDefaultEntity(
    config: Partial<CompoundSearchFilterConfig>
): SearchFilterEntityName {
    const defaultEntity = Object.keys(config)[0] as SearchFilterEntityName;
    return defaultEntity;
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
) {
    const entityConfig = config[entity];
    if (entityConfig && entityConfig.attributes) {
        const attributeNames = Object.keys(entityConfig.attributes);
        return attributeNames[0];
    }
    return '';
}
