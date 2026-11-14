import { useEffect, useMemo } from 'react';
import type { ReactElement } from 'react';
import { Pagination } from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import useIsRouteEnabled from 'hooks/useIsRouteEnabled';
import useFeatureFlags from 'hooks/useFeatureFlags';
import useURLPagination from 'hooks/useURLPagination';
import type { SearchResult, SearchResultCategory } from 'services/SearchService';
import type { SearchFilter } from 'types/search';

import FilterLinks from './FilterLinks';
import ViewLinks from './ViewLinks';
import { searchNavMap, searchResultCategoryMapFilteredIsRouteEnabled } from './searchCategories';
import type { SearchNavCategory } from './searchCategories';

// The global search API returns every matching result in one response (no server-side
// pagination), so paginate on the client to avoid rendering thousands of rows at once.
const defaultPerPage = 20;

function getLocationTextForCategory(location: string, category: SearchResultCategory) {
    return category === 'DEPLOYMENTS' ? location.replace(/^\//, '') : location.replace(/\/.+/, '');
}

function getLocationLabelForCategory(category: SearchResultCategory) {
    return category === 'DEPLOYMENTS' ? 'Cluster/Namespace' : 'Cluster';
}

type SearchTableProps = {
    navCategory: SearchNavCategory;
    searchFilter: SearchFilter;
    searchResults: SearchResult[];
};

function SearchTable({ navCategory, searchFilter, searchResults }: SearchTableProps): ReactElement {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isRouteEnabled = useIsRouteEnabled();
    const { page, perPage, setPage, setPerPage } = useURLPagination(defaultPerPage);
    const searchResultCategoryMap = searchResultCategoryMapFilteredIsRouteEnabled(
        isRouteEnabled,
        isFeatureFlagEnabled
    );

    const firstColumnHeading = searchNavMap[navCategory];
    const hasLocationColumn =
        navCategory === 'DEPLOYMENTS' || navCategory === 'NAMESPACES' || navCategory === 'NODES';
    const locationColumnHeading = hasLocationColumn ? getLocationLabelForCategory(navCategory) : '';
    const hasCategoryColumn = navCategory === 'SEARCH_UNSET';
    const hasViewLinkColumn =
        navCategory === 'SEARCH_UNSET' ||
        Boolean(searchResultCategoryMap[navCategory]?.viewLinks?.length);
    const hasFilterLinkColumn =
        navCategory === 'SEARCH_UNSET' || Boolean(searchResultCategoryMap[navCategory]?.filterOn);

    // Memoize so switching page does not re-filter and re-sort the full result set.
    const searchResultsFilteredAndSorted = useMemo(
        () =>
            navCategory === 'SEARCH_UNSET'
                ? [...searchResults].sort(
                      (
                          { name: namePrev, category: categoryPrev }: SearchResult,
                          { name: nameNext, category: categoryNext }: SearchResult
                      ) => {
                          if (namePrev < nameNext) {
                              return -1;
                          }
                          if (namePrev > nameNext) {
                              return 1;
                          }

                          // If equal by name, secondary sort by category text.
                          const categoryNavPrev = searchNavMap[categoryPrev] ?? categoryPrev;
                          const categoryNavNext = searchNavMap[categoryNext] ?? categoryNext;
                          if (categoryNavPrev < categoryNavNext) {
                              return -1;
                          }
                          if (categoryNavPrev > categoryNavNext) {
                              return 1;
                          }
                          return 0;
                      }
                  )
                : searchResults
                      .filter(({ category }) => category === navCategory)
                      .sort((a: SearchResult, b: SearchResult) => a.name.localeCompare(b.name)),
        [navCategory, searchResults]
    );

    const itemCount = searchResultsFilteredAndSorted.length;
    // Clamp defensively in case the URL carries a page beyond the current result range.
    const lastPage = Math.max(1, Math.ceil(itemCount / perPage));
    const currentPage = Math.min(page, lastPage);
    const startIndex = (currentPage - 1) * perPage;

    // Keep the URL page parameter in sync with the clamped page, e.g. when a bookmarked
    // or refreshed URL points beyond the result range after results shrink. Replace
    // history so the out-of-range value does not become a back-button target.
    useEffect(() => {
        if (page > lastPage) {
            setPage(lastPage, 'replace');
        }
    }, [page, lastPage, setPage]);
    const searchResultsForPage = searchResultsFilteredAndSorted.slice(
        startIndex,
        startIndex + perPage
    );

    return (
        <>
            <Pagination
                itemCount={itemCount}
                page={currentPage}
                perPage={perPage}
                onSetPage={(_, newPage) => setPage(newPage)}
                onPerPageSelect={(_, newPerPage) => setPerPage(newPerPage)}
            />
            <Table aria-label="Search results" variant="compact" isStickyHeader>
                <Thead>
                    <Tr>
                        <Th>{firstColumnHeading}</Th>
                        {hasLocationColumn && <Th>{locationColumnHeading}</Th>}
                        {hasCategoryColumn && <Th>Category</Th>}
                        {hasViewLinkColumn && <Th>View on</Th>}
                        {hasFilterLinkColumn && <Th>Filter on</Th>}
                    </Tr>
                </Thead>
                <Tbody>
                    {searchResultsForPage.map((searchResult) => {
                        const { category, id, location, name } = searchResult;
                        const locationTextForCategory = getLocationTextForCategory(
                            location,
                            category
                        );
                        return (
                            <Tr key={id}>
                                <Td dataLabel={firstColumnHeading} modifier="breakWord">
                                    {name}
                                    {navCategory === 'SEARCH_UNSET' &&
                                        category !== 'CLUSTERS' &&
                                        typeof location === 'string' &&
                                        location.length !== 0 && (
                                            <div aria-label={getLocationLabelForCategory(category)}>
                                                {locationTextForCategory}
                                            </div>
                                        )}
                                </Td>
                                {hasLocationColumn && (
                                    <Td dataLabel={locationColumnHeading}>
                                        {locationTextForCategory}
                                    </Td>
                                )}
                                {hasCategoryColumn && (
                                    <Td dataLabel="Category" modifier="nowrap">
                                        {searchNavMap[category] ?? category}
                                    </Td>
                                )}
                                {hasViewLinkColumn && (
                                    <Td dataLabel="View on">
                                        <ViewLinks
                                            searchResult={{
                                                ...searchResult,
                                                locationTextForCategory,
                                            }}
                                            searchResultCategoryMap={searchResultCategoryMap}
                                        />
                                    </Td>
                                )}
                                {hasFilterLinkColumn && (
                                    <Td dataLabel="Filter on">
                                        <FilterLinks
                                            filterValue={name}
                                            resultCategory={category}
                                            searchFilter={searchFilter}
                                            searchResultCategoryMap={searchResultCategoryMap}
                                        />
                                    </Td>
                                )}
                            </Tr>
                        );
                    })}
                </Tbody>
            </Table>
        </>
    );
}

export default SearchTable;
