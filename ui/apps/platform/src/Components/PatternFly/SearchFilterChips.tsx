import React from 'react';
import { Button, ChipGroup, Chip, Flex, FlexItem, ToolbarChip } from '@patternfly/react-core';
import noop from 'lodash/noop';

import useURLSearch from 'hooks/useURLSearch';
import { SearchFilter } from 'types/search';
import { searchValueAsArray } from 'utils/searchUtils';

import './SearchFilterChips.css';

export type FilterChipGroupDescriptor = {
    /** The name of the chip category that will be displayed in the toolbar */
    displayName: string;
    /** The name of the search filter category as it appears in the URL */
    searchFilterName: string;
    /** Optional render function for the chip. Defaults to rendering plain text inside a PatternFly `Chip` component */
    render?: (filter: string) => React.ReactElement;
};

export type SearchFilterChipsProps = {
    /** The search filter categories to display */
    filterChipGroupDescriptors: FilterChipGroupDescriptor[];
    /** Callback for when the search filter changes */
    onFilterChange?: (searchFilter: SearchFilter) => void;
};

/**
 * Displays and manages the search filter chips for a given set of search filter categories based
 * on the current URL search filter
 */
function SearchFilterChips({
    filterChipGroupDescriptors,
    onFilterChange = noop,
}: SearchFilterChipsProps) {
    const { searchFilter, setSearchFilter } = useURLSearch();

    function onChangeSearchFilter(newFilter: SearchFilter) {
        setSearchFilter(newFilter);
        onFilterChange(newFilter);
    }

    function onDelete(category: string, chip: ToolbarChip | string) {
        const newSearchFilter = { ...searchFilter };
        const newSearchFilterValues = searchValueAsArray(searchFilter[category]);
        const chipKey = typeof chip === 'string' ? chip : chip.key;
        newSearchFilter[category] = newSearchFilterValues.filter((fil: string) => fil !== chipKey);
        onChangeSearchFilter(newSearchFilter);
    }

    function onDeleteGroup(category: string) {
        const newSearchFilter = { ...searchFilter };
        delete newSearchFilter[category];
        onChangeSearchFilter(newSearchFilter);
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
                    <FlexItem key={searchFilterName} className="pf-u-pt-xs">
                        <ChipGroup
                            categoryName={displayName}
                            isClosable
                            onClick={() => onDeleteGroup(searchFilterName)}
                        >
                            {filters.map((filter) => (
                                <Chip
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
