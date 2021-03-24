import React, { ReactElement } from 'react';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import * as Icon from 'react-feather';
import { Tooltip, TooltipOverlay } from '@stackrox/ui-components';

import { actions as globalSearchActions } from 'reducers/globalSearch';

type GlobalSearchButtonProps = {
    toggleGlobalSearchView: () => void;
    topNavBtnTextClass: string;
    topNavBtnSvgClass: string;
    topNavBtnClass: string;
};

const GlobalSearchButton = ({
    toggleGlobalSearchView,
    topNavBtnTextClass,
    topNavBtnSvgClass,
    topNavBtnClass,
}: GlobalSearchButtonProps): ReactElement => (
    <Tooltip content={<TooltipOverlay>Search</TooltipOverlay>} className="sm:visible md:invisible">
        <button
            type="button"
            onClick={toggleGlobalSearchView}
            className={`${topNavBtnClass} border-l border-r border-base-400 ignore-react-onclickoutside`}
        >
            <Icon.Search className={topNavBtnSvgClass} />
            <span className={topNavBtnTextClass}>Search</span>
        </button>
    </Tooltip>
);

const mapDispatchToProps = (dispatch) => ({
    // TODO: type redux props
    // eslint-disable-next-line @typescript-eslint/no-unsafe-return
    toggleGlobalSearchView: () => dispatch(globalSearchActions.toggleGlobalSearchView()),
});

export default withRouter(connect(null, mapDispatchToProps)(GlobalSearchButton));
