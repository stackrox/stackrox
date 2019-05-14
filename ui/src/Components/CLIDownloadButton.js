import React from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import { actions as CLIDownloadActions } from 'reducers/cli';

import * as Icon from 'react-feather';
import Tooltip from 'rc-tooltip';
import 'rc-tooltip/assets/bootstrap.css';
import {
    topNavBtnTextClass,
    topNavBtnSvgClass,
    topNavBtnClass
} from 'Containers/Navigation/TopNavigation';

const CLIDownloadButton = ({ toggleCLIDownloadView }) => (
    <Tooltip
        placement="bottom"
        overlay={<div>CLI</div>}
        mouseLeaveDelay={0}
        overlayClassName="sm:visible md:invisible"
    >
        <button
            type="button"
            onClick={toggleCLIDownloadView}
            className={`${topNavBtnClass} ignore-cli-clickoutside`}
        >
            <Icon.Download className={topNavBtnSvgClass} />
            <span className={topNavBtnTextClass}>CLI</span>
        </button>
    </Tooltip>
);

CLIDownloadButton.propTypes = {
    toggleCLIDownloadView: PropTypes.func.isRequired
};

const mapDispatchToProps = dispatch => ({
    toggleCLIDownloadView: () => dispatch(CLIDownloadActions.toggleCLIDownloadView())
});

export default withRouter(
    connect(
        null,
        mapDispatchToProps
    )(CLIDownloadButton)
);
