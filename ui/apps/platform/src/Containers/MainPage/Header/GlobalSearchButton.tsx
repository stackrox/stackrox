import React, { ReactElement } from 'react';
import { useDispatch } from 'react-redux';
import { useHistory } from 'react-router-dom';
import { Flex, FlexItem } from '@patternfly/react-core';
import { SearchIcon } from '@patternfly/react-icons';

import useFeatureFlagEnabled from 'hooks/useFeatureFlags';
import { actions as globalSearchActions } from 'reducers/globalSearch';
import { searchPath } from 'routePaths';

/*
 * Use HTML button with PatternFly Button class to inherit masthead color.
 */
function GlobalSearchButton(): ReactElement {
    const dispatch = useDispatch();
    const history = useHistory();
    const { isFeatureFlagEnabled } = useFeatureFlagEnabled();

    function onClick() {
        if (isFeatureFlagEnabled('ROX_SEARCH_PAGE_UI')) {
            history.push(searchPath);
        } else {
            dispatch(globalSearchActions.toggleGlobalSearchView());
        }
    }

    return (
        <button type="button" onClick={onClick} className="pf-c-button ignore-react-onclickoutside">
            <Flex
                alignItems={{ default: 'alignItemsCenter' }}
                spaceItems={{ default: 'spaceItemsSm' }}
            >
                <FlexItem>
                    <SearchIcon alt="" />
                </FlexItem>
                <FlexItem>Search</FlexItem>
            </Flex>
        </button>
    );
}

export default GlobalSearchButton;
