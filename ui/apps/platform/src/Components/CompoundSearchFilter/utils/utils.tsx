import React from 'react';
import pick from 'lodash/pick';

import { FilterChip, FilterChipGroupDescriptor } from 'Components/PatternFly/SearchFilterChips';
import { SearchFilter } from 'types/search';
import { SetSearchFilter } from 'hooks/useURLSearch';
import {
    OnSearchPayload,
    SearchFilterAttribute,
    SearchFilterConfig,
    SelectSearchFilterAttribute,
} from '../types';

export const conditionMap = {
    'Is greater than': '>',
    'Is greater than or equal to': '>=',
    'Is equal to': '=',
    'Is less than or equal to': '<=',
    'Is less than': '<',
};

export const conditions = Object.keys(conditionMap);

export function getEntityConfig<T extends Record<string, SearchFilterConfig>>(
    config: T,
    entity: string
): SearchFilterConfig | undefined {
    const entityConfig = Object.values(config).find((entityConfig) => {
        return entityConfig.displayName === entity;
    });
    return entityConfig;
}

export function getAttributeConfig<T extends Record<string, SearchFilterConfig>>(
    config: T,
    entity: string,
    attribute: string
): SearchFilterAttribute | undefined {
    const entityConfig = getEntityConfig(config, entity);
    if (entityConfig && entityConfig.attributes) {
        return Object.values(entityConfig.attributes).find((attributeConfig) => {
            return attributeConfig.displayName === attribute;
        });
    }
    return undefined;
}

export function getEntities<T extends Record<string, SearchFilterConfig>>(config: T): (keyof T)[] {
    const entities = Object.values(config).map((entityConfig) => {
        const { displayName } = entityConfig;
        return displayName;
    });
    return entities;
}

export function getDefaultEntity<T extends Record<string, SearchFilterConfig>>(config: T): keyof T {
    const entities = getEntities(config);
    return entities[0];
}

export function getEntityAttributes<T extends Record<string, SearchFilterConfig>>(
    entity: string,
    config: T
): SearchFilterAttribute[] {
    const entityConfig = getEntityConfig(config, entity);
    if (entityConfig && entityConfig.attributes) {
        const attributeValues = Object.values(entityConfig.attributes);
        return attributeValues;
    }
    return [];
}

export function getDefaultAttribute<T extends Record<string, SearchFilterConfig>>(
    entity: string,
    config: T
): string | undefined {
    const entityConfig = getEntityConfig(config, entity);
    if (entityConfig && entityConfig.attributes) {
        const attributeNames = Object.values(entityConfig.attributes).map((attributeConfig) => {
            return attributeConfig.displayName;
        });
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
        condition: conditions[0],
        number: 0,
    };
}

export function isSelectType(
    attributeObject: SearchFilterAttribute
): attributeObject is SelectSearchFilterAttribute {
    return attributeObject.inputType === 'select';
}

/**
 * Helper function to convert a search filter config object into an
 * array of FilterChipGroupDescriptor objects for use in the SearchFilterChips component
 *
 * @param searchFilterConfig Config object for the search filter
 * @returns An array of FilterChipGroupDescriptor objects
 */
export function makeFilterChipDescriptors<T extends object>(
    searchFilterConfig: T
): FilterChipGroupDescriptor[] {
    const filterChipDescriptors = Object.values(searchFilterConfig).flatMap(
        ({ attributes = {} }: SearchFilterConfig) =>
            Object.values(attributes).map((attributeConfig) => {
                const baseConfig = {
                    displayName: attributeConfig.filterChipLabel,
                    searchFilterName: attributeConfig.searchTerm,
                };

                if (isSelectType(attributeConfig)) {
                    return {
                        ...baseConfig,
                        render: (filter: string) => {
                            const option = attributeConfig.inputProps.options.find(
                                (option) => option.value === filter
                            );
                            return <FilterChip name={option?.label || 'N/A'} />;
                        },
                    };
                }

                return baseConfig;
            })
    );
    return filterChipDescriptors;
}

// Function to take a compound search "onSearch" payload and update the URL
export const onURLSearch = (
    searchFilter: SearchFilter,
    setSearchFilter: SetSearchFilter,
    payload: OnSearchPayload
) => {
    const { action, category, value } = payload;
    const currentSelection = searchFilter[category] || [];
    let newSelection = !Array.isArray(currentSelection) ? [currentSelection] : currentSelection;
    if (action === 'ADD') {
        newSelection = [...newSelection, value];
    } else if (action === 'REMOVE') {
        newSelection = newSelection.filter((datum) => datum !== value);
    } else {
        // Do nothing
    }
    setSearchFilter({
        ...searchFilter,
        [category]: newSelection,
    });
};

export function pickAttributes<T extends Record<string, SearchFilterAttribute>>(
    attrs: T,
    keys: string[]
) {
    return pick(attrs, keys);
}
