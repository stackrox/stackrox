import React, { ReactElement, useEffect, useState } from 'react';
import capitalize from 'lodash/capitalize';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import {
    Badge,
    Bullseye,
    Button,
    Flex,
    FlexItem,
    Tabs,
    Tab,
    TabTitleText,
    TabContent,
    Title,
} from '@patternfly/react-core';
import { TableComposable, Tbody, Td, Thead, Th, Tr } from '@patternfly/react-table';

import { getUrlQueryStringForSearchFilter, searchOptionsToSearchFilter } from 'utils/searchUtils';
import { selectors } from 'reducers';
import { actions as globalSearchActions } from 'reducers/globalSearch';
import { SearchCategory, SearchCategoryCount, SearchResult } from 'services/SearchService';
import { SearchEntry } from 'types/search';
import { SortDirection } from 'types/table';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';

import searchCategoryDescriptorMap from './searchCategoryDescriptorMap';

type TabCategory = 'SEARCH_UNSET' | 'ALERTS' | 'POLICIES' | 'DEPLOYMENTS' | 'IMAGES' | 'SECRETS';

type SearchTab = {
    tabCategory: TabCategory;
    text: string;
};

const tabs: SearchTab[] = [
    {
        tabCategory: 'SEARCH_UNSET',
        text: 'All',
    },
    {
        tabCategory: 'ALERTS',
        text: 'Violations',
    },
    {
        tabCategory: 'POLICIES',
        text: 'Policies',
    },
    {
        tabCategory: 'DEPLOYMENTS',
        text: 'Deployments',
    },
    {
        tabCategory: 'IMAGES',
        text: 'Images',
    },
    {
        tabCategory: 'SECRETS',
        text: 'Secrets',
    },
];

const INITIAL_SORT_INDEX = 1; // Type column
const INITIAL_SORT_DIRECTION = 'asc'; // A->Z

interface StateProps {
    searchCategory: SearchCategory;
    searchCounts: SearchCategoryCount[];
    searchResults: SearchResult[];
    searchOptions: SearchEntry[];
}

interface DispatchProps {
    setSearchCategory: (category: string) => void;
}

interface PassedProps {
    onClose: (toURL: string) => void;
}

export type SearchResultsProps = StateProps & DispatchProps & PassedProps;

function SearchResults({
    onClose,
    searchCategory,
    searchCounts,
    searchOptions,
    searchResults,
    setSearchCategory,
}: SearchResultsProps): ReactElement {
    // index of the currently active column
    const [activeSortIndex, setActiveSortIndex] = useState(INITIAL_SORT_INDEX);
    // sort direction of the currently active column
    const [activeSortDirection, setActiveSortDirection] =
        useState<SortDirection>(INITIAL_SORT_DIRECTION);
    const [sortedRows, setSortedRows] = useState<SearchResult[]>([]);

    useEffect(() => {
        const newSortedResults = onSort(searchResults, INITIAL_SORT_INDEX, INITIAL_SORT_DIRECTION);
        setSortedRows(newSortedResults);
    }, [searchResults]);

    function onSort(
        currentRows: SearchResult[],
        index: number,
        direction: SortDirection
    ): SearchResult[] {
        setActiveSortIndex(index);
        setActiveSortDirection(direction);
        // sorts the rows
        const updatedRows = [...currentRows].sort((a, b) => {
            if (index === 0) {
                // sort on first column, name
                if (direction === 'asc') {
                    return a.name.localeCompare(b.name);
                }
                return b.name.localeCompare(a.name);
            }

            // sort on second column, type
            if (direction === 'asc') {
                return a.category.localeCompare(b.category);
            }
            return b.category.localeCompare(a.category);
        });
        return updatedRows;
    }

    function handleHeaderClick(_event, index, direction) {
        const updatedRows = onSort(sortedRows, index, direction);
        setSortedRows(updatedRows);
    }

    function onTabClick(_event, eventKey) {
        setSearchCategory(eventKey);
    }

    if (searchOptions.length === 0) {
        return (
            <Bullseye>
                <EmptyStateTemplate title="Search all data" headingLevel="h1">
                    Choose one or more filter values to search.
                </EmptyStateTemplate>
            </Bullseye>
        );
    }

    /*
     * Replace searchCounts.reduce(…) with searchResults.length after future improvement:
     * replace redundant requests for each selected tab categories
     * with filtering of the response for all categories
     */
    function getTabCategoryCount(tabCategory: TabCategory) {
        return tabCategory === 'SEARCH_UNSET'
            ? searchCounts.reduce((total, { count }) => total + Number(count), 0)
            : searchCounts.find(({ category }) => category === tabCategory)?.count ?? 0;
    }

    /* eslint-disable no-nested-ternary */
    return (
        <div className="bg-base-100 flex-1" data-testid="global-search-results">
            <Title headingLevel="h1" className="px-4 py-4">
                Search
            </Title>
            <section className="h-full">
                <Tabs activeKey={searchCategory} onSelect={onTabClick}>
                    {tabs.map(({ tabCategory, text }) => (
                        <Tab
                            key={tabCategory}
                            eventKey={tabCategory}
                            title={
                                <TabTitleText>
                                    <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                                        <FlexItem>{text}</FlexItem>
                                        <FlexItem>
                                            <Badge isRead>{getTabCategoryCount(tabCategory)}</Badge>
                                        </FlexItem>
                                    </Flex>
                                </TabTitleText>
                            }
                        />
                    ))}
                </Tabs>
                {tabs.map(({ tabCategory, text }) => (
                    <TabContent
                        eventKey={tabCategory}
                        className="overflow-auto"
                        id={tabCategory}
                        aria-label={text}
                        key={tabCategory}
                        hidden={tabCategory !== searchCategory}
                    >
                        {tabCategory !== searchCategory ? null : sortedRows.length === 0 ? (
                            <EmptyStateTemplate
                                title="No results with your chosen filters for the type"
                                headingLevel="h2"
                            >
                                Try changing the filter values.
                            </EmptyStateTemplate>
                        ) : (
                            <SearchResultsTable
                                activeSortDirection={activeSortDirection}
                                activeSortIndex={activeSortIndex}
                                handleHeaderClick={handleHeaderClick}
                                onClose={onClose}
                                searchOptions={searchOptions}
                                searchResults={sortedRows}
                            />
                        )}
                    </TabContent>
                ))}
            </section>
        </div>
    );
    /* eslint-enable no-nested-ternary */
}

type SearchResultsTableProps = {
    activeSortDirection: 'asc' | 'desc';
    activeSortIndex: number;
    handleHeaderClick: (_event, index, direction) => void;
    onClose: (toURL: string) => void;
    searchOptions: SearchEntry[];
    searchResults: SearchResult[];
};

function SearchResultsTable({
    activeSortDirection,
    activeSortIndex,
    handleHeaderClick,
    onClose,
    searchOptions,
    searchResults,
}: SearchResultsTableProps): ReactElement {
    return (
        <TableComposable aria-label="Matches" variant="compact" isStickyHeader>
            <Thead>
                <Tr>
                    <Th
                        width={25}
                        sort={{
                            sortBy: {
                                index: activeSortIndex,
                                direction: activeSortDirection,
                            },
                            onSort: handleHeaderClick,
                            columnIndex: 0,
                        }}
                    >
                        Name
                    </Th>
                    <Th
                        width={25}
                        sort={{
                            sortBy: {
                                index: activeSortIndex,
                                direction: activeSortDirection,
                            },
                            onSort: handleHeaderClick,
                            columnIndex: 1,
                        }}
                    >
                        Type
                    </Th>
                    <Th>View On:</Th>
                    <Th>Filter On:</Th>
                </Tr>
            </Thead>
            <Tbody>
                {searchResults.map(({ category, id, location, name }) => {
                    return (
                        <Tr key={id}>
                            <Td dataLabel="Name">
                                {name}
                                {!!location?.length && (
                                    <div
                                        aria-label="Location"
                                        className="pf-u-color-400 pf-u-font-size-sm"
                                    >
                                        <em>{location}</em>
                                    </div>
                                )}
                            </Td>
                            <Td dataLabel="Type">{capitalize(category)}</Td>
                            <Td dataLabel="View On:">
                                <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                                    <ViewLinks
                                        id={id}
                                        onClose={onClose}
                                        searchCategory={category}
                                    />
                                </Flex>
                            </Td>
                            <Td dataLabel="Filter On:">
                                <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                                    <FilterLinks
                                        filterValue={name}
                                        globalSearchOptions={searchOptions}
                                        onClose={onClose}
                                        searchCategory={category}
                                    />
                                </Flex>
                            </Td>
                        </Tr>
                    );
                })}
            </Tbody>
        </TableComposable>
    );
}

function NotApplicable(): ReactElement {
    return (
        <FlexItem>
            <Button variant="tertiary" isSmall isDisabled>
                N/A
            </Button>
        </FlexItem>
    );
}

type ViewLinksProps = {
    id: string;
    onClose: (linkPath: string) => void;
    searchCategory: SearchCategory;
};

function ViewLinks({ id, onClose, searchCategory }: ViewLinksProps): ReactElement {
    const searchCategoryDescriptor = searchCategoryDescriptorMap[searchCategory];

    if (searchCategoryDescriptor) {
        const { viewOn } = searchCategoryDescriptor;

        if (viewOn.length !== 0) {
            return (
                <>
                    {viewOn.map(({ basePath, linkText }) => (
                        <FlexItem key={linkText}>
                            <Button
                                variant="tertiary"
                                isSmall
                                onClick={() => {
                                    onClose(id ? `${basePath}/${id}` : basePath);
                                }}
                            >
                                {linkText}
                            </Button>
                        </FlexItem>
                    ))}
                </>
            );
        }
    }

    return <NotApplicable />;
}

type FilterLinksProps = {
    filterValue: string;
    globalSearchOptions: SearchEntry[];
    onClose: (linkPath: string) => void;
    searchCategory: SearchCategory;
};

function FilterLinks({
    filterValue,
    globalSearchOptions,
    onClose,
    searchCategory,
}: FilterLinksProps): ReactElement {
    const searchCategoryDescriptor = searchCategoryDescriptorMap[searchCategory];

    if (searchCategoryDescriptor) {
        const { filterCategory, filterOn } = searchCategoryDescriptor;

        if (filterOn.length !== 0) {
            const searchOptions: SearchEntry[] = filterValue
                ? [
                      ...globalSearchOptions,
                      {
                          value: filterCategory,
                          label: filterCategory,
                          type: 'categoryOption',
                      },
                      {
                          value: filterValue,
                          label: filterValue,
                      },
                  ]
                : globalSearchOptions;
            const searchFilter = searchOptionsToSearchFilter(searchOptions);
            const queryString = getUrlQueryStringForSearchFilter(searchFilter);

            return (
                <>
                    {filterOn.map(({ basePath, linkText }) => (
                        <FlexItem key={linkText}>
                            <Button
                                variant="tertiary"
                                isSmall
                                onClick={() => {
                                    onClose(`${basePath}?${queryString}`);
                                }}
                            >
                                {linkText}
                            </Button>
                        </FlexItem>
                    ))}
                </>
            );
        }
    }

    return <NotApplicable />;
}

const mapStateToProps = createStructuredSelector({
    searchCategory: selectors.getGlobalSearchCategory,
    searchCounts: selectors.getGlobalSearchCounts,
    searchResults: selectors.getGlobalSearchResults,
    searchOptions: selectors.getGlobalSearchOptions,
});

const mapDispatchToProps = (dispatch) => ({
    setSearchCategory: (category) =>
        // eslint-disable-next-line @typescript-eslint/no-unsafe-return
        dispatch(globalSearchActions.setGlobalSearchCategory(category)),
});

export default connect<StateProps, DispatchProps, PassedProps>(
    mapStateToProps,
    mapDispatchToProps
)(SearchResults);
