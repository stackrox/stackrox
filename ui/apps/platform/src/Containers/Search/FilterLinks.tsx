import React, { ReactElement } from 'react';
import { Link } from 'react-router-dom';
import { Flex, FlexItem } from '@patternfly/react-core';

import { SearchResultCategory } from 'services/SearchService';
import { SearchFilter } from 'types/search';
import { getUrlQueryStringForSearchFilter } from 'utils/searchUtils';

import NotApplicable from './NotApplicable';
import { SearchResultCategoryMap } from './searchCategories';

type FilterLinksProps = {
    filterValue: string;
    resultCategory: SearchResultCategory;
    searchFilter: SearchFilter;
    searchResultCategoryMap: SearchResultCategoryMap;
};

function FilterLinks({
    filterValue,
    resultCategory,
    searchFilter,
    searchResultCategoryMap,
}: FilterLinksProps): ReactElement {
    const { filterOn } = searchResultCategoryMap[resultCategory] ?? {};

    if (filterOn) {
        const { filterCategory, filterLinks } = filterOn;

        const queryString = getUrlQueryStringForSearchFilter({
            ...searchFilter,
            [filterCategory]: filterValue,
        });

        return (
            <Flex spaceItems={{ default: 'spaceItemsMd' }}>
                {filterLinks.map(({ basePath, linkText }) => (
                    <FlexItem key={linkText}>
                        <Link to={`${basePath}?${queryString}`}>{linkText}</Link>
                    </FlexItem>
                ))}
            </Flex>
        );
    }

    return <NotApplicable />;
}

export default FilterLinks;
