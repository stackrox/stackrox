import { CompoundSearchFilterConfig, SearchFilterAttribute, SearchFilterEntity } from '../types';

export function getEntities(config: Partial<CompoundSearchFilterConfig>): SearchFilterEntity[] {
    const entities = Object.keys(config) as SearchFilterEntity[];
    return entities;
}

export function getDefaultEntity(config: Partial<CompoundSearchFilterConfig>): SearchFilterEntity {
    const defaultEntity = Object.keys(config)[0] as SearchFilterEntity;
    return defaultEntity;
}

export function getEntityAttributes(
    entity: SearchFilterEntity,
    config: Partial<CompoundSearchFilterConfig>
): SearchFilterAttribute[] {
    if (config[entity] && config[entity]!.attributes) {
        const attributeValues = Object.values(config[entity]!.attributes);
        return attributeValues;
    }
    return [];
}

export function getDefaultAttribute(
    entity: SearchFilterEntity,
    config: Partial<CompoundSearchFilterConfig>
) {
    if (config[entity] && config[entity]!.attributes) {
        const attributeNames = Object.keys(config[entity]!.attributes);
        return attributeNames[0];
    }
    return '';
}
