import React from 'react';
import { Box } from 'react-feather';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { actions as pageActions } from 'reducers/network/page';
import { actions as sidepanelActions } from 'reducers/network/sidepanel';
import wizardStages from '../SidePanel/wizardStages';

const CIDRFormButton = ({
    sidePanelOpen,
    currentWizardStage,
    openSidePanel,
    closeSidePanel,
    setWizardStage,
    isDisabled,
}) => {
    function toggleForm() {
        if (sidePanelOpen && currentWizardStage === wizardStages.cidrForm) {
            closeSidePanel();
        } else {
            if (!sidePanelOpen) {
                openSidePanel();
            }
            setWizardStage(wizardStages.cidrForm);
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
    sidePanelOpen: selectors.getNetworkSidePanelOpen,
    currentWizardStage: selectors.getNetworkWizardStage,
});

const mapDispatchToProps = {
    openSidePanel: pageActions.openSidePanel,
    closeSidePanel: pageActions.closeSidePanel,
    setWizardStage: sidepanelActions.setNetworkWizardStage,
};

export default withRouter(connect(mapStateToProps, mapDispatchToProps)(CIDRFormButton));
