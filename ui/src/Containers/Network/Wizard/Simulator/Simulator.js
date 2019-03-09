import React, { Component } from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions as backendActions } from 'reducers/network/backend';
import { actions as pageActions } from 'reducers/network/page';
import PropTypes from 'prop-types';
import Panel from 'Components/Panel';

import wizardStages from '../wizardStages';
import ProcessingView from './ProcessingView';
import SuccessView from './SuccessView';
import ErrorView from './ErrorView';

class Simulator extends Component {
    static propTypes = {
        wizardOpen: PropTypes.bool.isRequired,
        wizardStage: PropTypes.string.isRequired,
        closeWizard: PropTypes.func.isRequired,
        setModification: PropTypes.func.isRequired,
        modificationState: PropTypes.string.isRequired
    };

    onClose = () => {
        this.props.closeWizard();
        this.props.setModification(null);
    };

    render() {
        if (!this.props.wizardOpen || this.props.wizardStage !== wizardStages.simulator) {
            return null;
        }

        const colorType = this.props.modificationState === 'ERROR' ? 'alert' : 'success';
        const header = 'Network Policy Simulator';

        return (
            <div
                data-test-id="network-simulator-panel"
                className="w-full h-full absolute pin-r pin-b pt-1 pb-1 pr-1 shadow-md bg-base-200"
            >
                <Panel
                    className="border-t-0 border-r-0 border-b-0"
                    header={header}
                    onClose={this.onClose}
                    closeButtonClassName={`bg-${colorType}-600 hover:bg-${colorType}-700`}
                    closeButtonIconColor="text-base-100"
                >
                    <ProcessingView />
                    <ErrorView />
                    <SuccessView />
                </Panel>
            </div>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    wizardOpen: selectors.getNetworkWizardOpen,
    wizardStage: selectors.getNetworkWizardStage,
    errorMessage: selectors.getNetworkErrorMessage,

    modificationState: selectors.getNetworkPolicyModificationState
});

const mapDispatchToProps = {
    closeWizard: pageActions.closeNetworkWizard,
    setModification: backendActions.fetchNetworkPolicyModification.success
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(Simulator);
