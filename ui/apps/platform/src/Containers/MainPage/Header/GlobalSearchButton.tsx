import React, { ReactElement } from 'react';
import { Link } from 'react-router-dom';
import { Flex, FlexItem } from '@patternfly/react-core';
import { SearchIcon } from '@patternfly/react-icons';

import { searchPath } from 'routePaths';

/*
 * React Router Link element with style rule in src/css/acs.css to inherit masthead color.
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
