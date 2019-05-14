import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { actions as backendActions } from 'reducers/policies/backend';
import { actions as pageActions } from 'reducers/policies/page';
import { actions as wizardActions } from 'reducers/policies/wizard';
import { actions as dialogueActions } from 'reducers/network/dialogue';
import { createStructuredSelector } from 'reselect';
import wizardStages from 'Containers/Policies/Wizard/wizardStages';
import dialogueStages from 'Containers/Network/Dialogue/dialogueStages';

import * as Icon from 'react-feather';
import PanelButton from 'Components/PanelButton';

// Buttons are the buttons above the table rows.
class Buttons extends Component {
    static propTypes = {
        selectedPolicyIds: PropTypes.arrayOf(PropTypes.string).isRequired,

        openDialogue: PropTypes.func.isRequired,
        reassessPolicies: PropTypes.func.isRequired,

        wizardOpen: PropTypes.bool.isRequired,
        openWizard: PropTypes.func.isRequired,
        setWizardStage: PropTypes.func.isRequired,
        setWizardPolicy: PropTypes.func.isRequired,
        setDialogueStage: PropTypes.func.isRequired,
        history: ReactRouterPropTypes.history.isRequired
    };

    addPolicy = () => {
        this.props.history.push({
            pathname: `/main/policies`
        });
        this.props.setWizardPolicy({ name: '' });
        this.props.setWizardStage(wizardStages.edit);
        this.props.openWizard();
    };

    showNotifierDialogue = () => {
        this.props.setDialogueStage(dialogueStages.notification);
    };

    render() {
        const buttonsDisabled = this.props.wizardOpen;
        const selectionCount = this.props.selectedPolicyIds.length;

        return (
            <React.Fragment>
                {selectionCount !== 0 && (
                    <PanelButton
                        icon={<Icon.Trash2 className="h-4 w- ml-1" />}
                        text={`Delete (${selectionCount})`}
                        className="btn btn-alert"
                        onClick={this.props.openDialogue}
                        disabled={buttonsDisabled}
                    />
                )}
                {selectionCount !== 0 && (
                    <PanelButton
                        icon={<Icon.Bell className="h-4 w- ml-1" />}
                        text="Enable Notification"
                        className="btn btn-primary ml-1"
                        onClick={this.showNotifierDialogue}
                        disabled={buttonsDisabled}
                        tooltip="Enable Notification"
                    />
                )}
                {selectionCount === 0 && (
                    <PanelButton
                        icon={<Icon.RefreshCw className="h-4 w-4 ml-1" />}
                        text="Reassess All"
                        className="btn btn-base mr-2"
                        onClick={this.props.reassessPolicies}
                        tooltip="Manually enrich external data"
                        disabled={buttonsDisabled}
                    />
                )}
                {selectionCount === 0 && (
                    <PanelButton
                        icon={<Icon.Plus className="h-4 w-4 ml-1" />}
                        text="New Policy"
                        className="btn btn-base"
                        onClick={this.addPolicy}
                        disabled={buttonsDisabled}
                    />
                )}
            </React.Fragment>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    selectedPolicyIds: selectors.getSelectedPolicyIds,
    wizardOpen: selectors.getWizardOpen
});

const mapDispatchToProps = {
    openDialogue: pageActions.openDialogue,
    openWizard: pageActions.openWizard,
    reassessPolicies: backendActions.reassessPolicies,

    setWizardStage: wizardActions.setWizardStage,
    setWizardPolicy: wizardActions.setWizardPolicy,
    setDialogueStage: dialogueActions.setNetworkDialogueStage
};

export default withRouter(
    connect(
        mapStateToProps,
        mapDispatchToProps
    )(Buttons)
);
