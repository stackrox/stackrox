import React from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import { actions as globalSearchActions } from 'reducers/globalSearch';

import * as Icon from 'react-feather';
import Tooltip from 'rc-tooltip';
import 'rc-tooltip/assets/bootstrap.css';
import {
    topNavBtnTextClass,
    topNavBtnSvgClass,
    topNavBtnClass
} from 'Containers/Navigation/TopNavigation';

const GlobalSearchButton = ({ toggleGlobalSearchView }) => (
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
    toggleGlobalSearchView: PropTypes.func.isRequired
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
