import React, { ReactElement, useEffect, useState } from 'react';
import capitalize from 'lodash/capitalize';
import lowerCase from 'lodash/lowerCase';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import { Bullseye, Flex } from '@patternfly/react-core';
import { TableComposable, Tbody, Td, Thead, Th, Tr } from '@patternfly/react-table';

import { addSearchModifier, addSearchKeyword } from 'utils/searchUtils';
import { selectors } from 'reducers';
import { actions as globalSearchActions } from 'reducers/globalSearch';
import Tabs from 'Components/Tabs';
import Tab from 'Components/Tab';
import EmptyGlobalSearch from './EmptyGlobalSearch';
import RelatedLink from './RelatedLink';

type GlobalSearchResult = {
    id: string;
    name: string;
    category: string;
    fieldToMatch: Record<string, unknown>;
    score: number;
    location: string;
};

type GlobalSearchOption = {
    value: string;
    label: string;
    type?: string;
};

type SearchTab = {
    text: string;
    category: string;
    disabled: boolean;
};
interface StateProps {
    globalSearchResults: GlobalSearchResult[];
    globalSearchOptions: GlobalSearchOption[];
    tabs: SearchTab[];
    defaultTab: SearchTab | null;
}

type SortDirection = 'asc' | 'desc' | undefined;

interface DispatchProps {
    setGlobalSearchCategory: (category: string) => void;
    passthroughGlobalSearchOptions: (searchOptions: GlobalSearchOption[], category: string) => void;
}

interface PassedProps {
    onClose: (toURL: string) => void;
}

export type SearchResultsProps = StateProps & DispatchProps & PassedProps;

const defaultTabs = [
    {
        text: 'All',
        category: '',
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

const mapping = {
    IMAGES: {
        filterOn: ['RISK', 'VIOLATIONS'],
        viewOn: ['IMAGES'],
        name: 'Image',
    },
    DEPLOYMENTS: {
        filterOn: ['VIOLATIONS', 'NETWORK'],
        viewOn: ['RISK'],
        name: 'Deployment',
    },
    POLICIES: {
        filterOn: ['VIOLATIONS'],
        viewOn: ['POLICIES'],
        name: 'Policy',
    },
    ALERTS: {
        filterOn: [],
        viewOn: ['VIOLATIONS'],
        name: 'Policy',
    },
    SECRETS: {
        filterOn: ['RISK'],
        viewOn: ['SECRETS'],
        name: 'Secret',
    },
};

const filterOnMapping = {
    RISK: 'DEPLOYMENTS',
    VIOLATIONS: 'ALERTS',
    NETWORK: 'NETWORK',
};

const getLink = (item: string, id?: string) => {
    let link = '/main';
    if (item === 'SECRETS') {
        link = `${link}/configmanagement`;
    } else if (item === 'IMAGES') {
        link = `${link}/vulnerability-management`;
    }
    return `${link}/${lowerCase(item)}${id ? `/${id}` : ''}`;
};

const INITIAL_SORT_INDEX = 1; // Type column
const INITIAL_SORT_DIRECTION = 'asc'; // A->Z

function SearchResults({
    onClose,
    globalSearchResults,
    globalSearchOptions,
    setGlobalSearchCategory,
    passthroughGlobalSearchOptions,
    tabs,
    defaultTab = null,
}: SearchResultsProps): ReactElement {
    // index of the currently active column
    const [activeSortIndex, setActiveSortIndex] = useState(INITIAL_SORT_INDEX);
    // sort direction of the currently active column
    const [activeSortDirection, setActiveSortDirection] = useState<SortDirection>(
        INITIAL_SORT_DIRECTION
    );
    const [sortedRows, setSortedRows] = useState<GlobalSearchResult[]>([]);

    useEffect(() => {
        const newSortedResults = onSort(
            globalSearchResults,
            INITIAL_SORT_INDEX,
            INITIAL_SORT_DIRECTION
        );
        setSortedRows(newSortedResults);
    }, [globalSearchResults]);

    function onSort(
        currentRows: GlobalSearchResult[],
        index: number,
        direction: SortDirection
    ): GlobalSearchResult[] {
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

    function handleHeaderClick(event, index, direction) {
        const updatedRows = onSort(sortedRows, index, direction);
        setSortedRows(updatedRows);
    }

    function onTabClick(tab: SearchTab) {
        setGlobalSearchCategory(tab.category);
    }

    const onLinkHandler = (
        searchCategory: string,
        category: string,
        toURL: string,
        name: string
    ) => () => {
        let searchOptions = globalSearchOptions.slice();
        if (name) {
            searchOptions = addSearchModifier(
                searchOptions,
                `${mapping[searchCategory].name as string}:`
            );
            searchOptions = addSearchKeyword(searchOptions, name);
        }
        passthroughGlobalSearchOptions(searchOptions, category);
        onClose(toURL);
    };

    const contents = sortedRows.length ? (
        <TableComposable aria-label="Matches" variant="compact">
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
                            <Td
                                key="resourceViewOn"
                                dataLabel="View On:"
                                data-testid="resourceViewOn"
                            >
                                <Flex>
                                    {!mapping[category]?.viewOn ? (
                                        <RelatedLink data-testid="view-on-label-chip" id={id}>
                                            N/A
                                        </RelatedLink>
                                    ) : (
                                        mapping[category].viewOn.map((item) => (
                                            <RelatedLink
                                                data-testid="view-on-label-chip"
                                                id={id}
                                                onClick={onLinkHandler(
                                                    category,
                                                    item,
                                                    getLink(item, id),
                                                    name
                                                )}
                                            >
                                                {item}
                                            </RelatedLink>
                                        ))
                                    )}
                                </Flex>
                            </Td>
                            <Td
                                key="resourceFilterOn"
                                dataLabel="Filter On:"
                                data-testid="resourceFilterOn"
                            >
                                <Flex>
                                    {!mapping[category]?.filterOn ? (
                                        <RelatedLink data-testid="view-on-label-chip" id={id}>
                                            N/A
                                        </RelatedLink>
                                    ) : (
                                        mapping[category].filterOn.map((item) => (
                                            <RelatedLink
                                                data-testid="filter-on-label-chip"
                                                id={id}
                                                onClick={onLinkHandler(
                                                    category,
                                                    filterOnMapping[item],
                                                    getLink(item),
                                                    name
                                                )}
                                            >
                                                {item}
                                            </RelatedLink>
                                        ))
                                    )}
                                </Flex>
                            </Td>
                        </Tr>
                    );
                })}
            </Tbody>
        </TableComposable>
    ) : (
        <EmptyGlobalSearch title="No results for your chosen filters">
            Try changing the filter values.
        </EmptyGlobalSearch>
    );

    const renderTabs = () => {
        return (
            <section className="flex flex-auto h-full">
                <div className="flex flex-1">
                    <Tabs
                        className="bg-base-100 mb-8"
                        headers={tabs}
                        onTabClick={onTabClick}
                        default={defaultTab}
                        tabClass="tab flex-1 items-center justify-center font-700 p-3 uppercase shadow-none hover:text-primary-600 border-b-2 border-transparent"
                        tabActiveClass="tab flex-1 items-center justify-center border-b-2 p-3 border-primary-400 shadow-none font-700 text-primary-700 uppercase"
                        tabDisabledClass="tab flex-1 items-center justify-center border-2 border-transparent p-3 font-700 disabled shadow-none uppercase"
                        tabContentBgColor="bg-base-100"
                    >
                        {tabs.map((tab) => (
                            <Tab key={tab.text}>{contents}</Tab>
                        ))}
                    </Tabs>
                </div>
            </section>
        );
    };

    return !globalSearchOptions.length ? (
        <Bullseye className="pf-u-background-color-100">
            <EmptyGlobalSearch title="Search all data across Advanced Cluster Security">
                Choose one or more filter values to search.
            </EmptyGlobalSearch>
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
    passthroughGlobalSearchOptions: (searchOptions, category) =>
        // TODO: type redux props
        // eslint-disable-next-line @typescript-eslint/no-unsafe-return
        dispatch(globalSearchActions.passthroughGlobalSearchOptions(searchOptions, category)),
});

export default connect<StateProps, DispatchProps, PassedProps>(
    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
    // @ts-ignore
    mapStateToProps,
    mapDispatchToProps
)(SearchResults);
