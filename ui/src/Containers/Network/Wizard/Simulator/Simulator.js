import React from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions as wizardActions } from 'reducers/network/wizard';
import { actions as pageActions } from 'reducers/network/page';
import PropTypes from 'prop-types';
import Panel from 'Components/Panel';

import wizardStages from '../wizardStages';
import ProcessingView from './ProcessingView';
import SuccessView from './SuccessView';
import ErrorView from './ErrorView';

const Simulator = ({
    closeWizard,
    setModification,
    wizardOpen,
    wizardStage,
    modificationState
}) => {
    function onClose() {
        closeWizard();
        setModification(null);
    }

    if (!wizardOpen || wizardStage !== wizardStages.simulator) {
        return null;
    }

    const colorType = modificationState === 'ERROR' ? 'alert' : 'success';

    return (
        <div
            data-test-id="network-simulator-panel"
            className="w-full h-full absolute pin-r pin-b pt-1 pb-1 pr-1 shadow-md bg-base-200"
        >
            <Panel
                className="border-t-0 border-r-0 border-b-0"
                header="Network Policy Simulator"
                onClose={onClose}
                closeButtonClassName={`bg-${colorType}-600 hover:bg-${colorType}-700`}
                closeButtonIconColor="text-base-100"
            >
                <ProcessingView />
                <ErrorView />
                <SuccessView />
            </Panel>
        </div>
    );
};

Simulator.propTypes = {
    wizardOpen: PropTypes.bool.isRequired,
    wizardStage: PropTypes.string.isRequired,
    closeWizard: PropTypes.func.isRequired,
    setModification: PropTypes.func.isRequired,
    modificationState: PropTypes.string.isRequired
};

const mapStateToProps = createStructuredSelector({
    wizardOpen: selectors.getNetworkWizardOpen,
    wizardStage: selectors.getNetworkWizardStage,
    errorMessage: selectors.getNetworkErrorMessage,

    modificationState: selectors.getNetworkPolicyModificationState
});

const mapDispatchToProps = {
    closeWizard: pageActions.closeNetworkWizard,
    setModification: wizardActions.setNetworkPolicyModification
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(Simulator);
