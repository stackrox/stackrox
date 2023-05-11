import React, { useEffect } from 'react';
import { Toolbar, ToolbarGroup, ToolbarContent, ToolbarChip } from '@patternfly/react-core';

import useURLSearch from 'hooks/useURLSearch';
import { uniq } from 'lodash';
import { DefaultFilters, VulnerabilitySeverityLabel, FixableStatus } from '../types';
import { Resource } from './FilterResourceDropdown';
import FilterAutocomplete, { FilterAutocompleteSelectProps } from './FilterAutocomplete';
import CVESeverityDropdown from './CVESeverityDropdown';
import CVEStatusDropdown from './CVEStatusDropdown';
import FilterChips from './FilterChips';

const emptyDefaultFilters = {
    Severity: [],
    Fixable: [],
};

type FilterType = 'Severity' | 'Fixable';
type WorkloadTableToolbarProps = {
    defaultFilters?: DefaultFilters;
    supportedResourceFilters?: FilterAutocompleteSelectProps['supportedResourceFilters'];
};

function WorkloadTableToolbar({
    defaultFilters = emptyDefaultFilters,
    supportedResourceFilters,
}: WorkloadTableToolbarProps) {
    const { searchFilter, setSearchFilter } = useURLSearch();
    const searchSeverity = (searchFilter.Severity as VulnerabilitySeverityLabel[]) || [];
    const searchFixable = (searchFilter.Fixable as FixableStatus[]) || [];
    const { Severity: defaultSeverity, Fixable: defaultFixable } = defaultFilters;

    function onSelect(type: FilterType, e, selection) {
        const { checked } = e.target as HTMLInputElement;
        const selectedSearchFilter = searchFilter[type] as string[];
        if (searchFilter[type]) {
            setSearchFilter({
                ...searchFilter,
                [type]: checked
                    ? [...selectedSearchFilter, selection]
                    : selectedSearchFilter.filter((value) => value !== selection),
            });
        } else {
            setSearchFilter({
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
        setSearchFilter(newSearchFilter);
    }

    function onDeleteGroup(category: FilterType | Resource) {
        const newSearchFilter = { ...searchFilter };
        delete newSearchFilter[category];
        setSearchFilter(newSearchFilter);
    }

    function onDeleteAll() {
        setSearchFilter({});
    }

    useEffect(() => {
        const severityFilter = uniq([...defaultSeverity, ...searchSeverity]);
        const fixableFilter = uniq([...defaultFixable, ...searchFixable]);
        setSearchFilter(
            {
                ...defaultFilters,
                ...searchFilter,
                Severity: severityFilter,
                Fixable: fixableFilter,
            },
            'replace'
        );
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
                />
                <ToolbarGroup>
                    <CVESeverityDropdown searchFilter={searchFilter} onSelect={onSelect} />
                    <CVEStatusDropdown searchFilter={searchFilter} onSelect={onSelect} />
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
