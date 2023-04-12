import React, { useEffect } from 'react';
import { Toolbar, ToolbarGroup, ToolbarContent, ToolbarChip } from '@patternfly/react-core';

import useURLSearch from 'hooks/useURLSearch';
import { uniq } from 'lodash';
import { DefaultFilters, VulnerabilitySeverityLabel, FixableStatus } from './types';
import { Resource } from './FilterResourceDropdown';
import FilterAutocomplete from './FilterAutocomplete';
import CVESeverityDropdown from './CVESeverityDropdown';
import CVEStatusDropdown from './CVEStatusDropdown';
import FilterChips from './components/FilterChips';

const emptyDefaultFilters = {
    Severity: [],
    Fixable: [],
};

type FilterType = 'Severity' | 'Fixable';
type WorkloadTableToolbarProps = {
    defaultFilters?: DefaultFilters;
    resourceContext?: Resource;
};

function WorkloadTableToolbar({
    defaultFilters = emptyDefaultFilters,
    resourceContext,
}: WorkloadTableToolbarProps) {
    const { searchFilter, setSearchFilter } = useURLSearch();

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
        if (
            searchFilter.Severity !== defaultFilters.Severity ||
            searchFilter.Fixable !== defaultFilters.Fixable
        ) {
            const searchSeverity = searchFilter.Severity as VulnerabilitySeverityLabel[];
            const searchFixable = searchFilter.Fixable as FixableStatus[];
            const { Severity: defaultSeverity, Fixable: defaultFixable } = defaultFilters;
            const severityFilter = searchSeverity
                ? uniq([...defaultSeverity, ...searchSeverity])
                : defaultSeverity;
            const fixableFilter = searchFixable
                ? uniq([...defaultFixable, ...searchFixable])
                : defaultFixable;
            setSearchFilter(
                {
                    ...defaultFilters,
                    ...searchFilter,
                    Severity: severityFilter,
                    Fixable: fixableFilter,
                },
                'replace'
            );
        }
    }, [defaultFilters, searchFilter, setSearchFilter]);

    return (
        <Toolbar id="workload-cves-table-toolbar">
            <ToolbarContent>
                <FilterAutocomplete
                    searchFilter={searchFilter}
                    setSearchFilter={setSearchFilter}
                    resourceContext={resourceContext}
                    onDeleteGroup={onDeleteGroup}
                />
                <ToolbarGroup>
                    <CVESeverityDropdown searchFilter={searchFilter} onSelect={onSelect} />
                    <CVEStatusDropdown searchFilter={searchFilter} onSelect={onSelect} />
                </ToolbarGroup>
                <ToolbarGroup>
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
