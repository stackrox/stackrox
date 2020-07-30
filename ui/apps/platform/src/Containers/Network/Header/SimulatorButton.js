import React, { Component } from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
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
        closeWizard: PropTypes.func.isRequired,
        history: ReactRouterPropTypes.history.isRequired,
    };

    toggleSimulation = () => {
        // @TODO: This isn't very nice. We'll have  to revisit this in the future. But adding this fix to address a customer issue (https://stack-rox.atlassian.net/browse/ROX-3118)
        this.props.history.push('/main/network');
        if (this.props.creatingOrSimulating) {
            this.props.closeWizard();
        } else {
            this.props.openWizard();
            this.props.setWizardStage(wizardStages.creator);
        }
    };

    render() {
        const className = this.props.creatingOrSimulating
            ? 'bg-success-200 border-success-500 hover:border-success-600 hover:text-success-600 text-success-500'
            : 'bg-base-200 hover:border-base-400 hover:text-base-700 border-base-300';
        const iconColor = this.props.creatingOrSimulating ? '#53c6a9' : '#d2d5ed';
        return (
            <button
                type="button"
                data-testid={`simulator-button-${this.props.creatingOrSimulating ? 'on' : 'off'}`}
                className={`flex items-center flex-shrink-0 border-2 rounded-sm text-sm pl-2 pr-2 h-10 ${className}`}
                onClick={this.toggleSimulation}
            >
                <Icon.Circle className="h-2 w-2" fill={iconColor} stroke={iconColor} />
                <span className="pl-2">Network Policy Simulator</span>
            </button>
        );
    }
}

const getCreatingOrSimulating = createSelector(
    [selectors.getNetworkWizardOpen, selectors.getNetworkWizardStage],
    (wizardOpen, wizardStage) =>
        wizardOpen &&
        wizardStage !== wizardStages.details &&
        wizardStage !== wizardStages.namespaceDetails
);

const mapStateToProps = createStructuredSelector({
    creatingOrSimulating: getCreatingOrSimulating,
});

const mapDispatchToProps = {
    openWizard: pageActions.openNetworkWizard,
    closeWizard: pageActions.closeNetworkWizard,
    setWizardStage: wizardActions.setNetworkWizardStage,
};

export default withRouter(connect(mapStateToProps, mapDispatchToProps)(SimulatorButton));
