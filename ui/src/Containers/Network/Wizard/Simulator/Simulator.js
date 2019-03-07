import React, { Component } from 'react';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions as pageActions } from 'reducers/network/page';
import PropTypes from 'prop-types';
import Panel from 'Components/Panel';

import DragAndDrop from './Tiles/DragAndDrop';
import GettingStarted from './Tiles/GettingStarted';
import LoadingSection from './Tiles/LoadingSection';

import wizardStages from '../wizardStages';
import SendNotificationSection from './SendNotificationSection';
import SuccessView from './SuccessView';
import ErrorView from './ErrorView';

class Simulator extends Component {
    static propTypes = {
        wizardOpen: PropTypes.bool.isRequired,
        wizardStage: PropTypes.string.isRequired,
        onClose: PropTypes.func.isRequired,
        modificationState: PropTypes.string.isRequired,

        errorMessage: PropTypes.string.isRequired
    };

    state = {
        showGetStartedSection: true,
        showDragAndDrop: true
    };

    hideGetStartedSection = () => this.setState({ showGetStartedSection: false });

    toggleDragAndDrop = showDragAndDrop => {
        this.setState({ showDragAndDrop });
    };

    onClose = () => {
        this.props.onClose();
    };

    renderProcessingView = () => {
        const { modificationState } = this.props;
        if (modificationState !== 'REQUEST') return null;

        return <div className="flex flex-col flex-1">{LoadingSection()}</div>;
    };

    renderUploadView = () => {
        const { modificationState } = this.props;
        if (modificationState !== 'INITIAL') return null;

        const uploadMessage = 'Click to upload or drop network policy yaml inside';
        return (
            <div className="flex flex-col overflow-auto w-full h-full pb-4">
                {this.state.showGetStartedSection && GettingStarted(this.hideGetStartedSection)}
                <DragAndDrop uploadMessage={uploadMessage} />
            </div>
        );
    };

    renderSuccessView = () => {
        const { modificationState } = this.props;
        if (modificationState !== 'SUCCESS') return null;

        const uploadMessage = 'Simulate another set of policies';
        return (
            <div className="flex flex-col w-full h-full space-between">
                {this.state.showDragAndDrop && <DragAndDrop uploadMessage={uploadMessage} />}
                <SuccessView onCollapse={this.toggleDragAndDrop} />
                <SendNotificationSection />
            </div>
        );
    };

    renderErrorView = () => {
        const { modificationState } = this.props;
        if (modificationState !== 'ERROR') return null;

        const uploadMessage = 'Simulate another set of policies';
        return (
            <div className="flex flex-col flex-1">
                {this.state.showDragAndDrop && <DragAndDrop uploadMessage={uploadMessage} />}
                <ErrorView
                    errorMessage={this.props.errorMessage}
                    onCollapse={this.toggleDragAndDrop}
                />
            </div>
        );
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
                    onClose={this.props.onClose}
                    closeButtonClassName={`bg-${colorType}-600 hover:bg-${colorType}-700`}
                    closeButtonIconColor="text-base-100"
                >
                    {this.renderUploadView()}
                    {this.renderProcessingView()}
                    {this.renderErrorView()}
                    {this.renderSuccessView()}
                </Panel>
            </div>
        );
    }
}

const getModificationState = createSelector(
    [selectors.getNetworkPolicyModification, selectors.getNetworkPolicyModificationState],
    (modification, modificationState) => {
        if (!modification) {
            return 'INITIAL';
        }
        return modificationState;
    }
);

const mapStateToProps = createStructuredSelector({
    wizardOpen: selectors.getNetworkWizardOpen,
    wizardStage: selectors.getNetworkWizardStage,
    errorMessage: selectors.getNetworkErrorMessage,
    modificationState: getModificationState
});

const mapDispatchToProps = {
    onClose: pageActions.closeNetworkWizard
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(Simulator);
