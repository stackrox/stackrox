import React, { ReactElement } from 'react';
import { Link } from 'react-router-dom';
import { Flex, FlexItem } from '@patternfly/react-core';

import { SearchResult } from 'services/SearchService';
import { safeGeneratePath } from 'utils/urlUtils';

import NotApplicable from './NotApplicable';
import { SearchResultCategoryMap } from './searchCategories';

type ViewLinksProps = {
    searchResult: SearchResult & {
        locationTextForCategory: string;
    };
    searchResultCategoryMap: SearchResultCategoryMap;
};

function ViewLinks({ searchResult, searchResultCategoryMap }: ViewLinksProps): ReactElement {
    const { viewLinks } = searchResultCategoryMap[searchResult.category] ?? {};

    if (viewLinks?.length) {
        return (
            <Flex spaceItems={{ default: 'spaceItemsMd' }}>
                {viewLinks.map(({ basePath, linkText }) => (
                    <FlexItem key={linkText}>
                        <Link
                            to={safeGeneratePath(basePath, searchResult, basePath)}
                            className="pf-v5-u-text-nowrap"
                        >
                            {linkText}
                        </Link>
                    </FlexItem>
                ))}
            </Flex>
        );
    }

    return <NotApplicable />;
}

export default ViewLinks;
