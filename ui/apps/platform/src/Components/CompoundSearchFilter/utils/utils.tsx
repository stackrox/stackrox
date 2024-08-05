import React from 'react';

import { FilterChip, FilterChipGroupDescriptor } from 'Components/PatternFly/SearchFilterChips';
import { SearchFilter } from 'types/search';
import { SetSearchFilter } from 'hooks/useURLSearch';
import {
    CompoundSearchFilterAttribute,
    CompoundSearchFilterConfig,
    CompoundSearchFilterEntity,
    OnSearchPayload,
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

export function getEntity(
    config: CompoundSearchFilterConfig,
    entityName: string
): CompoundSearchFilterEntity | undefined {
    if (!config || !Array.isArray(config)) {
        return undefined;
    }
    const entity = config.find((entity) => {
        return entity.displayName === entityName;
    });
    return entity;
}

export function getAttribute(
    config: CompoundSearchFilterConfig,
    entityName: string,
    attributeName: string
): CompoundSearchFilterAttribute | undefined {
    const entity = getEntity(config, entityName);
    return entity?.attributes?.find((attribute) => {
        return attribute.displayName === attributeName;
    });
}

export function getDefaultEntityName(config: CompoundSearchFilterConfig): string | undefined {
    if (!config || !Array.isArray(config)) {
        return undefined;
    }
    return config?.[0]?.displayName;
}

export function getEntityAttributes(
    config: CompoundSearchFilterConfig,
    entityName: string
): CompoundSearchFilterAttribute[] {
    const entity = getEntity(config, entityName);
    return entity?.attributes || [];
}

export function getDefaultAttributeName(
    config: CompoundSearchFilterConfig,
    entityName: string
): string | undefined {
    const attributes = getEntityAttributes(config, entityName);
    return attributes?.[0]?.displayName;
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
    attribute: CompoundSearchFilterAttribute
): attribute is SelectSearchFilterAttribute {
    return attribute.inputType === 'select';
}

/**
 * Helper function to convert a search filter config object into an
 * array of FilterChipGroupDescriptor objects for use in the SearchFilterChips component
 *
 * @param searchFilterConfig Config object for the search filter
 * @returns An array of FilterChipGroupDescriptor objects
 */
export function makeFilterChipDescriptors(
    config: CompoundSearchFilterConfig
): FilterChipGroupDescriptor[] {
    const filterChipDescriptors = config.flatMap(
        ({ attributes = [] }: CompoundSearchFilterEntity) =>
            attributes.map((attribute) => {
                const baseConfig = {
                    displayName: attribute.filterChipLabel,
                    searchFilterName: attribute.searchTerm,
                };

                if (isSelectType(attribute)) {
                    return {
                        ...baseConfig,
                        render: (filter: string) => {
                            const option = attribute.inputProps.options.find(
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
