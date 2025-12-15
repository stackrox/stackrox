import { useQuery } from '@apollo/client';
import { Divider, PageSection } from '@patternfly/react-core';

import entityTypes, { searchCategories } from 'constants/entityTypes';
import { SEARCH_OPTIONS_QUERY } from 'queries/search';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import useURLSearch from 'hooks/useURLSearch';
import { getHasSearchApplied } from 'utils/searchUtils';

import SearchFilterInput from 'Components/SearchFilterInput';
import searchOptionsToQuery from 'services/searchOptionsToQuery';
import {
    ORCHESTRATOR_COMPONENTS_KEY,
    orchestratorComponentsOption,
} from 'utils/orchestratorComponents';

import RiskTablePanel, { sortFields, defaultSortOption } from './RiskTablePanel';
import RiskPageHeader from './RiskPageHeader';

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

    const autoCompleteCategory = searchCategories[entityTypes.DEPLOYMENT];

    const orchestratorComponentShowState = localStorage.getItem(ORCHESTRATOR_COMPONENTS_KEY);
    const prependAutocompleteQuery =
        orchestratorComponentShowState !== 'true' ? orchestratorComponentsOption : [];

    return (
        <>
            <RiskPageHeader isViewFiltered={isViewFiltered} />
            {/* Nested PageSection here for visual consistency **as-is**. Once we move to Patternfly 6, we can remove this and clean up */}
            <PageSection>
                <PageSection variant="light" component="div">
                    <SearchFilterInput
                        className="w-full"
                        searchFilter={urlSearch.searchFilter}
                        searchOptions={filteredSearchOptions}
                        searchCategory={autoCompleteCategory}
                        placeholder="Filter deployments"
                        handleChangeSearchFilter={(newSearchFilter) => {
                            urlSearch.setSearchFilter(newSearchFilter);
                            urlPagination.setPage(1);
                        }}
                        autocompleteQueryPrefix={searchOptionsToQuery(prependAutocompleteQuery)}
                    />
                    <Divider component="div" className="pf-v5-u-mt-lg" />
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
