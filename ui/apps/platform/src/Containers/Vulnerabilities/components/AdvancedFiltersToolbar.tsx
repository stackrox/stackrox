import React from 'react';
import { Toolbar, ToolbarGroup, ToolbarContent, Flex } from '@patternfly/react-core';
import { uniq } from 'lodash';
import { Globe } from 'react-feather';

import CompoundSearchFilter, {
    CompoundSearchFilterProps,
} from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import { OnSearchPayload } from 'Components/CompoundSearchFilter/types';
import { makeFilterChipDescriptors } from 'Components/CompoundSearchFilter/utils/utils';
import SearchFilterChips from 'Components/PatternFly/SearchFilterChips';
import useFeatureFlags from 'hooks/useFeatureFlags';
import { SearchFilter } from 'types/search';
import { searchValueAsArray } from 'utils/searchUtils';

import { DefaultFilters } from '../types';
import CVESeverityDropdown from './CVESeverityDropdown';
import CVEStatusDropdown from './CVEStatusDropdown';

import './AdvancedFiltersToolbar.css';

type FilterChipProps = {
    isGlobal?: boolean;
    name: string;
};

function FilterChip({ isGlobal, name }: FilterChipProps) {
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

function makeDefaultFilterDescriptor(
    defaultFilters: DefaultFilters,
    { displayName, searchFilterName }: { displayName: string; searchFilterName: string }
) {
    return {
        displayName,
        searchFilterName,
        render: (filter: string) => (
            <FilterChip
                isGlobal={defaultFilters[searchFilterName]?.some((value) => value === filter)}
                name={filter}
            />
        ),
    };
}

const emptyDefaultFilters = {
    SEVERITY: [],
    FIXABLE: [],
};

type AdvancedFiltersToolbarProps = {
    searchFilterConfig: CompoundSearchFilterProps['config'];
    searchFilter: SearchFilter;
    onFilterChange: (searchFilter: SearchFilter, payload: OnSearchPayload) => void;
    className?: string;
    defaultFilters?: DefaultFilters;
    includeCveFilters?: boolean;
    // TODO We need to be able to apply the autocomplete search context to the advanced filters component @see FilterAutocomplete.tsx
    // autocompleteSearchContext?: unknown;
};

function AdvancedFiltersToolbar({
    searchFilterConfig,
    searchFilter,
    onFilterChange,
    className = '',
    defaultFilters = emptyDefaultFilters,
    includeCveFilters = true,
    // TODO We need to be able to apply the autocomplete search context to the advanced filters component
    // autocompleteSearchContext,
}: AdvancedFiltersToolbarProps) {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isFixabilityFiltersEnabled = isFeatureFlagEnabled('ROX_WORKLOAD_CVES_FIXABILITY_FILTERS');

    const filterChipGroupDescriptors = makeFilterChipDescriptors(searchFilterConfig)
        .concat(
            includeCveFilters
                ? makeDefaultFilterDescriptor(defaultFilters, {
                      displayName: 'CVE severity',
                      searchFilterName: 'SEVERITY',
                  })
                : []
        )
        .concat(
            includeCveFilters && isFixabilityFiltersEnabled
                ? makeDefaultFilterDescriptor(defaultFilters, {
                      displayName: 'CVE status',
                      searchFilterName: 'FIXABLE',
                  })
                : []
        );

    function onFilterApplied({ category, value, action }: OnSearchPayload) {
        const selectedSearchFilter = searchValueAsArray(searchFilter[category]);

        const newFilter = {
            ...searchFilter,
            [category]:
                action === 'ADD'
                    ? uniq([...selectedSearchFilter, value])
                    : selectedSearchFilter.filter((oldValue) => value !== oldValue),
        };
        onFilterChange(newFilter, { category, value, action });
    }

    return (
        <Toolbar className={`advanced-filters-toolbar ${className}`}>
            <ToolbarContent>
                <ToolbarGroup
                    variant="filter-group"
                    className="pf-v5-u-display-flex pf-v5-u-flex-grow-1"
                >
                    <CompoundSearchFilter
                        config={searchFilterConfig}
                        searchFilter={searchFilter}
                        onSearch={onFilterApplied}
                    />
                </ToolbarGroup>
                {includeCveFilters && (
                    <ToolbarGroup>
                        <CVESeverityDropdown
                            searchFilter={searchFilter}
                            onSelect={(category, checked, value) =>
                                onFilterApplied({
                                    category,
                                    value,
                                    action: checked ? 'ADD' : 'REMOVE',
                                })
                            }
                        />
                        {isFixabilityFiltersEnabled && (
                            <CVEStatusDropdown
                                searchFilter={searchFilter}
                                onSelect={(category, checked, value) =>
                                    onFilterApplied({
                                        category,
                                        value,
                                        action: checked ? 'ADD' : 'REMOVE',
                                    })
                                }
                            />
                        )}
                    </ToolbarGroup>
                )}
                <ToolbarGroup aria-label="applied search filters" className="pf-v5-u-w-100">
                    <SearchFilterChips filterChipGroupDescriptors={filterChipGroupDescriptors} />
                </ToolbarGroup>
            </ToolbarContent>
        </Toolbar>
    );
}

export default AdvancedFiltersToolbar;
