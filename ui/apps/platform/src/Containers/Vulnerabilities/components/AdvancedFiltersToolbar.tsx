import type { ReactElement, ReactNode } from 'react';
import { Toolbar, ToolbarContent, ToolbarGroup } from '@patternfly/react-core';

import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import type { CompoundSearchFilterProps } from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import SearchFilterChips, {
    FilterChip,
    makeFilterChipDescriptors,
} from 'Components/CompoundSearchFilter/components/SearchFilterChips';
import type { OnSearchPayload } from 'Components/CompoundSearchFilter/types';
import { updateSearchFilter } from 'Components/CompoundSearchFilter/utils/utils';
import type { SearchFilter } from 'types/search';
import { getHasSearchApplied } from 'utils/searchUtils';

import type { DefaultFilters } from '../types';
import {
    cveSeverityFilterDescriptor,
    cveSnoozedDescriptor,
    cveStatusClusterFixableDescriptor,
    cveStatusFixableDescriptor,
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
}: AdvancedFiltersToolbarProps): ReactElement {
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

    function onFilterApplied(payload: OnSearchPayload) {
        onFilterChange(updateSearchFilter(searchFilter, payload), payload);
    }

    return (
        <Toolbar className={`advanced-filters-toolbar ${className}`}>
            <ToolbarContent>
                <ToolbarGroup
                    variant="filter-group"
                    className="pf-v6-u-display-flex pf-v6-u-flex-grow-1"
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
                                    onFilterApplied([
                                        {
                                            category,
                                            value,
                                            action: checked ? 'SELECT_INCLUSIVE' : 'REMOVE',
                                        },
                                    ])
                                }
                            />
                        )}
                        {includeCveStatusFilters && (
                            <CVEStatusDropdown
                                filterField={cveStatusFilterField}
                                searchFilter={searchFilter}
                                onSelect={(category, checked, value) =>
                                    onFilterApplied([
                                        {
                                            category,
                                            value,
                                            action: checked ? 'SELECT_INCLUSIVE' : 'REMOVE',
                                        },
                                    ])
                                }
                            />
                        )}
                    </ToolbarGroup>
                )}
                {children}
                {getHasSearchApplied(searchFilter) && (
                    <ToolbarGroup aria-label="applied search filters" className="pf-v6-u-w-100">
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
