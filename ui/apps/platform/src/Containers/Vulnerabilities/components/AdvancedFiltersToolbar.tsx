import { useMemo } from 'react';
import type { ReactElement, ReactNode } from 'react';
import { Toolbar, ToolbarContent, ToolbarGroup } from '@patternfly/react-core';

import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import type { CompoundSearchFilterProps } from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import CompoundSearchFilterLabels from 'Components/CompoundSearchFilter/components/CompoundSearchFilterLabels';
import SearchFilterSelectInclusive from 'Components/CompoundSearchFilter/components/SearchFilterSelectInclusive';
import type {
    CompoundSearchFilterAttribute,
    OnSearchPayload,
} from 'Components/CompoundSearchFilter/types';
import { updateSearchFilter } from 'Components/CompoundSearchFilter/utils/utils';
import type { SearchFilter } from 'types/search';
import { getHasSearchApplied, searchValueAsArray } from 'utils/searchUtils';

import type { DefaultFilters } from '../types';
import {
    attributeForClusterCveFixableInFrontend,
    attributeForFixableInFrontendAndLocalStorage,
    attributeForSeverityInFrontendAndLocalStorage,
    attributeForSnoozed,
} from '../searchFilterConfig';
import { normalizeSearchFilterKeys } from '../utils/searchUtils';

const emptyDefaultFilters = {
    Severity: [],
    Fixable: [],
};

type AdvancedFiltersToolbarProps = {
    searchFilterConfig: CompoundSearchFilterProps['config'];
    searchFilter: SearchFilter;
    onFilterChange: (searchFilter: SearchFilter, payload?: OnSearchPayload) => void;
    cveStatusFilterField?: 'Fixable' | 'Cluster CVE Fixable';
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
    cveStatusFilterField = 'Fixable',
    className = '',
    defaultFilters = emptyDefaultFilters,
    includeCveSeverityFilters = true,
    includeCveStatusFilters = true,
    defaultSearchFilterEntity,
    additionalContextFilter,
    children,
}: AdvancedFiltersToolbarProps): ReactElement {
    // Normalize legacy URL keys (e.g. SEVERITY → Severity) so that child
    // components render correctly even for bookmarked URLs from before the rename.
    const normalizedFilter = useMemo(() => normalizeSearchFilterKeys(searchFilter), [searchFilter]);

    const attributesSeparateFromConfig: CompoundSearchFilterAttribute[] = [attributeForSnoozed];
    if (includeCveSeverityFilters) {
        attributesSeparateFromConfig.push(attributeForSeverityInFrontendAndLocalStorage);
    }
    if (includeCveStatusFilters) {
        attributesSeparateFromConfig.push(
            attributeForFixableInFrontendAndLocalStorage,
            attributeForClusterCveFixableInFrontend
        );
    }

    function isGlobalPredicate(category: string, value: string) {
        const values = searchValueAsArray(defaultFilters[category]);
        return values.some((valueDefault) => valueDefault === value);
    }

    function onFilterApplied(payload: OnSearchPayload) {
        onFilterChange(updateSearchFilter(normalizedFilter, payload), payload);
    }

    return (
        <Toolbar className={`advanced-filters-toolbar ${className}`}>
            <ToolbarContent>
                <CompoundSearchFilter
                    config={searchFilterConfig}
                    searchFilter={normalizedFilter}
                    additionalContextFilter={additionalContextFilter}
                    onSearch={onFilterApplied}
                    defaultEntity={defaultSearchFilterEntity}
                />
                {(includeCveSeverityFilters || includeCveStatusFilters) && (
                    <ToolbarGroup variant="filter-group">
                        {includeCveSeverityFilters && (
                            <SearchFilterSelectInclusive
                                attribute={attributeForSeverityInFrontendAndLocalStorage}
                                isSeparate
                                onSearch={onFilterApplied}
                                searchFilter={normalizedFilter}
                            />
                        )}
                        {includeCveStatusFilters && (
                            <SearchFilterSelectInclusive
                                attribute={
                                    cveStatusFilterField === 'Fixable'
                                        ? attributeForFixableInFrontendAndLocalStorage
                                        : attributeForClusterCveFixableInFrontend
                                }
                                isSeparate
                                onSearch={onFilterApplied}
                                searchFilter={normalizedFilter}
                            />
                        )}
                    </ToolbarGroup>
                )}
                {children}
                {getHasSearchApplied(normalizedFilter) && (
                    <ToolbarGroup aria-label="applied search filters" className="pf-v6-u-w-100">
                        <CompoundSearchFilterLabels
                            attributesSeparateFromConfig={attributesSeparateFromConfig}
                            config={searchFilterConfig}
                            isGlobalPredicate={isGlobalPredicate}
                            onFilterChange={onFilterChange}
                            searchFilter={normalizedFilter}
                        />
                    </ToolbarGroup>
                )}
            </ToolbarContent>
        </Toolbar>
    );
}

export default AdvancedFiltersToolbar;
