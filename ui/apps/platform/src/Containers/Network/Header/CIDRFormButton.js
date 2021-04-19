import React from 'react';
import { Box } from 'react-feather';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { actions as pageActions } from 'reducers/network/page';
import { actions as sidepanelActions } from 'reducers/network/sidepanel';
import sidepanelStages from '../SidePanel/sidepanelStages';

const CIDRFormButton = ({
    sidePanelOpen,
    sidePanelStage,
    openSidePanel,
    closeSidePanel,
    setSidePanelStage,
    isDisabled,
}) => {
    function toggleForm() {
        if (sidePanelOpen && sidePanelStage === sidepanelStages.cidrForm) {
            closeSidePanel();
        } else {
            if (!sidePanelOpen) {
                openSidePanel();
            }
            setSidePanelStage(sidepanelStages.cidrForm);
        }
    }

    return (
        <button
            type="button"
            onClick={toggleForm}
            className="flex border-l border-dashed border-base-400 items-center px-4 font-condensed uppercase justify-center hover:bg-base-300"
            disabled={isDisabled}
        >
            <Box />
            <span className="text-left pl-2">
                Configure
                <br />
                CIDR Blocks
            </span>
        </button>
    );
};

const mapStateToProps = createStructuredSelector({
    sidePanelOpen: selectors.getSidePanelOpen,
    sidePanelStage: selectors.getSidePanelStage,
});

const mapDispatchToProps = {
    openSidePanel: pageActions.openSidePanel,
    closeSidePanel: pageActions.closeSidePanel,
    setSidePanelStage: sidepanelActions.setSidePanelStage,
};

export default withRouter(connect(mapStateToProps, mapDispatchToProps)(CIDRFormButton));
