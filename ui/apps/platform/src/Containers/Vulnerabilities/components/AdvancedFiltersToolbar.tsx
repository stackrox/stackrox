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
        onFilterChange(updateSearchFilter(searchFilter, payload), payload);
    }

    return (
        <Toolbar className={`advanced-filters-toolbar ${className}`}>
            <ToolbarContent>
                <CompoundSearchFilter
                    config={searchFilterConfig}
                    searchFilter={searchFilter}
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
                                searchFilter={searchFilter}
                            />
                        )}
                        {includeCveStatusFilters && (
                            <SearchFilterSelectInclusive
                                attribute={
                                    cveStatusFilterField === 'FIXABLE'
                                        ? attributeForFixableInFrontendAndLocalStorage
                                        : attributeForClusterCveFixableInFrontend
                                }
                                isSeparate
                                onSearch={onFilterApplied}
                                searchFilter={searchFilter}
                            />
                        )}
                    </ToolbarGroup>
                )}
                {children}
                {getHasSearchApplied(searchFilter) && (
                    <ToolbarGroup aria-label="applied search filters" className="pf-v6-u-w-100">
                        <CompoundSearchFilterLabels
                            attributesSeparateFromConfig={attributesSeparateFromConfig}
                            config={searchFilterConfig}
                            isGlobalPredicate={isGlobalPredicate}
                            onFilterChange={onFilterChange}
                            searchFilter={searchFilter}
                        />
                    </ToolbarGroup>
                )}
            </ToolbarContent>
        </Toolbar>
    );
}

export default AdvancedFiltersToolbar;
