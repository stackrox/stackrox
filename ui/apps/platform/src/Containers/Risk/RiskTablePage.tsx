import { useQuery } from '@apollo/client';
import { PageSection } from '@patternfly/react-core';

import { searchCategories } from 'constants/entityTypes';
import { SEARCH_OPTIONS_QUERY } from 'queries/search';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import useURLSearch from 'hooks/useURLSearch';
import { getHasSearchApplied } from 'utils/searchUtils';

import RiskPageHeader from './RiskPageHeader';
import RiskTablePanel, { sortFields, defaultSortOption } from './RiskTablePanel';

const DEFAULT_RISK_PAGE_SIZE = 20;

function RiskTablePage() {
    const urlSort = useURLSort({
        sortFields,
        defaultSortOption,
        onSort: () => urlPagination.setPage(1),
    });
    const urlPagination = useURLPagination(DEFAULT_RISK_PAGE_SIZE);
    const urlSearch = useURLSearch();

    const isViewFiltered = getHasSearchApplied(urlSearch.searchFilter);

    const searchQueryOptions = {
        variables: {
            categories: [searchCategories.DEPLOYMENT],
        },
    };
    const { data: searchData } = useQuery(SEARCH_OPTIONS_QUERY, searchQueryOptions);
    const searchOptions = searchData?.searchOptions ?? [];
    const filteredSearchOptions = searchOptions.filter(
        (option) => option !== 'Orchestrator Component'
    );
    return (
        <>
            <RiskPageHeader
                isViewFiltered={isViewFiltered}
                searchOptions={filteredSearchOptions}
                searchFilter={urlSearch.searchFilter}
                onSearch={(newSearchFilter) => {
                    urlPagination.setPage(1);
                    urlSearch.setSearchFilter(newSearchFilter);
                }}
            />
            {/* Nested PageSection here for visual consistency **as-is**. Once we move to Patternfly 6, we can remove this and clean up */}
            <PageSection>
                <PageSection variant="light" component="div">
                    <RiskTablePanel
                        sortOption={urlSort.sortOption}
                        getSortParams={urlSort.getSortParams}
                        searchFilter={urlSearch.searchFilter}
                        onSearchFilterChange={(newSearchFilter) => {
                            urlSearch.setSearchFilter(newSearchFilter);
                            urlPagination.setPage(1);
                        }}
                        pagination={urlPagination}
                    />
                </PageSection>
            </PageSection>
        </>
    );
}

export default RiskTablePage;
