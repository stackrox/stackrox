import React, { ReactElement } from 'react';
import { TableComposable, Tbody, Td, Thead, Th, Tr } from '@patternfly/react-table';

import { SearchResult, SearchResultCategory } from 'services/SearchService';
import { SearchFilter } from 'types/search';

import FilterLinks from './FilterLinks';
import ViewLinks from './ViewLinks';
import { SearchTabCategory, searchResultCategoryMap, searchTabMap } from './searchCategories';

function getLocationTextForCategory(location: string, category: SearchResultCategory) {
    return category === 'DEPLOYMENTS' ? location.replace(/^\//, '') : location.replace(/\/.+/, '');
}

function getLocationLabelForCategory(category: SearchResultCategory) {
    return category === 'DEPLOYMENTS' ? 'Cluster/Namespace' : 'Cluster';
}

type SearchTableProps = {
    searchFilter: SearchFilter;
    searchResults: SearchResult[];
    tabCategory: SearchTabCategory;
};

function SearchTable({ searchFilter, searchResults, tabCategory }: SearchTableProps): ReactElement {
    const firstColumnHeading = searchTabMap[tabCategory];
    const hasLocationColumn =
        tabCategory === 'DEPLOYMENTS' || tabCategory === 'NAMESPACES' || tabCategory === 'NODES';
    const locationColumnHeading = hasLocationColumn ? getLocationLabelForCategory(tabCategory) : '';
    const hasCategoryColumn = tabCategory === 'SEARCH_UNSET';
    const hasViewLinkColumn =
        tabCategory === 'SEARCH_UNSET' ||
        searchResultCategoryMap[tabCategory].viewLinks.length !== 0;
    const hasFilterLinkColumn =
        tabCategory === 'SEARCH_UNSET' || !!searchResultCategoryMap[tabCategory].filterOn;

    const searchResultsFilteredAndSorted =
        tabCategory === 'SEARCH_UNSET'
            ? [...searchResults].sort((a: SearchResult, b: SearchResult) => {
                  const byName = a.name.localeCompare(b.name);
                  if (byName === 0) {
                      // If equal by name, secondary sort by category text.
                      return searchTabMap[a.category].localeCompare(searchTabMap[b.category]);
                  }
                  return byName;
              })
            : searchResults
                  .filter(({ category }) => category === tabCategory)
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
                                {tabCategory === 'SEARCH_UNSET' &&
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
                                    {searchTabMap[category]}
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
