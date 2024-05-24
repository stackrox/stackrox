import React from 'react';
import { Button, Chip, ChipGroup, Flex, FlexItem, ToolbarChip } from '@patternfly/react-core';

import { UseUrlSearchReturn } from 'hooks/useURLSearch';
import { SearchFilter } from 'types/search';
import { searchValueAsArray } from 'utils/searchUtils';
import { SearchFilterAttribute, compoundSearchFilter } from '../types';

export type SearchFilterChipsProps = {
    searchFilter: UseUrlSearchReturn['searchFilter'];
    setSearchFilter: UseUrlSearchReturn['setSearchFilter'];
};

function getFilterChipLabelMap(): Record<string, string> {
    const map = {};
    Object.values(compoundSearchFilter).forEach((entityObject) => {
        const { attributes } = entityObject;
        Object.values(attributes).forEach((attributeObject: SearchFilterAttribute) => {
            const { filterChipLabel, searchTerm } = attributeObject;
            map[searchTerm] = filterChipLabel;
        });
    });
    return map;
}

const filterChipLabelMap = getFilterChipLabelMap();

function SearchFilterChips({ searchFilter, setSearchFilter }: SearchFilterChipsProps) {
    function onChangeSearchFilter(newFilter: SearchFilter) {
        setSearchFilter(newFilter);
        // onFilterChange(newFilter);
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

    return (
        <Flex direction={{ default: 'row' }} spaceItems={{ default: 'spaceItemsMd' }}>
            {Object.keys(searchFilter).map((searchKey) => {
                const searchCategoryName = filterChipLabelMap[searchKey] || searchKey;
                const searchValue = searchFilter[searchKey];
                const filters = searchValueAsArray(searchValue);
                if (!filters.length) {
                    return null;
                }
                return (
                    <FlexItem>
                        <ChipGroup
                            categoryName={searchCategoryName}
                            isClosable
                            onClick={() => onDeleteGroup(searchKey)}
                        >
                            {filters.map((filter) => (
                                <Chip
                                    closeBtnAriaLabel="Remove filter"
                                    key={filter}
                                    onClick={() => onDelete(searchKey, filter)}
                                >
                                    {filter}
                                </Chip>
                            ))}
                        </ChipGroup>
                    </FlexItem>
                );
            })}
            {Object.keys(searchFilter).length !== 0 && (
                <Button variant="link" onClick={onDeleteAll}>
                    Clear filters
                </Button>
            )}
        </Flex>
    );
}

export default SearchFilterChips;
