import React, { ReactElement } from 'react';
import { Link } from 'react-router-dom';
import { Flex, FlexItem } from '@patternfly/react-core';
import { SearchIcon } from '@patternfly/react-icons';

import { searchPath } from 'routePaths';

/*
 * Use React Router Link element with PatternFly Button class to inherit masthead color.
 */
function GlobalSearchButton(): ReactElement {
    return (
        <Link to={searchPath}>
            <Flex
                alignItems={{ default: 'alignItemsCenter' }}
                spaceItems={{ default: 'spaceItemsSm' }}
            >
                <FlexItem>
                    <SearchIcon alt="" />
                </FlexItem>
                <FlexItem>Search</FlexItem>
            </Flex>
        </Link>
    );
}

export default GlobalSearchButton;
