import React, { ReactElement } from 'react';
import { Link } from 'react-router-dom';
import { Flex, FlexItem, Nav, NavItem, NavList, Split, SplitItem } from '@patternfly/react-core';

import usePermissions from 'hooks/usePermissions';
import { SearchResponse } from 'services/SearchService';
import { SearchFilter } from 'types/search';
import { searchPath } from 'routePaths';

import SearchTable from './SearchTable';
import { SearchTabCategory, searchResultCategoryMap, searchTabMap } from './searchCategories';
import { stringifyQueryObject } from './searchQuery';

type SearchTabsProps = {
    activeTabCategory: SearchTabCategory;
    searchFilter: SearchFilter;
    searchResponse: SearchResponse;
};

function SearchTabs({
    activeTabCategory,
    searchFilter,
    searchResponse,
}: SearchTabsProps): ReactElement {
    const { hasReadAccess } = usePermissions();

    const { counts, results } = searchResponse;

    function getTabCategoryCount(tabCategory: SearchTabCategory) {
        return tabCategory === 'SEARCH_UNSET'
            ? results.length
            : counts.find(({ category }) => category === tabCategory)?.count ?? 0;
    }

    const searchTabEntriesFiltered = Object.entries(searchTabMap).filter(
        ([tabCategory]) =>
            tabCategory === 'SEARCH_UNSET' ||
            hasReadAccess(searchResultCategoryMap[tabCategory].resourceName)
    );

    return (
        <Split hasGutter>
            <SplitItem>
                <Nav aria-label="Categories" theme="light">
                    <NavList>
                        {searchTabEntriesFiltered.map(([tabCategory, text]) => (
                            <NavItem key={tabCategory} isActive={tabCategory === activeTabCategory}>
                                <Link
                                    to={`${searchPath}${stringifyQueryObject({
                                        searchFilter,
                                        tabCategory: tabCategory as SearchTabCategory,
                                    })}`}
                                    replace
                                >
                                    <Flex
                                        justifyContent={{ default: 'justifyContentSpaceBetween' }}
                                        style={{ minWidth: '13em' }}
                                    >
                                        <FlexItem>{text}</FlexItem>
                                        <FlexItem>
                                            {getTabCategoryCount(tabCategory as SearchTabCategory)}
                                        </FlexItem>
                                    </Flex>
                                </Link>
                            </NavItem>
                        ))}
                    </NavList>
                </Nav>
            </SplitItem>
            <SplitItem isFilled>
                <SearchTable
                    searchFilter={searchFilter}
                    searchResults={results}
                    tabCategory={activeTabCategory}
                />
            </SplitItem>
        </Split>
    );
}

export default SearchTabs;
