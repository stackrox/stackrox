import type { ReactElement, ReactNode } from 'react';
import { Button, ChipGroup, Chip, Flex, FlexItem } from '@patternfly/react-core';
import type { ToolbarChip } from '@patternfly/react-core';
import { Globe } from 'react-feather';

import type { SearchFilter } from 'types/search';
import { searchValueAsArray } from 'utils/searchUtils';

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
