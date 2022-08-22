import React, { ReactElement } from 'react';
import { Button, Flex, FlexItem } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';
import { SearchResultCategory } from 'services/SearchService';
import { SearchFilter } from 'types/search';
import { getUrlQueryStringForSearchFilter } from 'utils/searchUtils';

import NotApplicable from './NotApplicable';
import { searchResultCategoryMap } from './searchCategories';

type FilterLinksProps = {
    filterValue: string;
    resultCategory: SearchResultCategory;
    searchFilter: SearchFilter;
};

function FilterLinks({
    filterValue,
    resultCategory,
    searchFilter,
}: FilterLinksProps): ReactElement {
    const { filterOn } = searchResultCategoryMap[resultCategory];

    if (filterOn !== null) {
        const { filterCategory, filterLinks } = filterOn;

        const queryString = getUrlQueryStringForSearchFilter({
            ...searchFilter,
            [filterCategory]: filterValue,
        });

        return (
            <Flex spaceItems={{ default: 'spaceItemsMd' }}>
                {filterLinks.map(({ basePath, linkText }) => (
                    <FlexItem key={linkText}>
                        <Button
                            variant="link"
                            isInline
                            component={LinkShim}
                            href={`${basePath}?${queryString}`}
                        >
                            {linkText}
                        </Button>
                    </FlexItem>
                ))}
            </Flex>
        );
    }

    return <NotApplicable />;
}

export default FilterLinks;
