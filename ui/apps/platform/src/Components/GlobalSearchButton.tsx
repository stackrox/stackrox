import React, { ReactElement } from 'react';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import { SearchIcon } from '@patternfly/react-icons';
import { Tooltip, TooltipOverlay } from '@stackrox/ui-components';

import { actions as globalSearchActions } from 'reducers/globalSearch';

type GlobalSearchButtonProps = {
    toggleGlobalSearchView: () => void;
    topNavBtnClass: string;
};

const GlobalSearchButton = ({
    toggleGlobalSearchView,
    topNavBtnClass,
}: GlobalSearchButtonProps): ReactElement => (
    <Tooltip content={<TooltipOverlay>Search</TooltipOverlay>} className="sm:visible md:invisible">
        <button
            type="button"
            onClick={toggleGlobalSearchView}
            className={`${topNavBtnClass} ignore-react-onclickoutside`}
        >
            <SearchIcon alt="" />
            <span className="ml-2">Search</span>
        </button>
    </Tooltip>
);

const mapDispatchToProps = (dispatch) => ({
    // TODO: type redux props
    // eslint-disable-next-line @typescript-eslint/no-unsafe-return
    toggleGlobalSearchView: () => dispatch(globalSearchActions.toggleGlobalSearchView()),
});

export default withRouter(connect(null, mapDispatchToProps)(GlobalSearchButton));
