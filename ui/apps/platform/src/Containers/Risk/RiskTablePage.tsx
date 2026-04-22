import { useQuery } from '@apollo/client';
import { Flex, PageSection } from '@patternfly/react-core';

import { searchCategories } from 'constants/entityTypes';
import { SEARCH_OPTIONS_QUERY } from 'queries/search';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import useURLSearch from 'hooks/useURLSearch';

import SearchFilterInput from 'Components/SearchFilterInput';
import type { FilteredWorkflowView } from 'Components/FilteredWorkflowViewSelector/types';
import useFilteredWorkflowViewURLState from 'Components/FilteredWorkflowViewSelector/useFilteredWorkflowViewURLState';
import type { SearchFilter } from 'types/search';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';

import RiskTablePanel, { sortFields, defaultSortOption } from './RiskTablePanel';
import RiskPageHeader from './RiskPageHeader';

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
    const additionalContextFilter = getFilteredWorkflowViewSearchFilter(filteredWorkflowView);

    const searchQueryOptions = {
        variables: {
            categories: [searchCategories.DEPLOYMENT],
        },
    };
    const { data: searchData } = useQuery<{
        searchOptions: string[];
    }>(SEARCH_OPTIONS_QUERY, searchQueryOptions);
    const searchOptions = searchData?.searchOptions ?? [];
    const filteredOptions = searchOptions.filter(
        (option) => option !== 'Platform Component' && option !== 'Orchestrator Component'
    );

    const autoCompleteCategory = searchCategories.DEPLOYMENT;

    const mergedSearchFilter: SearchFilter = {
        ...urlSearch.searchFilter,
        ...additionalContextFilter,
    };

    return (
        <>
            <RiskPageHeader />
            {/* Nested PageSection here for visual consistency **as-is**. Once we move to Patternfly 6, we can remove this and clean up */}
            <PageSection>
                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
                    <SearchFilterInput
                        className=""
                        searchFilter={urlSearch.searchFilter}
                        searchOptions={filteredOptions}
                        searchCategory={autoCompleteCategory}
                        placeholder="Filter deployments"
                        handleChangeSearchFilter={(newSearchFilter) => {
                            urlSearch.setSearchFilter(newSearchFilter);
                            urlPagination.setPage(1);
                        }}
                        autocompleteQueryPrefix={getRequestQueryStringForSearchFilter(
                            additionalContextFilter
                        )}
                    />
                    <RiskTablePanel
                        sortOption={urlSort.sortOption}
                        getSortParams={urlSort.getSortParams}
                        searchFilter={mergedSearchFilter}
                        onSearchFilterChange={(newSearchFilter) => {
                            urlSearch.setSearchFilter(newSearchFilter);
                            urlPagination.setPage(1);
                        }}
                        pagination={urlPagination}
                    />
                </Flex>
            </PageSection>
        </>
    );
}

export default RiskTablePage;
