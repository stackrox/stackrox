import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { actions as backendActions } from 'reducers/policies/backend';
import { actions as pageActions } from 'reducers/policies/page';
import { actions as wizardActions } from 'reducers/policies/wizard';
import { createStructuredSelector } from 'reselect';
import wizardStages from 'Containers/Policies/Wizard/wizardStages';
import Menu from 'Components/Menu';

import * as Icon from 'react-feather';
import PanelButton from 'Components/PanelButton';
import policyBulkActions from '../policyBulkActions';

// Buttons are the buttons above the table rows.
class Buttons extends Component {
    static propTypes = {
        selectedPolicyIds: PropTypes.arrayOf(PropTypes.string).isRequired,

        setPoliciesAction: PropTypes.func.isRequired,
        reassessPolicies: PropTypes.func.isRequired,

        wizardOpen: PropTypes.bool.isRequired,
        wizardPolicy: PropTypes.shape({
            id: PropTypes.string,
            name: PropTypes.string
        }),
        openWizard: PropTypes.func.isRequired,
        setWizardStage: PropTypes.func.isRequired,
        setWizardPolicy: PropTypes.func.isRequired,
        history: ReactRouterPropTypes.history.isRequired
    };

    static defaultProps = {
        wizardPolicy: null
    };

    addPolicy = () => {
        this.props.history.push({
            pathname: `/main/policies`
        });
        this.props.setWizardPolicy({ name: '' });
        this.props.setWizardStage(wizardStages.edit);
        this.props.openWizard();
    };

    openDialogue = policiesAction => {
        this.props.setPoliciesAction(policiesAction);
    };

    render() {
        const buttonsDisabled = this.props.wizardOpen;
        const selectionCount = this.props.selectedPolicyIds.length;
        const bulkOperationOptions = [
            {
                label: 'Enable Notification',
                onClick: () => this.openDialogue(policyBulkActions.enableNotification),
                icon: <Icon.Bell className="h-4" />
            },
            {
                label: 'Disable Notification',
                onClick: () => this.openDialogue(policyBulkActions.disableNotification),
                icon: <Icon.BellOff className="h-4" />
            },
            {
                label: 'Delete Policies',
                onClick: () => this.openDialogue(policyBulkActions.deletePolicies),
                className: 'border-t bg-alert-100 text-alert-700',
                icon: <Icon.Trash2 className="h-4" />
            }
        ];

        return (
            <React.Fragment>
                {selectionCount !== 0 && (
                    <Menu
                        className="mr-2"
                        buttonClass="btn btn-base"
                        buttonContent={
                            <div className="flex items-center">
                                Actions
                                <Icon.ChevronDown className="ml-2 h-4 w-4 pointer-events-none" />
                            </div>
                        }
                        options={bulkOperationOptions}
                        disabled={
                            buttonsDisabled &&
                            this.props.selectedPolicyIds.find(
                                id => id === this.props.wizardPolicy.id
                            ) !== undefined
                        }
                    />
                )}
                {selectionCount === 0 && (
                    <PanelButton
                        icon={<Icon.RefreshCw className="h-4 w-4 ml-1" />}
                        className="btn btn-base mr-2"
                        onClick={this.props.reassessPolicies}
                        tooltip="Manually enrich external data"
                        disabled={buttonsDisabled}
                    >
                        Reassess All
                    </PanelButton>
                )}
                {selectionCount === 0 && (
                    <PanelButton
                        icon={<Icon.Plus className="h-4 w-4 ml-1" />}
                        className="btn btn-base"
                        onClick={this.addPolicy}
                        disabled={buttonsDisabled}
                    >
                        New Policy
                    </PanelButton>
                )}
            </React.Fragment>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    selectedPolicyIds: selectors.getSelectedPolicyIds,
    wizardOpen: selectors.getWizardOpen,
    wizardPolicy: selectors.getWizardPolicy
});

const mapDispatchToProps = {
    setPoliciesAction: pageActions.setPoliciesAction,
    openWizard: pageActions.openWizard,
    reassessPolicies: backendActions.reassessPolicies,

    setWizardStage: wizardActions.setWizardStage,
    setWizardPolicy: wizardActions.setWizardPolicy
};

export default withRouter(
    connect(
        mapStateToProps,
        mapDispatchToProps
    )(Buttons)
);
