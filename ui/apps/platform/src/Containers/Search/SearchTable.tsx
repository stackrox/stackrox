import React, { ReactElement } from 'react';
import { TableComposable, Tbody, Td, Thead, Th, Tr } from '@patternfly/react-table';

import { SearchResult, SearchResultCategory } from 'services/SearchService';
import { SearchFilter } from 'types/search';

import FilterLinks from './FilterLinks';
import ViewLinks from './ViewLinks';
import { SearchNavCategory, searchResultCategoryMap, searchNavMap } from './searchCategories';

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
    const firstColumnHeading = searchNavMap[navCategory];
    const hasLocationColumn =
        navCategory === 'DEPLOYMENTS' || navCategory === 'NAMESPACES' || navCategory === 'NODES';
    const locationColumnHeading = hasLocationColumn ? getLocationLabelForCategory(navCategory) : '';
    const hasCategoryColumn = navCategory === 'SEARCH_UNSET';
    const hasViewLinkColumn =
        navCategory === 'SEARCH_UNSET' ||
        searchResultCategoryMap[navCategory].viewLinks.length !== 0;
    const hasFilterLinkColumn =
        navCategory === 'SEARCH_UNSET' || !!searchResultCategoryMap[navCategory].filterOn;

    const searchResultsFilteredAndSorted =
        navCategory === 'SEARCH_UNSET'
            ? [...searchResults].sort((a: SearchResult, b: SearchResult) => {
                  const byName = a.name.localeCompare(b.name);
                  if (byName === 0) {
                      // If equal by name, secondary sort by category text.
                      return searchNavMap[a.category].localeCompare(searchNavMap[b.category]);
                  }
                  return byName;
              })
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
                                            className="pf-u-color-400"
                                        >
                                            {getLocationTextForCategory(location, category)}
                                        </div>
                                    )}
                            </Td>
                            {hasLocationColumn && (
                                <Td dataLabel={locationColumnHeading} className="pf-u-color-400">
                                    {getLocationTextForCategory(location, category)}
                                </Td>
                            )}
                            {hasCategoryColumn && (
                                <Td dataLabel="Category" modifier="nowrap">
                                    {searchNavMap[category]}
                                </Td>
                            )}
                            {hasViewLinkColumn && (
                                <Td dataLabel="View on">
                                    <ViewLinks id={id} resultCategory={category} />
                                </Td>
                            )}
                            {hasFilterLinkColumn && (
                                <Td dataLabel="Filter on">
                                    <FilterLinks
                                        filterValue={name}
                                        resultCategory={category}
                                        searchFilter={searchFilter}
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
