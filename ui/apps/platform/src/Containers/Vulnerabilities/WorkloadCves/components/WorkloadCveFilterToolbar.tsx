import React from 'react';
import noop from 'lodash/noop';
import { Toolbar, ToolbarGroup, ToolbarContent, Flex } from '@patternfly/react-core';
import useURLSearch from 'hooks/useURLSearch';
import { SearchFilter } from 'types/search';
import { Globe } from 'react-feather';

import SearchFilterChips, { SearchFilterChipsProps } from 'Components/PatternFly/SearchFilterChips';
import useFeatureFlags from 'hooks/useFeatureFlags';
import useAnalytics, {
    WORKLOAD_CVE_FILTER_APPLIED,
    isSearchCategoryWithFilter,
    isSearchCategoryWithoutFilter,
} from 'hooks/useAnalytics';
import { searchValueAsArray } from 'utils/searchUtils';

import { SearchOption, SearchOptionValue } from '../../searchOptions';
import { DefaultFilters } from '../../types';
import FilterAutocomplete, {
    FilterAutocompleteSelectProps,
} from '../../components/FilterAutocomplete';
import CVESeverityDropdown from '../../components/CVESeverityDropdown';
import CVEStatusDropdown from '../../components/CVEStatusDropdown';

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

const emptyDefaultFilters = {
    SEVERITY: [],
    FIXABLE: [],
};

type WorkloadCveFilterToolbarProps = {
    defaultFilters?: DefaultFilters;
    searchOptions: SearchOption[];
    autocompleteSearchContext?: FilterAutocompleteSelectProps['autocompleteSearchContext'];
    onFilterChange?: (searchFilter: SearchFilter) => void;
};

function WorkloadCveFilterToolbar({
    defaultFilters = emptyDefaultFilters,
    searchOptions,
    autocompleteSearchContext,
    onFilterChange = noop,
}: WorkloadCveFilterToolbarProps) {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isFixabilityFiltersEnabled = isFeatureFlagEnabled('ROX_WORKLOAD_CVES_FIXABILITY_FILTERS');
    const { analyticsTrack } = useAnalytics();

    const { searchFilter, setSearchFilter } = useURLSearch();

    function trackAppliedFilter(category: SearchOptionValue, filter: string) {
        if (isSearchCategoryWithFilter(category)) {
            analyticsTrack({
                event: WORKLOAD_CVE_FILTER_APPLIED,
                properties: { category, filter },
            });
        } else if (isSearchCategoryWithoutFilter(category)) {
            analyticsTrack({
                event: WORKLOAD_CVE_FILTER_APPLIED,
                properties: { category },
            });
        }
    }

    function onChangeSearchFilter(newFilter: SearchFilter) {
        setSearchFilter(newFilter);
        onFilterChange(newFilter);
    }

    function onSelect(
        type: Extract<SearchOptionValue, 'SEVERITY' | 'FIXABLE'>,
        checked: boolean,
        selection: string
    ) {
        const selectedSearchFilter = searchValueAsArray(searchFilter[type]);
        onChangeSearchFilter({
            ...searchFilter,
            [type]: checked
                ? [...selectedSearchFilter, selection]
                : selectedSearchFilter.filter((value) => value !== selection),
        });

        if (checked) {
            trackAppliedFilter(type, selection);
        }
    }

    const filterChipGroupDescriptors: (SearchFilterChipsProps['filterChipGroupDescriptors'][number] & {
        searchFilterName: SearchOptionValue;
    })[] = [
        {
            displayName: 'Deployment',
            searchFilterName: 'DEPLOYMENT',
        },
        {
            displayName: 'CVE',
            searchFilterName: 'CVE',
        },
        {
            displayName: 'Image',
            searchFilterName: 'IMAGE',
        },
        {
            displayName: 'Namespace',
            searchFilterName: 'NAMESPACE',
        },
        {
            displayName: 'Cluster',
            searchFilterName: 'CLUSTER',
        },
        {
            displayName: 'Component',
            searchFilterName: 'COMPONENT',
        },
        {
            displayName: 'Component Source',
            searchFilterName: 'COMPONENT SOURCE',
        },
        {
            displayName: 'Severity',
            searchFilterName: 'SEVERITY',
            render: (filter: string) => (
                <FilterChip
                    isGlobal={defaultFilters.SEVERITY?.some((severity) => severity === filter)}
                    name={filter}
                />
            ),
        },
    ];

    if (isFixabilityFiltersEnabled) {
        filterChipGroupDescriptors.push({
            displayName: 'Fixable',
            searchFilterName: 'FIXABLE',
            render: (filter: string) => (
                <FilterChip
                    isGlobal={defaultFilters.FIXABLE?.some((fixability) => fixability === filter)}
                    name={filter}
                />
            ),
        });
    }

    return (
        <Toolbar>
            <ToolbarContent>
                <FilterAutocomplete
                    searchFilter={searchFilter}
                    onFilterChange={(newFilter, { action, category, value }) => {
                        setSearchFilter(newFilter);
                        if (action === 'ADD') {
                            trackAppliedFilter(category, value);
                        }
                    }}
                    searchOptions={searchOptions}
                    autocompleteSearchContext={autocompleteSearchContext}
                />
                <ToolbarGroup>
                    <CVESeverityDropdown searchFilter={searchFilter} onSelect={onSelect} />
                    {isFixabilityFiltersEnabled && (
                        <CVEStatusDropdown searchFilter={searchFilter} onSelect={onSelect} />
                    )}
                </ToolbarGroup>
                <ToolbarGroup aria-label="applied search filters" className="pf-u-w-100">
                    <SearchFilterChips
                        onFilterChange={onFilterChange}
                        filterChipGroupDescriptors={filterChipGroupDescriptors}
                    />
                </ToolbarGroup>
            </ToolbarContent>
        </Toolbar>
    );
}

export default WorkloadCveFilterToolbar;
