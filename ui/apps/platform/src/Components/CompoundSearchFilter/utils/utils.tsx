import React from 'react';
import { FilterChip, FilterChipGroupDescriptor } from 'Components/PatternFly/SearchFilterChips';

import { SearchFilter } from 'types/search';
import { SetSearchFilter } from 'hooks/useURLSearch';
import {
    OnSearchPayload,
    PartialCompoundSearchFilterConfig,
    SearchFilterAttribute,
    SearchFilterAttributeName,
    SearchFilterEntityName,
    SelectSearchFilterAttribute,
    compoundSearchEntityNames,
} from '../types';

export const conditionMap = {
    'Is greater than': '>',
    'Is greater than or equal to': '>=',
    'Is equal to': '=',
    'Is less than or equal to': '<=',
    'Is less than': '<',
};

export const conditions = Object.keys(conditionMap);

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
export function makeFilterChipDescriptors(
    searchFilterConfig: PartialCompoundSearchFilterConfig
): FilterChipGroupDescriptor[] {
    const filterChipDescriptors = Object.values(searchFilterConfig).flatMap(({ attributes = {} }) =>
        Object.values(attributes).map((attributeConfig: SearchFilterAttribute) => {
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
