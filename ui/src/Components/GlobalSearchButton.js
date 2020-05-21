import React from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import { actions as globalSearchActions } from 'reducers/globalSearch';

import * as Icon from 'react-feather';
import Tooltip from 'Components/Tooltip';
import TooltipOverlay from 'Components/TooltipOverlay';

const GlobalSearchButton = ({
    toggleGlobalSearchView,
    topNavBtnTextClass,
    topNavBtnSvgClass,
    topNavBtnClass,
}) => (
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

GlobalSearchButton.propTypes = {
    toggleGlobalSearchView: PropTypes.func.isRequired,
    topNavBtnTextClass: PropTypes.string.isRequired,
    topNavBtnSvgClass: PropTypes.string.isRequired,
    topNavBtnClass: PropTypes.string.isRequired,
};

const mapDispatchToProps = (dispatch) => ({
    toggleGlobalSearchView: () => dispatch(globalSearchActions.toggleGlobalSearchView()),
});

export default withRouter(connect(null, mapDispatchToProps)(GlobalSearchButton));
