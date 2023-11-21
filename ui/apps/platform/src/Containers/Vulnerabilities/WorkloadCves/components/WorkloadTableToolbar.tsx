import React from 'react';
import noop from 'lodash/noop';
import { Toolbar, ToolbarGroup, ToolbarContent, Flex } from '@patternfly/react-core';

import useURLSearch from 'hooks/useURLSearch';
import { SearchFilter } from 'types/search';
import { Globe } from 'react-feather';
import SearchFilterChips, { SearchFilterChipsProps } from 'Components/PatternFly/SearchFilterChips';
import useFeatureFlags from 'hooks/useFeatureFlags';
import { SearchOption, SearchOptionValue } from 'Containers/Vulnerabilities/searchOptions';
import { DefaultFilters } from '../types';
import FilterAutocomplete, {
    FilterAutocompleteSelectProps,
} from '../../components/FilterAutocomplete';
import CVESeverityDropdown from './CVESeverityDropdown';
import CVEStatusDropdown from './CVEStatusDropdown';

import './WorkloadTableToolbar.css';

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

type WorkloadTableToolbarProps = {
    defaultFilters?: DefaultFilters;
    searchOptions: SearchOption[];
    autocompleteSearchContext?: FilterAutocompleteSelectProps['autocompleteSearchContext'];
    onFilterChange?: (searchFilter: SearchFilter) => void;
};

function WorkloadTableToolbar({
    defaultFilters = emptyDefaultFilters,
    searchOptions,
    autocompleteSearchContext,
    onFilterChange = noop,
}: WorkloadTableToolbarProps) {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isFixabilityFiltersEnabled = isFeatureFlagEnabled('ROX_WORKLOAD_CVES_FIXABILITY_FILTERS');

    const { searchFilter, setSearchFilter } = useURLSearch();

    function onChangeSearchFilter(newFilter: SearchFilter) {
        setSearchFilter(newFilter);
        onFilterChange(newFilter);
    }

    function onSelect(
        type: Extract<SearchOptionValue, 'SEVERITY' | 'FIXABLE'>,
        checked: boolean,
        selection: string
    ) {
        const selectedSearchFilter = searchFilter[type] as string[];
        if (searchFilter[type]) {
            onChangeSearchFilter({
                ...searchFilter,
                [type]: checked
                    ? [...selectedSearchFilter, selection]
                    : selectedSearchFilter.filter((value) => value !== selection),
            });
        } else {
            onChangeSearchFilter({
                ...searchFilter,
                [type]: checked
                    ? [selection]
                    : selectedSearchFilter.filter((value) => value !== selection),
            });
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
        <Toolbar className="workload-table-toolbar">
            <ToolbarContent>
                <FilterAutocomplete
                    searchFilter={searchFilter}
                    setSearchFilter={setSearchFilter}
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

export default WorkloadTableToolbar;
