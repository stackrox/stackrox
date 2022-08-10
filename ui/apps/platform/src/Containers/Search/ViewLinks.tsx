import React, { ReactElement } from 'react';
import { Button, Flex, FlexItem } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';
import { SearchResultCategory } from 'services/SearchService';

import NotApplicable from './NotApplicable';
import { searchResultCategoryMap } from './searchCategories';

type ViewLinksProps = {
    id: string;
    resultCategory: SearchResultCategory;
};

function ViewLinks({ id, resultCategory }: ViewLinksProps): ReactElement {
    const { viewLinks } = searchResultCategoryMap[resultCategory];

    if (viewLinks.length !== 0) {
        return (
            <Flex spaceItems={{ default: 'spaceItemsMd' }}>
                {viewLinks.map(({ basePath, linkText }) => (
                    <FlexItem key={linkText}>
                        <Button
                            variant="link"
                            isInline
                            component={LinkShim}
                            href={id ? `${basePath}/${id}` : basePath}
                            className="pf-u-text-nowrap"
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

export default ViewLinks;
