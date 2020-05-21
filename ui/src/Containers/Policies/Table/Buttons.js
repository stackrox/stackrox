import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import { connect } from 'react-redux';
import { Bell, BellOff, Plus, RefreshCw, Trash2, Upload } from 'react-feather';

import { selectors } from 'reducers';
import { actions as backendActions } from 'reducers/policies/backend';
import { actions as pageActions } from 'reducers/policies/page';
import { actions as wizardActions } from 'reducers/policies/wizard';
import { createStructuredSelector } from 'reselect';
import wizardStages from 'Containers/Policies/Wizard/wizardStages';
import Menu from 'Components/Menu';
import PanelButton from 'Components/PanelButton';
import { knownBackendFlags } from 'utils/featureFlags';
import FeatureEnabled from 'Containers/FeatureEnabled';
import policyBulkActions from '../policyBulkActions';

// Buttons are the buttons above the table rows.
class Buttons extends Component {
    static propTypes = {
        selectedPolicyIds: PropTypes.arrayOf(PropTypes.string).isRequired,
        startPolicyImport: PropTypes.func.isRequired,
        setPoliciesAction: PropTypes.func.isRequired,
        reassessPolicies: PropTypes.func.isRequired,
        wizardOpen: PropTypes.bool.isRequired,
        wizardPolicy: PropTypes.shape({
            id: PropTypes.string,
            name: PropTypes.string,
        }),
        openWizard: PropTypes.func.isRequired,
        setWizardStage: PropTypes.func.isRequired,
        setWizardPolicy: PropTypes.func.isRequired,
        history: ReactRouterPropTypes.history.isRequired,
    };

    static defaultProps = {
        wizardPolicy: null,
    };

    addPolicy = () => {
        this.props.history.push({
            pathname: `/main/policies`,
        });
        this.props.setWizardPolicy({ name: '' });
        this.props.setWizardStage(wizardStages.edit);
        this.props.openWizard();
    };

    startPolicyImport = () => {
        this.props.startPolicyImport();
    };

    openDialogue = (policiesAction) => {
        this.props.setPoliciesAction(policiesAction);
    };

    render() {
        const buttonsDisabled = this.props.wizardOpen;
        const selectionCount = this.props.selectedPolicyIds.length;
        const bulkOperationOptions = [
            {
                label: 'Enable Notification',
                onClick: () => this.openDialogue(policyBulkActions.enableNotification),
                icon: <Bell className="h-4" />,
            },
            {
                label: 'Disable Notification',
                onClick: () => this.openDialogue(policyBulkActions.disableNotification),
                icon: <BellOff className="h-4" />,
            },
            {
                label: 'Delete Policies',
                onClick: () => this.openDialogue(policyBulkActions.deletePolicies),
                className: 'border-t bg-alert-100 text-alert-700',
                icon: <Trash2 className="h-4" />,
            },
        ];

        return (
            <>
                {selectionCount !== 0 && (
                    <Menu
                        className="mr-2"
                        buttonClass="btn btn-base"
                        buttonText="Actions"
                        options={bulkOperationOptions}
                        disabled={
                            buttonsDisabled &&
                            this.props.selectedPolicyIds.find(
                                (id) => id === this.props.wizardPolicy.id
                            ) !== undefined
                        }
                    />
                )}
                {selectionCount === 0 && (
                    <PanelButton
                        icon={<RefreshCw className="h-4 w-4 ml-1" />}
                        className="btn btn-base mr-2"
                        onClick={this.props.reassessPolicies}
                        tooltip="Manually enrich external data"
                        disabled={buttonsDisabled}
                    >
                        Reassess All
                    </PanelButton>
                )}
                {selectionCount === 0 && (
                    <>
                        <FeatureEnabled featureFlag={knownBackendFlags.ROX_POLICY_IMPORT_EXPORT}>
                            <PanelButton
                                icon={<Upload className="h-4 w-4 ml-1" />}
                                className="btn btn-base mr-2"
                                onClick={this.startPolicyImport}
                                disabled={buttonsDisabled}
                                tooltip="Import a policy"
                                dataTestId="import-policy-btn"
                            >
                                Import Policy
                            </PanelButton>
                        </FeatureEnabled>
                        <PanelButton
                            icon={<Plus className="h-4 w-4 ml-1" />}
                            className="btn btn-base"
                            onClick={this.addPolicy}
                            disabled={buttonsDisabled}
                            tooltip="Create a new policy"
                        >
                            New Policy
                        </PanelButton>
                    </>
                )}
            </>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    selectedPolicyIds: selectors.getSelectedPolicyIds,
    wizardOpen: selectors.getWizardOpen,
    wizardPolicy: selectors.getWizardPolicy,
});

const mapDispatchToProps = {
    setPoliciesAction: pageActions.setPoliciesAction,
    openWizard: pageActions.openWizard,
    reassessPolicies: backendActions.reassessPolicies,

    setWizardStage: wizardActions.setWizardStage,
    setWizardPolicy: wizardActions.setWizardPolicy,
};

export default withRouter(connect(mapStateToProps, mapDispatchToProps)(Buttons));
