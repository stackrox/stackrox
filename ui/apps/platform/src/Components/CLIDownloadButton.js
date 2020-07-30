import React from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import { actions as CLIDownloadActions } from 'reducers/cli';

import * as Icon from 'react-feather';
import Tooltip from 'Components/Tooltip';
import TooltipOverlay from 'Components/TooltipOverlay';

const CLIDownloadButton = ({
    toggleCLIDownloadView,
    topNavBtnTextClass,
    topNavBtnSvgClass,
    topNavBtnClass,
}) => (
    <Tooltip content={<TooltipOverlay>CLI</TooltipOverlay>} className="sm:visible md:invisible">
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
    toggleCLIDownloadView: PropTypes.func.isRequired,
    topNavBtnTextClass: PropTypes.string.isRequired,
    topNavBtnSvgClass: PropTypes.string.isRequired,
    topNavBtnClass: PropTypes.string.isRequired,
};

const mapDispatchToProps = (dispatch) => ({
    toggleCLIDownloadView: () => dispatch(CLIDownloadActions.toggleCLIDownloadView()),
});

export default withRouter(connect(null, mapDispatchToProps)(CLIDownloadButton));
