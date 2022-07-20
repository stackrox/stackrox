import React, { ReactElement, useEffect, useState } from 'react';
import capitalize from 'lodash/capitalize';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import {
    Bullseye,
    Button,
    Flex,
    FlexItem,
    Tabs,
    Tab,
    TabTitleText,
    TabContent,
} from '@patternfly/react-core';
import { TableComposable, Tbody, Td, Thead, Th, Tr } from '@patternfly/react-table';

import { getUrlQueryStringForSearchFilter, searchOptionsToSearchFilter } from 'utils/searchUtils';
import { selectors } from 'reducers';
import { actions as globalSearchActions } from 'reducers/globalSearch';
import { SearchCategory, SearchResult } from 'services/SearchService';
import { SearchEntry } from 'types/search';
import { SortDirection } from 'types/table';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';

import searchCategoryDescriptorMap from './searchCategoryDescriptorMap';

type SearchTab = {
    text: string;
    category: SearchCategory;
    disabled: boolean;
};
interface StateProps {
    globalSearchResults: SearchResult[];
    globalSearchOptions: SearchEntry[];
    tabs: SearchTab[];
    defaultTab: SearchTab | null;
}

interface DispatchProps {
    setGlobalSearchCategory: (category: string) => void;
}

interface PassedProps {
    onClose: (toURL: string) => void;
}

export type SearchResultsProps = StateProps & DispatchProps & PassedProps;

const defaultTabs: SearchTab[] = [
    {
        text: 'All',
        category: 'SEARCH_UNSET',
        disabled: false,
    },
    {
        text: 'Violations',
        category: 'ALERTS',
        disabled: false,
    },
    {
        text: 'Policies',
        category: 'POLICIES',
        disabled: false,
    },
    {
        text: 'Deployments',
        category: 'DEPLOYMENTS',
        disabled: false,
    },
    {
        text: 'Images',
        category: 'IMAGES',
        disabled: false,
    },
    {
        text: 'Secrets',
        category: 'SECRETS',
        disabled: false,
    },
];

const INITIAL_SORT_INDEX = 1; // Type column
const INITIAL_SORT_DIRECTION = 'asc'; // A->Z

function SearchResults({
    onClose,
    globalSearchResults,
    globalSearchOptions,
    setGlobalSearchCategory,
    tabs,
    defaultTab = null,
}: SearchResultsProps): ReactElement {
    // index of the currently active column
    const [activeSortIndex, setActiveSortIndex] = useState(INITIAL_SORT_INDEX);
    // sort direction of the currently active column
    const [activeSortDirection, setActiveSortDirection] =
        useState<SortDirection>(INITIAL_SORT_DIRECTION);
    const [sortedRows, setSortedRows] = useState<SearchResult[]>([]);

    useEffect(() => {
        const newSortedResults = onSort(
            globalSearchResults,
            INITIAL_SORT_INDEX,
            INITIAL_SORT_DIRECTION
        );
        setSortedRows(newSortedResults);
    }, [globalSearchResults]);

    function onSort(
        currentRows: SearchResult[],
        index: number,
        direction: SortDirection
    ): SearchResult[] {
        setActiveSortIndex(index);
        setActiveSortDirection(direction);
        // sorts the rows
        const updatedRows = currentRows.sort((a, b) => {
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
        const selectedTab = defaultTabs[eventKey];
        setGlobalSearchCategory(selectedTab.category);
    }

    const contents = sortedRows.length ? (
        <TableComposable aria-label="Matches" variant="compact" isStickyHeader>
            <Thead>
                <Tr>
                    <Th
                        key="resourceName"
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
                        key="resourceType"
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
                    <Th key="resourceViewOn">View On:</Th>
                    <Th key="resourceFilterOn">Filter On:</Th>
                </Tr>
            </Thead>
            <Tbody>
                {sortedRows.map((result) => {
                    const { category, id, location, name } = result;
                    return (
                        <Tr key={id}>
                            <Td key="resourceName" dataLabel="Name">
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
                            <Td key="resourceType" dataLabel="Type" data-testid="resourceType">
                                {capitalize(category)}
                            </Td>
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
                                        globalSearchOptions={globalSearchOptions}
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
    ) : (
        <EmptyStateTemplate title="No results for your chosen filters" headingLevel="h2">
            Try changing the filter values.
        </EmptyStateTemplate>
    );

    const activeTabKey = tabs.findIndex((tab) => tab.category === defaultTab?.category) || 0;

    const renderTabs = () => {
        return (
            <section className="h-full">
                <Tabs key="tab-bar" activeKey={activeTabKey} onSelect={onTabClick}>
                    {tabs.map((tab, index) => (
                        <Tab
                            key={tab.category}
                            eventKey={index}
                            title={<TabTitleText>{tab.text}</TabTitleText>}
                        />
                    ))}
                </Tabs>
                {tabs.map((tab, index) => (
                    <TabContent
                        eventKey={index}
                        className="overflow-auto"
                        id={tab.category}
                        aria-label={tab.text}
                        key={tab.category}
                        hidden={index !== activeTabKey}
                    >
                        {contents}
                    </TabContent>
                ))}
            </section>
        );
    };

    return !globalSearchOptions.length ? (
        <Bullseye>
            <EmptyStateTemplate title="Search all data" headingLevel="h1">
                Choose one or more filter values to search.
            </EmptyStateTemplate>
        </Bullseye>
    ) : (
        <div className="bg-base-100 flex-1" data-testid="global-search-results">
            <h1 className="w-full text-2xl text-primary-700 px-4 py-6 font-600">
                {globalSearchResults.length} search results
            </h1>
            {renderTabs()}
        </div>
    );
}

const getTabs = createSelector(
    selectors.getGlobalSearchCounts,
    (globalSearchCounts: Record<string, unknown>[]) => {
        if (globalSearchCounts.length === 0) {
            return defaultTabs;
        }

        const newTabs: SearchTab[] = [];
        defaultTabs.forEach((tab: SearchTab) => {
            const newTab: SearchTab = { ...tab };
            const currentTab = globalSearchCounts.find((obj) => obj.category === tab.category);
            if (currentTab) {
                newTab.text += ` (${currentTab.count as string})`;
                if (currentTab.count === '0') {
                    newTab.disabled = true;
                }
            }
            newTabs.push(newTab);
        });
        return newTabs;
    }
);

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

const getDefaultTab = createSelector(
    [selectors.getGlobalSearchCategory],
    (globalSearchCategory) => {
        const tab = defaultTabs.find((obj) => obj.category === globalSearchCategory);
        return tab;
    }
);

const mapStateToProps = createStructuredSelector({
    globalSearchResults: selectors.getGlobalSearchResults,
    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
    // @ts-ignore
    globalSearchOptions: selectors.getGlobalSearchOptions,
    tabs: getTabs,
    defaultTab: getDefaultTab,
});

const mapDispatchToProps = (dispatch) => ({
    setGlobalSearchCategory: (category) =>
        // TODO: type redux props
        // eslint-disable-next-line @typescript-eslint/no-unsafe-return
        dispatch(globalSearchActions.setGlobalSearchCategory(category)),
});

export default connect<StateProps, DispatchProps, PassedProps>(
    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
    // @ts-ignore
    mapStateToProps,
    mapDispatchToProps
)(SearchResults);
