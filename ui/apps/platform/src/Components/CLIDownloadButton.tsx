import React, { ReactElement } from 'react';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import { DownloadIcon } from '@patternfly/react-icons';
import { Tooltip, TooltipOverlay } from '@stackrox/ui-components';

import { actions as CLIDownloadActions } from 'reducers/cli';

type CLIDownloadButtonProps = {
    toggleCLIDownloadView: () => void;
    topNavBtnClass: string;
};

const CLIDownloadButton = ({
    toggleCLIDownloadView,
    topNavBtnClass,
}: CLIDownloadButtonProps): ReactElement => (
    <Tooltip content={<TooltipOverlay>CLI</TooltipOverlay>} className="sm:visible md:invisible">
        <button
            type="button"
            onClick={toggleCLIDownloadView}
            className={`${topNavBtnClass} ignore-cli-clickoutside`}
        >
            <DownloadIcon alt="" />
            <span className="ml-2">CLI</span>
        </button>
    </Tooltip>
);

const mapDispatchToProps = (dispatch) => ({
    // TODO: type redux props
    // eslint-disable-next-line @typescript-eslint/no-unsafe-return
    toggleCLIDownloadView: () => dispatch(CLIDownloadActions.toggleCLIDownloadView()),
});

export default withRouter(connect(null, mapDispatchToProps)(CLIDownloadButton));
