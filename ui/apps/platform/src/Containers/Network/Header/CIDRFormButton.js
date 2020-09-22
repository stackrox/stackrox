import React from 'react';
import { Box } from 'react-feather';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { actions as pageActions } from 'reducers/network/page';
import { actions as wizardActions } from 'reducers/network/wizard';
import wizardStages from '../Wizard/wizardStages';

const CIDRFormButton = ({
    isWizardOpen,
    currentWizardStage,
    openWizard,
    closeWizard,
    setWizardStage,
}) => {
    function toggleForm() {
        if (isWizardOpen && currentWizardStage === wizardStages.cidrForm) {
            closeWizard();
        } else {
            if (!isWizardOpen) {
                openWizard();
            }
            setWizardStage(wizardStages.cidrForm);
        }
    }

    return (
        <button
            type="button"
            onClick={toggleForm}
            className="flex border-l border-dashed border-base-400 items-center px-4 font-condensed uppercase justify-center hover:bg-base-300"
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
    isWizardOpen: selectors.getNetworkWizardOpen,
    currentWizardStage: selectors.getNetworkWizardStage,
});

const mapDispatchToProps = {
    openWizard: pageActions.openNetworkWizard,
    closeWizard: pageActions.closeNetworkWizard,
    setWizardStage: wizardActions.setNetworkWizardStage,
};

export default withRouter(connect(mapStateToProps, mapDispatchToProps)(CIDRFormButton));
