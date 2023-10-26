import React from 'react';
import noop from 'lodash/noop';
import { Toolbar, ToolbarGroup, ToolbarContent, Flex } from '@patternfly/react-core';

import useURLSearch from 'hooks/useURLSearch';
import { SearchFilter } from 'types/search';
import { Globe } from 'react-feather';
import SearchFilterChips from 'Components/PatternFly/SearchFilterChips';
import { SearchOption } from 'Containers/Vulnerabilities/components/SearchOptionsDropdown';
import { DefaultFilters, VulnerabilitySeverityLabel } from '../types';
import FilterAutocomplete, {
    FilterAutocompleteSelectProps,
} from '../../components/FilterAutocomplete';
import CVESeverityDropdown from './CVESeverityDropdown';

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
    Severity: [],
    Fixable: [],
};

type FilterType = 'Severity' | 'Fixable';
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
    const { searchFilter, setSearchFilter } = useURLSearch();

    function onChangeSearchFilter(newFilter: SearchFilter) {
        setSearchFilter(newFilter);
        onFilterChange(newFilter);
    }

    function onSelect(type: FilterType, e, selection) {
        const { checked } = e.target as HTMLInputElement;
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

    return (
        <Toolbar id="workload-cves-table-toolbar">
            <ToolbarContent>
                <FilterAutocomplete
                    searchFilter={searchFilter}
                    setSearchFilter={setSearchFilter}
                    searchOptions={searchOptions}
                    autocompleteSearchContext={autocompleteSearchContext}
                />
                <ToolbarGroup>
                    <CVESeverityDropdown searchFilter={searchFilter} onSelect={onSelect} />
                    {/* CVEStatusDropdown is disabled until fixability filters are fixed */}
                </ToolbarGroup>
                <ToolbarGroup aria-label="applied search filters" className="pf-u-w-100">
                    <SearchFilterChips
                        onFilterChange={onFilterChange}
                        filterChipGroupDescriptors={[
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
                                displayName: 'Severity',
                                searchFilterName: 'Severity',
                                render: (filter: string) => (
                                    <FilterChip
                                        isGlobal={defaultFilters.Severity?.includes(
                                            filter as VulnerabilitySeverityLabel
                                        )}
                                        name={filter}
                                    />
                                ),
                            },
                        ]}
                    />
                </ToolbarGroup>
            </ToolbarContent>
        </Toolbar>
    );
}

export default WorkloadTableToolbar;
