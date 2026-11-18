import { useMemo } from 'react';
import { Flex, PageSection } from '@patternfly/react-core';

import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import useURLSearch from 'hooks/useURLSearch';

import type { FilteredWorkflowView } from 'Components/FilteredWorkflowViewSelector/types';
import useFilteredWorkflowViewURLState from 'Components/FilteredWorkflowViewSelector/useFilteredWorkflowViewURLState';
import type { OnSearchCallback } from 'Components/CompoundSearchFilter/types';
import { updateSearchFilter } from 'Components/CompoundSearchFilter/utils/utils';
import type { SearchFilter } from 'types/search';

import RiskTablePanel, { sortFields, defaultSortOption } from './RiskTablePanel';
import RiskPageHeader from './RiskPageHeader';
import RiskTableSearchFilter from './RiskTableSearchFilter';

const DEFAULT_RISK_PAGE_SIZE = 20;

function getFilteredWorkflowViewSearchFilter(
    filteredWorkflowView: FilteredWorkflowView
): SearchFilter {
    switch (filteredWorkflowView) {
        case 'Applications view':
            return { 'Platform Component': 'false' };
        case 'Platform view':
            return { 'Platform Component': 'true' };
        case 'Full view':
        default:
            return {};
    }
}

function RiskTablePage() {
    const urlSort = useURLSort({
        sortFields,
        defaultSortOption,
        onSort: () => urlPagination.setPage(1),
    });
    const urlPagination = useURLPagination(DEFAULT_RISK_PAGE_SIZE);
    const urlSearch = useURLSearch();

    const { filteredWorkflowView } = useFilteredWorkflowViewURLState();
    const additionalContextFilter = useMemo(
        () => getFilteredWorkflowViewSearchFilter(filteredWorkflowView),
        [filteredWorkflowView]
    );
    const mergedSearchFilter = useMemo(
        () => ({
            ...urlSearch.searchFilter,
            ...additionalContextFilter,
        }),
        [urlSearch.searchFilter, additionalContextFilter]
    );

    const onSearch: OnSearchCallback = (payload) => {
        urlSearch.setSearchFilter(updateSearchFilter(urlSearch.searchFilter, payload));
        urlPagination.setPage(1);
    };

    const onFilterChange = (newSearchFilter: SearchFilter) => {
        urlSearch.setSearchFilter(newSearchFilter);
        urlPagination.setPage(1);
    };

    return (
        <>
            <RiskPageHeader />
            {/* Nested PageSection here for visual consistency **as-is**. Once we move to Patternfly 6, we can remove this and clean up */}
            <PageSection>
                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
                    <RiskTableSearchFilter
                        searchFilter={urlSearch.searchFilter}
                        onSearch={onSearch}
                        onFilterChange={onFilterChange}
                        additionalContextFilter={additionalContextFilter}
                    />
                    <RiskTablePanel
                        sortOption={urlSort.sortOption}
                        getSortParams={urlSort.getSortParams}
                        searchFilter={mergedSearchFilter}
                        onSearchFilterChange={onFilterChange}
                        pagination={urlPagination}
                    />
                </Flex>
            </PageSection>
        </>
    );
}

export default RiskTablePage;
