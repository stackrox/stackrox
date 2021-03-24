import React, { ReactElement } from 'react';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import * as Icon from 'react-feather';
import { Tooltip, TooltipOverlay } from '@stackrox/ui-components';

import { actions as CLIDownloadActions } from 'reducers/cli';

type CLIDownloadButtonProps = {
    toggleCLIDownloadView: () => void;
    topNavBtnTextClass: string;
    topNavBtnSvgClass: string;
    topNavBtnClass: string;
};

const CLIDownloadButton = ({
    toggleCLIDownloadView,
    topNavBtnTextClass,
    topNavBtnSvgClass,
    topNavBtnClass,
}: CLIDownloadButtonProps): ReactElement => (
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

const mapDispatchToProps = (dispatch) => ({
    // TODO: type redux props
    // eslint-disable-next-line @typescript-eslint/no-unsafe-return
    toggleCLIDownloadView: () => dispatch(CLIDownloadActions.toggleCLIDownloadView()),
});

export default withRouter(connect(null, mapDispatchToProps)(CLIDownloadButton));
