import React, { ReactNode } from 'react';
import { Toolbar, ToolbarGroup, ToolbarContent } from '@patternfly/react-core';
import { uniq } from 'lodash';

import CompoundSearchFilter, {
    CompoundSearchFilterProps,
} from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import { OnSearchPayload } from 'Components/CompoundSearchFilter/types';
import { makeFilterChipDescriptors } from 'Components/CompoundSearchFilter/utils/utils';
import SearchFilterChips, { FilterChip } from 'Components/PatternFly/SearchFilterChips';
import { SearchFilter } from 'types/search';
import { getHasSearchApplied, searchValueAsArray } from 'utils/searchUtils';

import { DefaultFilters } from '../types';
import CVESeverityDropdown from './CVESeverityDropdown';
import CVEStatusDropdown from './CVEStatusDropdown';

import './AdvancedFiltersToolbar.css';

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
    onFilterChange: (searchFilter: SearchFilter, payload?: OnSearchPayload) => void;
    cveStatusFilterField?: 'FIXABLE' | 'CLUSTER CVE FIXABLE';
    className?: string;
    defaultFilters?: DefaultFilters;
    includeCveSeverityFilters?: boolean;
    includeCveStatusFilters?: boolean;
    defaultSearchFilterEntity?: string;
    additionalContextFilter?: SearchFilter;
    children?: ReactNode;
};

function AdvancedFiltersToolbar({
    searchFilterConfig,
    searchFilter,
    onFilterChange,
    cveStatusFilterField = 'FIXABLE',
    className = '',
    defaultFilters = emptyDefaultFilters,
    includeCveSeverityFilters = true,
    includeCveStatusFilters = true,
    defaultSearchFilterEntity,
    additionalContextFilter,
    children,
}: AdvancedFiltersToolbarProps) {
    const filterChipGroupDescriptors = makeFilterChipDescriptors(searchFilterConfig)
        .concat({
            displayName: 'CVE snoozed',
            searchFilterName: 'CVE Snoozed',
        })
        .concat(
            includeCveSeverityFilters
                ? makeDefaultFilterDescriptor(defaultFilters, {
                      displayName: 'CVE severity',
                      searchFilterName: 'SEVERITY',
                  })
                : []
        )
        .concat(
            includeCveStatusFilters
                ? [
                      makeDefaultFilterDescriptor(defaultFilters, {
                          displayName: 'CVE status',
                          searchFilterName: 'FIXABLE',
                      }),
                      makeDefaultFilterDescriptor(defaultFilters, {
                          displayName: 'CVE status',
                          searchFilterName: 'CLUSTER CVE FIXABLE',
                      }),
                  ]
                : []
        );

    function onFilterApplied({ category, value, action }: OnSearchPayload) {
        const selectedSearchFilter = searchValueAsArray(searchFilter[category]);

        const newFilter = {
            ...searchFilter,
            [category]:
                action === 'ADD'
                    ? uniq([...selectedSearchFilter, value])
                    : action === 'REMOVE'
                      ? selectedSearchFilter.filter((oldValue) => value !== oldValue)
                      : [value],
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
                        additionalContextFilter={additionalContextFilter}
                        onSearch={onFilterApplied}
                        defaultEntity={defaultSearchFilterEntity}
                    />
                </ToolbarGroup>
                {(includeCveSeverityFilters || includeCveStatusFilters) && (
                    <ToolbarGroup>
                        {includeCveSeverityFilters && (
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
                        )}
                        {includeCveStatusFilters && (
                            <CVEStatusDropdown
                                filterField={cveStatusFilterField}
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
                {children}
                {getHasSearchApplied(searchFilter) && (
                    <ToolbarGroup aria-label="applied search filters" className="pf-v5-u-w-100">
                        <SearchFilterChips
                            searchFilter={searchFilter}
                            onFilterChange={onFilterChange}
                            filterChipGroupDescriptors={filterChipGroupDescriptors}
                        />
                    </ToolbarGroup>
                )}
            </ToolbarContent>
        </Toolbar>
    );
}

export default AdvancedFiltersToolbar;
