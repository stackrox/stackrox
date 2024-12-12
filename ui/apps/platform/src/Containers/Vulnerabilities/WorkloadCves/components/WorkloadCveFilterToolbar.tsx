import React from 'react';
import noop from 'lodash/noop';
import { Toolbar, ToolbarGroup, ToolbarContent } from '@patternfly/react-core';
import useURLSearch from 'hooks/useURLSearch';
import { SearchFilter } from 'types/search';

import SearchFilterChips, {
    FilterChip,
    SearchFilterChipsProps,
} from 'Components/PatternFly/SearchFilterChips';
import useAnalytics, {
    WORKLOAD_CVE_FILTER_APPLIED,
    isSearchCategoryWithFilter,
} from 'hooks/useAnalytics';
import { searchValueAsArray } from 'utils/searchUtils';

import { SearchOption, SearchOptionValue } from '../../searchOptions';
import { DefaultFilters } from '../../types';
import FilterAutocomplete, {
    FilterAutocompleteSelectProps,
} from '../../components/FilterAutocomplete';
import CVESeverityDropdown from '../../components/CVESeverityDropdown';
import CVEStatusDropdown from '../../components/CVEStatusDropdown';

const emptyDefaultFilters = {
    SEVERITY: [],
    FIXABLE: [],
};

type WorkloadCveFilterToolbarProps = {
    defaultFilters?: DefaultFilters;
    searchOptions: SearchOption[];
    autocompleteSearchContext?: FilterAutocompleteSelectProps['autocompleteSearchContext'];
    onFilterChange?: (searchFilter: SearchFilter) => void;
    showCveFilterDropdowns?: boolean;
};

function WorkloadCveFilterToolbar({
    defaultFilters = emptyDefaultFilters,
    searchOptions,
    autocompleteSearchContext,
    onFilterChange = noop,
    showCveFilterDropdowns = true,
}: WorkloadCveFilterToolbarProps) {
    const { analyticsTrack } = useAnalytics();

    const { searchFilter, setSearchFilter } = useURLSearch();

    function trackAppliedFilter(category: SearchOptionValue, filter: string) {
        if (isSearchCategoryWithFilter(category)) {
            analyticsTrack({
                event: WORKLOAD_CVE_FILTER_APPLIED,
                properties: { category, filter },
            });
        } else {
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
        {
            displayName: 'CVE status',
            searchFilterName: 'FIXABLE',
            render: (filter: string) => (
                <FilterChip
                    isGlobal={defaultFilters.FIXABLE?.some((fixability) => fixability === filter)}
                    name={filter}
                />
            ),
        },
    ];

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
                {showCveFilterDropdowns && (
                    <ToolbarGroup>
                        <CVESeverityDropdown searchFilter={searchFilter} onSelect={onSelect} />
                        <CVEStatusDropdown
                            filterField="FIXABLE"
                            searchFilter={searchFilter}
                            onSelect={onSelect}
                        />
                    </ToolbarGroup>
                )}
                <ToolbarGroup aria-label="applied search filters" className="pf-v5-u-w-100">
                    <SearchFilterChips
                        searchFilter={searchFilter}
                        onFilterChange={setSearchFilter}
                        filterChipGroupDescriptors={filterChipGroupDescriptors}
                    />
                </ToolbarGroup>
            </ToolbarContent>
        </Toolbar>
    );
}

export default WorkloadCveFilterToolbar;
