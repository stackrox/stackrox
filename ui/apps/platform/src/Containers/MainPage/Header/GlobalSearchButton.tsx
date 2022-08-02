import React, { ReactElement } from 'react';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import { Flex, FlexItem } from '@patternfly/react-core';
import { SearchIcon } from '@patternfly/react-icons';

import { actions as globalSearchActions } from 'reducers/globalSearch';

type GlobalSearchButtonProps = {
    toggleGlobalSearchView: () => void;
};

/*
 * Use HTML button with PatternFly Button class to inherit masthead color.
 */
const GlobalSearchButton = ({ toggleGlobalSearchView }: GlobalSearchButtonProps): ReactElement => (
    <button
        type="button"
        onClick={toggleGlobalSearchView}
        className="pf-c-button ignore-react-onclickoutside"
    >
        <Flex alignItems={{ default: 'alignItemsCenter' }} spaceItems={{ default: 'spaceItemsSm' }}>
            <FlexItem>
                <SearchIcon alt="" />
            </FlexItem>
            <FlexItem>Search</FlexItem>
        </Flex>
    </button>
);

const mapDispatchToProps = (dispatch) => ({
    // TODO: type redux props
    // eslint-disable-next-line @typescript-eslint/no-unsafe-return
    toggleGlobalSearchView: () => dispatch(globalSearchActions.toggleGlobalSearchView()),
});

export default withRouter(connect(null, mapDispatchToProps)(GlobalSearchButton));
