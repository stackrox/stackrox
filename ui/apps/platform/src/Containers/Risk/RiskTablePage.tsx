import { useParams } from 'react-router-dom-v5-compat';
import { useQuery } from '@apollo/client';

import { PageBody } from 'Components/Panel';
import { DEFAULT_PAGE_SIZE } from 'Components/Table';
import { searchCategories } from 'constants/entityTypes';
import { SEARCH_OPTIONS_QUERY } from 'queries/search';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import useURLSearch from 'hooks/useURLSearch';
import { getHasSearchApplied } from 'utils/searchUtils';

import RiskPageHeader from './RiskPageHeader';
import RiskTablePanel, { sortFields, defaultSortOption } from './RiskTablePanel';

function RiskTablePage() {
    const params = useParams();
    const { deploymentId } = params;
    const urlSort = useURLSort({ sortFields, defaultSortOption });
    const urlPagination = useURLPagination(DEFAULT_PAGE_SIZE);
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
            <PageBody>
                <div className="flex-shrink-1 overflow-hidden w-full">
                    <RiskTablePanel
                        selectedDeploymentId={deploymentId}
                        isViewFiltered={isViewFiltered}
                        sortOption={urlSort.sortOption}
                        onSortOptionChange={(sortOption) => {
                            urlSort.setSortOption(sortOption);
                            urlPagination.setPage(1);
                        }}
                        searchFilter={urlSearch.searchFilter}
                        pagination={urlPagination}
                    />
                </div>
            </PageBody>
        </>
    );
}

export default RiskTablePage;
