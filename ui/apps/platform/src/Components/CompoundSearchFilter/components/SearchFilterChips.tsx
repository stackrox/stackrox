import type { ReactElement, ReactNode } from 'react';
import { Button, Chip, ChipGroup, Flex, FlexItem } from '@patternfly/react-core';
import type { ToolbarChip } from '@patternfly/react-core';
import { Globe } from 'react-feather'; // eslint-disable-line limited/no-feather-icons

import type { SearchFilter } from 'types/search';
import { searchValueAsArray } from 'utils/searchUtils';

import type {
    CompoundSearchFilterAttribute,
    CompoundSearchFilterConfig,
    CompoundSearchFilterEntity,
} from '../types';
import { hasGroupedSelectOptions, isSelectType, updateSearchFilter } from '../utils/utils';

import { convertFromInternalToExternalConditionText } from './SearchFilterConditionText';

import './SearchFilterChips.css';

export type FilterChipProps = {
    isGlobal?: boolean;
    name: string;
};

export function FilterChip({ isGlobal, name }: FilterChipProps) {
    if (isGlobal) {
        return (
            <Flex alignItems={{ default: 'alignItemsCenter' }} flexWrap={{ default: 'nowrap' }}>
                <Globe height="15px" />
                {name}
            </Flex>
        );
    }
    return <Flex>{name}</Flex>;
}

export type FilterChipGroupDescriptor = {
    /** The name of the chip category that will be displayed in the toolbar */
    displayName: string;
    /** The name of the search filter category as it appears in the URL */
    searchFilterName: string;
    /** Optional render function for the chip. Defaults to rendering plain text inside a PatternFly `Chip` component */
    render?: (filter: string) => ReactNode;
};

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
            attributes.map(makeFilterChipDescriptorFromAttribute)
    );
    return filterChipDescriptors;
}

export function makeFilterChipDescriptorFromAttribute(
    attribute: CompoundSearchFilterAttribute
): FilterChipGroupDescriptor {
    const baseConfig = {
        displayName: attribute.filterChipLabel,
        searchFilterName: attribute.searchTerm,
    };

    if (isSelectType(attribute)) {
        const options = hasGroupedSelectOptions(attribute.inputProps)
            ? attribute.inputProps.groupOptions.flatMap((group) => group.options)
            : attribute.inputProps.options;
        return {
            ...baseConfig,
            render: (filter: string) => {
                const option = options.find((option) => option.value === filter);
                return <FilterChip name={option?.label || 'N/A'} />;
            },
        };
    }

    if (attribute.inputType === 'condition-text') {
        return {
            ...baseConfig,
            render: (filter: string) => {
                return (
                    <FilterChip
                        name={convertFromInternalToExternalConditionText(
                            attribute.inputProps,
                            filter
                        )}
                    />
                );
            },
        };
    }

    return baseConfig;
}

export type SearchFilterChipsProps = {
    /** The search filter categories to display */
    filterChipGroupDescriptors: FilterChipGroupDescriptor[];
    /** The current search filter */
    searchFilter: SearchFilter;
    /** Callback for when the search filter changes */
    onFilterChange: (searchFilter: SearchFilter) => void;
};

/**
 * Displays and manages the search filter chips for a given set of search filter categories based
 * on the current URL search filter
 */
function SearchFilterChips({
    filterChipGroupDescriptors,
    searchFilter,
    onFilterChange,
}: SearchFilterChipsProps): ReactElement {
    function onChangeSearchFilter(newFilter: SearchFilter) {
        onFilterChange(newFilter);
    }

    function onDelete(category: string, chip: ToolbarChip | string) {
        const value = typeof chip === 'string' ? chip : chip.key;
        onChangeSearchFilter(
            updateSearchFilter(searchFilter, [{ action: 'REMOVE', category, value }])
        );
    }

    function onDeleteGroup(category: string) {
        onChangeSearchFilter(updateSearchFilter(searchFilter, [{ action: 'DELETE', category }]));
    }

    function onDeleteAll() {
        onChangeSearchFilter({});
    }

    const hasSearchApplied = filterChipGroupDescriptors.some(({ searchFilterName }) => {
        const filters = searchValueAsArray(searchFilter[searchFilterName]);
        return filters.length > 0;
    });

    return (
        <Flex className="search-filter-chips" spaceItems={{ default: 'spaceItemsXs' }}>
            {filterChipGroupDescriptors.map(({ searchFilterName, displayName, render }) => {
                const filters = searchValueAsArray(searchFilter[searchFilterName]);
                if (!filters.length) {
                    return null;
                }
                return (
                    <FlexItem key={searchFilterName} className="pf-v5-u-pt-xs">
                        <ChipGroup
                            categoryName={displayName}
                            isClosable
                            onClick={() => onDeleteGroup(searchFilterName)}
                        >
                            {filters.map((filter) => (
                                <Chip
                                    closeBtnAriaLabel="Remove filter"
                                    key={filter}
                                    onClick={() => onDelete(searchFilterName, filter)}
                                >
                                    {render ? render(filter) : filter}
                                </Chip>
                            ))}
                        </ChipGroup>
                    </FlexItem>
                );
            })}
            {hasSearchApplied && (
                <Button variant="link" onClick={onDeleteAll}>
                    Clear filters
                </Button>
            )}
        </Flex>
    );
}

export default SearchFilterChips;
