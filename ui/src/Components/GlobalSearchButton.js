import React from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import { actions as globalSearchActions } from 'reducers/globalSearch';

import * as Icon from 'react-feather';
import Tooltip from 'rc-tooltip';
import 'rc-tooltip/assets/bootstrap.css';

const GlobalSearchButton = ({
    toggleGlobalSearchView,
    topNavBtnTextClass,
    topNavBtnSvgClass,
    topNavBtnClass
}) => (
    <Tooltip
        placement="bottom"
        overlay={<div>Search</div>}
        mouseLeaveDelay={0}
        overlayClassName="sm:visible md:invisible"
    >
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
    topNavBtnClass: PropTypes.string.isRequired
};

const mapDispatchToProps = dispatch => ({
    toggleGlobalSearchView: () => dispatch(globalSearchActions.toggleGlobalSearchView())
});

export default withRouter(
    connect(
        null,
        mapDispatchToProps
    )(GlobalSearchButton)
);
