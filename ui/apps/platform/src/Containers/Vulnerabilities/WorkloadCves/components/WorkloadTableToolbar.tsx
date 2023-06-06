import React, { useEffect } from 'react';
import noop from 'lodash/noop';
import { Toolbar, ToolbarGroup, ToolbarContent, ToolbarChip } from '@patternfly/react-core';

import useURLSearch from 'hooks/useURLSearch';
import { SearchFilter } from 'types/search';
import { DefaultFilters } from '../types';
import { Resource } from './FilterResourceDropdown';
import FilterAutocomplete, { FilterAutocompleteSelectProps } from './FilterAutocomplete';
import CVESeverityDropdown from './CVESeverityDropdown';
import FilterChips from './FilterChips';

const emptyDefaultFilters = {
    Severity: [],
    Fixable: [],
};

type FilterType = 'Severity' | 'Fixable';
type WorkloadTableToolbarProps = {
    defaultFilters?: DefaultFilters;
    supportedResourceFilters?: FilterAutocompleteSelectProps['supportedResourceFilters'];
    autocompleteSearchContext?: FilterAutocompleteSelectProps['autocompleteSearchContext'];
    onFilterChange?: (searchFilter: SearchFilter) => void;
};

function WorkloadTableToolbar({
    defaultFilters = emptyDefaultFilters,
    supportedResourceFilters,
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

    function onDelete(category: FilterType | Resource, chip: ToolbarChip | string) {
        const newSearchFilter = { ...searchFilter };
        const newResourceFilter = searchFilter[category] as string[];
        const chipKey = typeof chip === 'string' ? chip : chip.key;
        newSearchFilter[category] = newResourceFilter.filter((fil: string) => fil !== chipKey);
        onChangeSearchFilter(newSearchFilter);
    }

    function onDeleteGroup(category: FilterType | Resource) {
        const newSearchFilter = { ...searchFilter };
        delete newSearchFilter[category];
        onChangeSearchFilter(newSearchFilter);
    }

    function onDeleteAll() {
        onChangeSearchFilter({});
    }

    // The `onChangeSearchFilter` function is intentionally not used in place of `setSearchFilter` below since
    // it is intended to respond to a change via user action, and this useEffect is intended to sync the
    // state when the page loads or local storage changes.
    useEffect(() => {
        setSearchFilter(defaultFilters, 'replace');
        // unsure how to reset filters with URL filters only on defaultFilter change
    }, [defaultFilters, setSearchFilter]);

    return (
        <Toolbar id="workload-cves-table-toolbar">
            <ToolbarContent>
                <FilterAutocomplete
                    searchFilter={searchFilter}
                    setSearchFilter={setSearchFilter}
                    supportedResourceFilters={supportedResourceFilters}
                    onDeleteGroup={onDeleteGroup}
                    autocompleteSearchContext={autocompleteSearchContext}
                />
                <ToolbarGroup>
                    <CVESeverityDropdown searchFilter={searchFilter} onSelect={onSelect} />
                    {/* CVEStatusDropdown is disabled until fixability filters are fixed */}
                </ToolbarGroup>
                <ToolbarGroup className="pf-u-w-100">
                    <FilterChips
                        defaultFilters={defaultFilters}
                        searchFilter={searchFilter}
                        onDeleteGroup={onDeleteGroup}
                        onDelete={onDelete}
                        onDeleteAll={onDeleteAll}
                    />
                </ToolbarGroup>
            </ToolbarContent>
        </Toolbar>
    );
}

export default WorkloadTableToolbar;
