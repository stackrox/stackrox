import React, { ReactElement } from 'react';
import { TableComposable, Tbody, Td, Thead, Th, Tr } from '@patternfly/react-table';

import useIsRouteEnabled from 'hooks/useIsRouteEnabled';
import { SearchResult, SearchResultCategory } from 'services/SearchService';
import { SearchFilter } from 'types/search';

import FilterLinks from './FilterLinks';
import ViewLinks from './ViewLinks';
import {
    SearchNavCategory,
    searchResultCategoryMapFilteredIsRouteEnabled,
    searchNavMap,
} from './searchCategories';

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
    const isRouteEnabled = useIsRouteEnabled();
    const searchResultCategoryMap = searchResultCategoryMapFilteredIsRouteEnabled(isRouteEnabled);

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

    const searchResultsFilteredAndSorted =
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
                  .sort((a: SearchResult, b: SearchResult) => a.name.localeCompare(b.name));

    return (
        <TableComposable aria-label="Search results" variant="compact" isStickyHeader>
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
                {searchResultsFilteredAndSorted.map(({ category, id, location, name }) => {
                    return (
                        <Tr key={id}>
                            <Td dataLabel={firstColumnHeading} modifier="breakWord">
                                {name}
                                {navCategory === 'SEARCH_UNSET' &&
                                    category !== 'CLUSTERS' &&
                                    typeof location === 'string' &&
                                    location.length !== 0 && (
                                        <div
                                            aria-label={getLocationLabelForCategory(category)}
                                            className="pf-u-color-200"
                                        >
                                            {getLocationTextForCategory(location, category)}
                                        </div>
                                    )}
                            </Td>
                            {hasLocationColumn && (
                                <Td dataLabel={locationColumnHeading} className="pf-u-color-200">
                                    {getLocationTextForCategory(location, category)}
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
                                        id={id}
                                        resultCategory={category}
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
        </TableComposable>
    );
}

export default SearchTable;
