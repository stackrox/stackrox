import React, { Component } from 'react';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import { selectors } from 'reducers';
import { actions as pageActions } from 'reducers/network/page';
import { actions as wizardActions } from 'reducers/network/wizard';

import wizardStages from '../Wizard/wizardStages';

class SimulatorButton extends Component {
    static propTypes = {
        creatingOrSimulating: PropTypes.bool.isRequired,
        openWizard: PropTypes.func.isRequired,
        setWizardStage: PropTypes.func.isRequired,
        closeWizard: PropTypes.func.isRequired
    };

    toggleSimulation = () => {
        if (this.props.creatingOrSimulating) {
            this.props.closeWizard();
        } else {
            this.props.openWizard();
            this.props.setWizardStage(wizardStages.simulator);
        }
    };

    render() {
        const className = this.props.creatingOrSimulating
            ? 'bg-success-200 border-success-500 hover:border-success-600 hover:text-success-600 text-success-500'
            : 'bg-base-200 hover:border-base-300 hover:text-base-600 border-base-200 text-base-500';
        const iconColor = this.props.creatingOrSimulating ? '#53c6a9' : '#d2d5ed';
        return (
            <button
                type="button"
                data-test-id={`simulator-button-${this.props.creatingOrSimulating ? 'on' : 'off'}`}
                className={`flex-no-shrink border-2 rounded-sm text-sm ml-2 pl-2 pr-2 h-9 ${className}`}
                onClick={this.toggleSimulation}
            >
                <span className="pr-1">Simulate Network Policy</span>
                <Icon.Circle className="h-2 w-2" fill={iconColor} stroke={iconColor} />
            </button>
        );
    }
}

const getCreatingOrSimulating = createSelector(
    [selectors.getNetworkWizardOpen, selectors.getNetworkWizardStage],
    (wizardOpen, wizardStage) => wizardOpen && wizardStage !== wizardStages.details
);

const mapStateToProps = createStructuredSelector({
    creatingOrSimulating: getCreatingOrSimulating
});

const mapDispatchToProps = {
    openWizard: pageActions.openNetworkWizard,
    closeWizard: pageActions.closeNetworkWizard,
    setWizardStage: wizardActions.setNetworkWizardStage
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(SimulatorButton);
