import React, { Component } from 'react';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions as notificationActions } from 'reducers/notifications';
import { actions as pageActions } from 'reducers/network/page';
import { actions as wizardActions } from 'reducers/network/wizard';
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
        setYamlFile: PropTypes.func.isRequired,
        yamlUploadState: PropTypes.string.isRequired,

        errorMessage: PropTypes.string.isRequired,
        yamlFile: PropTypes.shape({
            content: PropTypes.string,
            name: PropTypes.string
        }),
        addToast: PropTypes.func.isRequired,
        removeToast: PropTypes.func.isRequired
    };

    static defaultProps = {
        yamlFile: null
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

    onDrop = acceptedFiles => {
        acceptedFiles.forEach(file => {
            if (file && !file.name.includes('.yaml')) {
                this.showToast();
                return;
            }
            const reader = new FileReader();
            reader.onload = () => {
                const fileAsBinaryString = reader.result;
                this.props.setYamlFile({ content: fileAsBinaryString, name: file.name });
            };
            reader.readAsBinaryString(file);
        });
    };

    showToast = () => {
        const errorMessage = 'Invalid file type. Try again.';
        this.props.addToast(errorMessage);
        setTimeout(this.props.removeToast, 500);
    };

    renderProcessingView = () => {
        const { yamlUploadState } = this.props;
        if (yamlUploadState !== 'REQUEST') return null;

        return <div className="flex flex-col flex-1">{LoadingSection()}</div>;
    };

    renderUploadView = () => {
        const { yamlUploadState } = this.props;
        if (yamlUploadState !== 'INITIAL') return null;

        const uploadMessage = 'Click to upload or drop network policy yaml inside';
        return (
            <div className="flex flex-col overflow-auto w-full h-full pb-4">
                {this.state.showGetStartedSection && GettingStarted(this.hideGetStartedSection)}
                {DragAndDrop({ message: uploadMessage, onDrop: this.onDrop })}
            </div>
        );
    };

    renderSuccessView = () => {
        const { yamlUploadState } = this.props;
        if (yamlUploadState !== 'SUCCESS') return null;

        const uploadMessage = 'Simulate another set of policies';
        return (
            <div className="flex flex-col w-full h-full space-between">
                {this.state.showDragAndDrop && (
                    <div>{DragAndDrop({ message: uploadMessage, onDrop: this.onDrop })}</div>
                )}
                <SuccessView onCollapse={this.toggleDragAndDrop} />
                <SendNotificationSection />
            </div>
        );
    };

    renderErrorView = () => {
        const { yamlUploadState } = this.props;
        if (yamlUploadState !== 'ERROR') return null;

        const uploadMessage = 'Simulate another set of policies';
        return (
            <div className="flex flex-col flex-1">
                {this.state.showDragAndDrop && (
                    <div>{DragAndDrop({ message: uploadMessage, onDrop: this.onDrop })}</div>
                )}
                <ErrorView
                    yamlFile={this.props.yamlFile}
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

        const { yamlFile } = this.props;
        const colorType = this.props.yamlUploadState === 'ERROR' ? 'alert' : 'success';
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
                    {!yamlFile && this.renderUploadView()}
                    {yamlFile && this.renderProcessingView()}
                    {yamlFile && this.renderErrorView()}
                    {yamlFile && this.renderSuccessView()}
                </Panel>
            </div>
        );
    }
}

const getYamlUploadState = createSelector(
    [selectors.getNetworkYamlFile, selectors.getNetworkGraphState],
    (yamlFile, networkGraphState) => {
        if (!yamlFile) {
            return 'INITIAL';
        }
        return networkGraphState;
    }
);

const mapStateToProps = createStructuredSelector({
    wizardOpen: selectors.getNetworkWizardOpen,
    wizardStage: selectors.getNetworkWizardStage,
    errorMessage: selectors.getNetworkErrorMessage,
    yamlFile: selectors.getNetworkYamlFile,
    yamlUploadState: getYamlUploadState
});

const mapDispatchToProps = {
    addToast: notificationActions.addNotification,
    removeToast: notificationActions.removeOldestNotification,
    setYamlFile: wizardActions.setNetworkYamlFile,
    onClose: pageActions.closeNetworkWizard
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(Simulator);
