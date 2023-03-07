import React, { useEffect } from 'react';
import {
    Toolbar,
    ToolbarItem,
    ToolbarFilter,
    ToolbarToggleGroup,
    ToolbarGroup,
    ToolbarContent,
    ToolbarChip,
    Flex,
} from '@patternfly/react-core';
import { FilterIcon } from '@patternfly/react-icons';
import { Globe } from 'react-feather';

import useURLSearch from 'hooks/useURLSearch';
import { uniq } from 'lodash';
import { DefaultFilters, VulnerabilitySeverityLabel, FixableStatus } from './types';
import FilterResourceDropdown, { Resource } from './FilterResourceDropdown';
import FilterAutocompleteSelect from './FilterAutocompleteSelect';
import CVESeverityDropdown from './CVESeverityDropdown';
import CVEStatusDropdown from './CVEStatusDropdown';

import './WorkloadTableToolbar.css';

type FilterType = 'resource' | 'Severity' | 'Fixable';

type WorkloadTableToolbarProps = {
    defaultFilters: DefaultFilters;
    resourceContext?: Resource;
};

function WorkloadTableToolbar({ defaultFilters, resourceContext }: WorkloadTableToolbarProps) {
    const { searchFilter, setSearchFilter } = useURLSearch();
    const severityFilterChips: ToolbarChip[] = [];
    const fixableFilterChips: ToolbarChip[] = [];

    function onSelect(type: FilterType, e, selection) {
        if (type === 'resource') {
            setSearchFilter({
                ...searchFilter,
                resource: selection,
            });
        } else {
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
    }

    function onDelete(type: FilterType, id: string) {
        if (type === 'Severity') {
            const severitySearchFilter = searchFilter.Severity as string[];
            setSearchFilter({
                ...searchFilter,
                Severity: severitySearchFilter.filter((fil: string) => fil !== id),
            });
        } else if (type === 'Fixable') {
            const fixableSearchFilter = searchFilter.Fixable as string[];
            setSearchFilter({
                ...searchFilter,
                Fixable: fixableSearchFilter.filter((fil: string) => fil !== id),
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
            setSearchFilter({
                ...defaultFilters,
                ...searchFilter,
                Severity: severityFilter,
                Fixable: fixableFilter,
            });
        }
    }, [defaultFilters, searchFilter, setSearchFilter]);

    useEffect(() => {
        if (!searchFilter.resource) {
            setSearchFilter({ ...searchFilter, resource: 'CVE' });
        }
    }, []);

    const severitySearchFilter = searchFilter.Severity as VulnerabilitySeverityLabel[];
    severitySearchFilter?.forEach((sev) => {
        if (defaultFilters.Severity?.includes(sev)) {
            severityFilterChips.push({
                key: sev,
                node: (
                    <Flex alignItems={{ default: 'alignItemsCenter' }}>
                        <Globe height="15px" />
                        {sev}
                    </Flex>
                ),
            });
        } else {
            severityFilterChips.push({
                key: sev,
                node: <Flex>{sev}</Flex>,
            });
        }
    });

    const fixableSearchFilter = searchFilter.Fixable as FixableStatus[];
    fixableSearchFilter?.forEach((status) => {
        if (defaultFilters.Fixable?.includes(status)) {
            fixableFilterChips.push({
                key: status,
                node: (
                    <Flex alignItems={{ default: 'alignItemsCenter' }}>
                        <Globe height="15px" />
                        {status}
                    </Flex>
                ),
            });
        } else {
            fixableFilterChips.push({
                key: status,
                node: <Flex>{status}</Flex>,
            });
        }
    });

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
                    className="pf-u-flex-1"
                >
                    <ToolbarGroup variant="filter-group" className="pf-u-flex-grow-1">
                        <ToolbarItem className="pf-u-w-25">
                            <FilterResourceDropdown
                                onSelect={onSelect}
                                searchFilter={searchFilter}
                                resourceContext={resourceContext}
                            />
                        </ToolbarItem>
                        <ToolbarItem variant="search-filter" className="pf-u-flex-grow-1">
                            <FilterAutocompleteSelect
                                searchFilter={searchFilter}
                                setSearchFilter={setSearchFilter}
                            />
                        </ToolbarItem>
                    </ToolbarGroup>
                    <ToolbarGroup>
                        <ToolbarFilter
                            chips={severityFilterChips}
                            deleteChip={(category, chip) =>
                                onDelete(category as FilterType, chip as string)
                            }
                            deleteChipGroup={(category) => onDeleteGroup(category as FilterType)}
                            categoryName="Severity"
                        >
                            <CVESeverityDropdown searchFilter={searchFilter} onSelect={onSelect} />
                        </ToolbarFilter>
                        <ToolbarFilter
                            chips={fixableFilterChips}
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
