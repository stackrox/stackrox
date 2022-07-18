import React, { ReactElement, useEffect, useState } from 'react';
import capitalize from 'lodash/capitalize';
import lowerCase from 'lodash/lowerCase';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import {
    Bullseye,
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
import { SearchEntry } from 'types/search';
import { SortDirection } from 'types/table';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import RelatedLink from './RelatedLink';

type GlobalSearchResult = {
    id: string;
    name: string;
    category: string;
    fieldToMatch: Record<string, unknown>;
    score: number;
    location: string;
};

type SearchTab = {
    text: string;
    category: string;
    disabled: boolean;
};
interface StateProps {
    globalSearchResults: GlobalSearchResult[];
    globalSearchOptions: SearchEntry[];
    tabs: SearchTab[];
    defaultTab: SearchTab | null;
}

interface DispatchProps {
    setGlobalSearchCategory: (category: string) => void;
    passthroughGlobalSearchOptions: (searchOptions: SearchEntry[], category: string) => void;
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
    tabs,
    defaultTab = null,
}: SearchResultsProps): ReactElement {
    // index of the currently active column
    const [activeSortIndex, setActiveSortIndex] = useState(INITIAL_SORT_INDEX);
    // sort direction of the currently active column
    const [activeSortDirection, setActiveSortDirection] =
        useState<SortDirection>(INITIAL_SORT_DIRECTION);
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

    function handleHeaderClick(_event, index, direction) {
        const updatedRows = onSort(sortedRows, index, direction);
        setSortedRows(updatedRows);
    }

    function onTabClick(_event, eventKey) {
        const selectedTab = defaultTabs[eventKey];
        setGlobalSearchCategory(selectedTab.category);
    }

    const amendSearchOptions = (searchCategory: string, name: string): SearchEntry[] => {
        if (name) {
            const searchModifier = `${mapping[searchCategory].name as string}:`;
            return [
                ...globalSearchOptions,
                {
                    value: searchModifier,
                    label: searchModifier,
                    type: 'categoryOption',
                },
                {
                    value: name,
                    label: name,
                    className: 'Select-create-option-placeholder',
                } as SearchEntry,
            ];
        }
        return [...globalSearchOptions];
    };

    const onFilterLinkHandler =
        (searchCategory: string, category: string, toURL: string, name: string) => () => {
            const searchOptions = amendSearchOptions(searchCategory, name);
            const searchFilter = searchOptionsToSearchFilter(searchOptions);
            const queryString = getUrlQueryStringForSearchFilter(searchFilter);
            onClose(`${toURL}?${queryString}`);
        };

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
                            <Td
                                key="resourceViewOn"
                                dataLabel="View On:"
                                data-testid="resourceViewOn"
                            >
                                <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                                    {!mapping[category]?.viewOn ? (
                                        <FlexItem key="na">
                                            <RelatedLink data-testid="view-on-label-chip" id={id}>
                                                N/A
                                            </RelatedLink>
                                        </FlexItem>
                                    ) : (
                                        mapping[category].viewOn.map((item) => (
                                            <FlexItem key={item}>
                                                <RelatedLink
                                                    data-testid="view-on-label-chip"
                                                    id={id}
                                                    onClick={() => onClose(getLink(item, id))}
                                                >
                                                    {item}
                                                </RelatedLink>
                                            </FlexItem>
                                        ))
                                    )}
                                </Flex>
                            </Td>
                            <Td
                                key="resourceFilterOn"
                                dataLabel="Filter On:"
                                data-testid="resourceFilterOn"
                            >
                                <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                                    {!mapping[category]?.filterOn ? (
                                        <FlexItem key="na">
                                            <RelatedLink data-testid="view-on-label-chip" id={id}>
                                                N/A
                                            </RelatedLink>
                                        </FlexItem>
                                    ) : (
                                        mapping[category].filterOn.map((item) => (
                                            <FlexItem key={item}>
                                                <RelatedLink
                                                    data-testid="filter-on-label-chip"
                                                    id={id}
                                                    onClick={onFilterLinkHandler(
                                                        category,
                                                        filterOnMapping[item],
                                                        getLink(item),
                                                        name
                                                    )}
                                                >
                                                    {item}
                                                </RelatedLink>
                                            </FlexItem>
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
                            key={tab.category || tab.text}
                            eventKey={index}
                            title={<TabTitleText>{tab.text}</TabTitleText>}
                        />
                    ))}
                </Tabs>
                {tabs.map((tab, index) => (
                    <TabContent
                        eventKey={index}
                        className="overflow-auto"
                        id={tab.category || tab.text}
                        aria-label={tab.text}
                        key={tab.category || tab.text}
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
