import type { ReactElement } from 'react';
import { Link } from 'react-router-dom-v5-compat';
import { Flex, FlexItem } from '@patternfly/react-core';

import type { SearchResultCategory } from 'services/SearchService';
import type { SearchFilter } from 'types/search';
import { getUrlQueryStringForSearchFilter } from 'utils/searchUtils';

import NotApplicable from './NotApplicable';
import type { SearchResultCategoryMap } from './searchCategories';

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
                {filterLinks.map(({ basePath, linkText, searchParams }) => {
                    const extra = searchParams ? `${searchParams}&` : '';
                    const separator = basePath.includes('?') ? '&' : '?';
                    return (
                        <FlexItem key={linkText}>
                            <Link to={`${basePath}${separator}${extra}${queryString}`}>
                                {linkText}
                            </Link>
                        </FlexItem>
                    );
                })}
            </Flex>
        );
    }

    return <NotApplicable />;
}

export default FilterLinks;
