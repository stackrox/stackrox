import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { actions as backendActions } from 'reducers/network/backend';
import { actions as wizardActions } from 'reducers/network/wizard';

import wizardStages from '../../wizardStages';

class GenerateButton extends Component {
    static propTypes = {
        setWizardStage: PropTypes.func.isRequired,
        requestNetworkPolicyModification: PropTypes.func.isRequired
    };

    onClick = () => {
        this.props.requestNetworkPolicyModification();
        this.props.setWizardStage(wizardStages.simulator);
    };

    render() {
        return (
            <div className="flex m-3 py-2 items-center justify-center">
                <button
                    type="button"
                    className="rounded-sm px-4 py-3 border-2 border-primary-300 hover:border-primary-400 text-center text-3xlg font-700 text-primary-700 bg-primary-100 hover:bg-primary-200"
                    onClick={this.onClick}
                >
                    Generate and simulate network policies
                </button>
            </div>
        );
    }
}

const mapDispatchToProps = {
    setWizardStage: wizardActions.setNetworkWizardStage,
    requestNetworkPolicyModification: backendActions.fetchNetworkPolicyModification.request
};

export default connect(
    null,
    mapDispatchToProps
)(GenerateButton);
