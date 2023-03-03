import React, { useEffect } from 'react';
import {
    Toolbar,
    ToolbarItem,
    ToolbarFilter,
    ToolbarToggleGroup,
    ToolbarGroup,
    ToolbarContent,
} from '@patternfly/react-core';
import { FilterIcon } from '@patternfly/react-icons';

import useURLSearch from 'hooks/useURLSearch';
import { uniq } from 'lodash';
import { DefaultFilters } from './types';
import FilterResourceDropdown, { Resource } from './FilterResourceDropdown';
import FilterAutocompleteInput from './FilterAutocompleteInput';
import CVESeverityDropdown from './CVESeverityDropdown';
import CVEStatusDropdown from './CVEStatusDropdown';

type FilterType = 'resource' | 'Severity' | 'Fixable';

type WorkloadTableToolbarProps = {
    defaultFilters: DefaultFilters;
    resourceContext?: Resource;
};

function WorkloadTableToolbar({ defaultFilters, resourceContext }: WorkloadTableToolbarProps) {
    const { searchFilter, setSearchFilter } = useURLSearch();

    function onSelect(type: FilterType, e, selection) {
        if (type === 'resource') {
            setSearchFilter({
                ...searchFilter,
                resource: selection,
            });
        } else {
            const { checked } = e.target as HTMLInputElement;
            if (searchFilter[type]) {
                setSearchFilter({
                    ...searchFilter,
                    [type]: checked
                        ? [...searchFilter[type], selection]
                        : searchFilter[type]?.filter((value) => value !== selection),
                });
            } else {
                setSearchFilter({
                    ...searchFilter,
                    [type]: checked
                        ? [selection]
                        : searchFilter[type]?.filter((value) => value !== selection),
                });
            }
        }
    }

    function onDelete(type: FilterType, id: string) {
        if (type === 'Severity') {
            setSearchFilter({
                ...searchFilter,
                Severity: searchFilter.Severity?.filter((fil: string) => fil !== id),
            });
        } else if (type === 'Fixable') {
            setSearchFilter({
                ...searchFilter,
                Fixable: searchFilter.Fixable?.filter((fil: string) => fil !== id),
            });
        }
    }

    function onDeleteGroup(type: FilterType) {
        if (type === 'Severity') {
            const { Severity, ...remainingSearchFilter } = searchFilter;
            setSearchFilter({
                ...remainingSearchFilter,
            });
        } else if (type === 'Fixable') {
            const { Fixable, ...remainingSearchFilter } = searchFilter;
            setSearchFilter({
                ...remainingSearchFilter,
            });
        }
    }

    function onDeleteAll() {
        const { Severity, Fixable, ...remainingSearchFilter } = searchFilter;
        setSearchFilter({
            ...remainingSearchFilter,
        });
    }

    useEffect(() => {
        const { Severity: searchSeverity, Fixable: searchFixable } = searchFilter;
        const { Severity: defaultSeverity, Fixable: defaultFixable } = defaultFilters;
        const severityFilter = searchSeverity
            ? uniq([...defaultSeverity, ...searchSeverity])
            : defaultSeverity;
        const fixableFilter = searchFixable
            ? uniq([...defaultFixable, ...searchFixable])
            : defaultFixable;
        setSearchFilter({
            ...defaultFilters,
            ...searchFilter,
            Severity: severityFilter,
            Fixable: fixableFilter,
        });
    }, [defaultFilters, setSearchFilter]);

    useEffect(() => {
        if (!searchFilter.resource) {
            setSearchFilter({ ...searchFilter, resource: 'CVE' });
        }
    }, []);

    return (
        <Toolbar
            id="workload-cves-table-toolbar"
            collapseListedFiltersBreakpoint="xl"
            clearAllFilters={onDeleteAll}
        >
            <ToolbarContent>
                <ToolbarToggleGroup
                    toggleIcon={<FilterIcon />}
                    breakpoint="xl"
                    className="pf-u-w-100"
                >
                    <ToolbarGroup variant="filter-group" className="pf-u-w-100">
                        <ToolbarItem>
                            <FilterResourceDropdown
                                onSelect={onSelect}
                                searchFilter={searchFilter}
                                resourceContext={resourceContext}
                            />
                        </ToolbarItem>
                        <ToolbarItem variant="search-filter" className="pf-u-w-100">
                            <FilterAutocompleteInput
                                searchFilter={searchFilter}
                                setSearchFilter={setSearchFilter}
                            />
                        </ToolbarItem>
                    </ToolbarGroup>
                    <ToolbarGroup>
                        <ToolbarFilter
                            chips={searchFilter.Severity as string[]}
                            deleteChip={(category, chip) =>
                                onDelete(category as FilterType, chip as string)
                            }
                            deleteChipGroup={(category) => onDeleteGroup(category as FilterType)}
                            categoryName="Severity"
                        >
                            <CVESeverityDropdown searchFilter={searchFilter} onSelect={onSelect} />
                        </ToolbarFilter>
                        <ToolbarFilter
                            chips={searchFilter.Fixable as string[]}
                            deleteChip={(category, chip) =>
                                onDelete(category as FilterType, chip as string)
                            }
                            deleteChipGroup={(category) => onDeleteGroup(category as FilterType)}
                            categoryName="Fixable"
                        >
                            <CVEStatusDropdown searchFilter={searchFilter} onSelect={onSelect} />
                        </ToolbarFilter>
                    </ToolbarGroup>
                </ToolbarToggleGroup>
            </ToolbarContent>
        </Toolbar>
    );
}

export default WorkloadTableToolbar;
