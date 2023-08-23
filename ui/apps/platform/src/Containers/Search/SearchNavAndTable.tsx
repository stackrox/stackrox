import React, { ReactElement } from 'react';
import { Link } from 'react-router-dom';
import { Flex, FlexItem, Nav, NavItem, NavList, Split, SplitItem } from '@patternfly/react-core';

import { SearchResponse } from 'services/SearchService';
import { SearchFilter } from 'types/search';
import { searchPath } from 'routePaths';

import SearchTable from './SearchTable';
import { SearchNavCategory, searchNavMap } from './searchCategories';
import { stringifyQueryObject } from './searchQuery';

type SearchNavAndTableProps = {
    activeNavCategory: SearchNavCategory;
    searchFilter: SearchFilter;
    searchResponse: SearchResponse;
};

function SearchNavAndTable({
    activeNavCategory,
    searchFilter,
    searchResponse,
}: SearchNavAndTableProps): ReactElement {
    const { counts, results } = searchResponse;

    function getNavCategoryCount(navCategory: SearchNavCategory) {
        return navCategory === 'SEARCH_UNSET'
            ? results.length
            : counts.find(({ category }) => category === navCategory)?.count ?? 0;
    }

    return (
        <Split hasGutter>
            <SplitItem>
                <Nav aria-label="Categories" theme="light">
                    <NavList>
                        {Object.entries(searchNavMap).map(([navCategory, text]) => (
                            <NavItem key={navCategory} isActive={navCategory === activeNavCategory}>
                                <Link
                                    to={`${searchPath}${stringifyQueryObject({
                                        searchFilter,
                                        navCategory: navCategory as SearchNavCategory,
                                    })}`}
                                    replace
                                >
                                    <Flex
                                        justifyContent={{ default: 'justifyContentSpaceBetween' }}
                                        style={{ minWidth: '13em' }}
                                    >
                                        <FlexItem>{text}</FlexItem>
                                        <FlexItem>
                                            {getNavCategoryCount(navCategory as SearchNavCategory)}
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
                    navCategory={activeNavCategory}
                />
            </SplitItem>
        </Split>
    );
}

export default SearchNavAndTable;
