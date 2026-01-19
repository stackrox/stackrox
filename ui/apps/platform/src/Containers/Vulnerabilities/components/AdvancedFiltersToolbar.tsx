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
    attributeForClusterCveFixable,
    attributeForFixable,
    attributeForSeverity,
    attributeForSnoozed,
} from '../searchFilterConfig';

import './AdvancedFiltersToolbar.css';

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
        attributesSeparateFromConfig.push(attributeForSeverity);
    }
    if (includeCveStatusFilters) {
        attributesSeparateFromConfig.push(attributeForFixable, attributeForClusterCveFixable);
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
                    <ToolbarGroup className="vm-filter-toolbar-dropdown">
                        {includeCveSeverityFilters && (
                            <SearchFilterSelectInclusive
                                attribute={attributeForSeverity}
                                isSeparate
                                onSearch={onFilterApplied}
                                searchFilter={searchFilter}
                            />
                        )}
                        {includeCveStatusFilters && (
                            <SearchFilterSelectInclusive
                                attribute={
                                    cveStatusFilterField === 'FIXABLE'
                                        ? attributeForFixable
                                        : attributeForClusterCveFixable
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
                    <ToolbarGroup aria-label="applied search filters" className="pf-v5-u-w-100">
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
