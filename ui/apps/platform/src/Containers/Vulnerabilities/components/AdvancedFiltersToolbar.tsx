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
import {
    cveStatusClusterFixableDescriptor,
    cveStatusFixableDescriptor,
    cveSeverityFilterDescriptor,
    cveSnoozedDescriptor,
} from '../filterChipDescriptor';
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
    /** Hide filter input controls, only show applied filter chips */
    hideInputControls?: boolean;
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
    hideInputControls = false,
}: AdvancedFiltersToolbarProps) {
    const baseDescriptors = makeFilterChipDescriptors(searchFilterConfig);

    const severityDescriptors = includeCveSeverityFilters
        ? [makeDefaultFilterDescriptor(defaultFilters, cveSeverityFilterDescriptor)]
        : [];

    const statusDescriptors = includeCveStatusFilters
        ? [
              makeDefaultFilterDescriptor(defaultFilters, cveStatusFixableDescriptor),
              makeDefaultFilterDescriptor(defaultFilters, cveStatusClusterFixableDescriptor),
          ]
        : [];

    const filterChipGroupDescriptors = baseDescriptors.concat(
        cveSnoozedDescriptor,
        severityDescriptors,
        statusDescriptors
    );

    function onFilterApplied({ category, value, action }: OnSearchPayload) {
        const selectedSearchFilter = searchValueAsArray(searchFilter[category]);

        const newFilter = {
            ...searchFilter,
            [category]:
                action === 'ADD'
                    ? uniq([...selectedSearchFilter, value])
                    : selectedSearchFilter.filter((oldValue) => value !== oldValue),
        };
        onFilterChange(newFilter, { category, value, action });
    }

    return (
        <Toolbar className={`advanced-filters-toolbar ${className}`}>
            <ToolbarContent>
                {!hideInputControls && (
                    <>
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
                    </>
                )}
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
