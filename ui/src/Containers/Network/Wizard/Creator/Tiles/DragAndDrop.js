import React, { Component } from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import ReactDropzone from 'react-dropzone';

import { actions as notificationActions } from 'reducers/notifications';
import { actions as backendActions } from 'reducers/network/backend';
import { actions as wizardActions } from 'reducers/network/wizard';
import wizardStages from 'Containers/Network/Wizard/wizardStages';

class DragAndDrop extends Component {
    static propTypes = {
        setNetworkPolicyModificationSuccess: PropTypes.func.isRequired,
        setNetworkPolicyModificationFailure: PropTypes.func.isRequired,
        setNetworkPolicyModificationName: PropTypes.func.isRequired,
        setWizardStage: PropTypes.func.isRequired,
        setNetworkPolicyModificationSource: PropTypes.func.isRequired,

        addToast: PropTypes.func.isRequired,
        removeToast: PropTypes.func.isRequired
    };

    onDrop = acceptedFiles => {
        acceptedFiles.forEach(file => {
            // check file type.
            if (file && !file.name.includes('.yaml')) {
                this.showToast();
                return;
            }

            this.props.setNetworkPolicyModificationName(file.name);
            const reader = new FileReader();
            reader.onload = () => {
                const fileAsBinaryString = reader.result;
                this.props.setNetworkPolicyModificationSuccess({ applyYaml: fileAsBinaryString });
            };
            reader.onerror = error => {
                this.props.setNetworkPolicyModificationFailure(error);
            };
            reader.readAsBinaryString(file);
            this.props.setNetworkPolicyModificationSource('UPLOAD');
            this.props.setWizardStage(wizardStages.simulator);
        });
    };

    showToast = () => {
        const errorMessage = 'Invalid file type. Try again.';
        this.props.addToast(errorMessage);
        setTimeout(this.props.removeToast, 500);
    };

    renderHeader = () => (
        <div className="flex text-accent-700 p-3 border-b border-base-300 mb-2 items-center flex-no-shrink">
            <Icon.Upload size="20px" strokeWidth="1.5px" />

            <div className="pl-3 font-700 text-lg">Upload a network policy YAML</div>
        </div>
    );

    renderDescription = () => (
        <div className="mb-3 px-3 font-600 text-lg leading-loose text-base-600">
            Upload your network policies to quickly preview your environment under different policy
            configurations and time windows. When ready, apply the network policies directly or
            share them with your team.
        </div>
    );

    renderDropZone = () => (
        <ReactDropzone
            onDrop={this.onDrop}
            className="flex w-full min-h-32 h-full mt-3 py-3 flex-col self-center uppercase hover:bg-warning-100 border border-dashed border-warning-500 bg-warning-100 hover:bg-warning-200 cursor-pointer justify-center"
        >
            <div className="flex flex-no-shrink flex-col">
                <div
                    className="mt-3 h-18 w-18 self-center rounded-full flex items-center justify-center flex-no-shrink"
                    style={{ background: '#faecd2', color: '#b39357' }}
                >
                    <Icon.Upload className="h-8 w-8" strokeWidth="1.5px" />
                </div>
                <span className="font-700 mt-3 text-center text-warning-800">
                    Upload and simulate network policy YAML
                </span>
            </div>
        </ReactDropzone>
    );

    render() {
        return (
            <div className="flex flex-col bg-base-100 rounded-sm shadow flex-grow flex-no-shrink mb-4">
                {this.renderHeader()}
                {this.renderDescription()}
                {this.renderDropZone()}
            </div>
        );
    }
}

const mapDispatchToProps = {
    setNetworkPolicyModificationSuccess: backendActions.fetchNetworkPolicyModification.success,
    setNetworkPolicyModificationFailure: backendActions.fetchNetworkPolicyModification.failure,
    setNetworkPolicyModificationName: wizardActions.setNetworkPolicyModificationName,
    setNetworkPolicyModificationSource: wizardActions.setNetworkPolicyModificationSource,

    setWizardStage: wizardActions.setNetworkWizardStage,

    addToast: notificationActions.addNotification,
    removeToast: notificationActions.removeOldestNotification
};

export default connect(
    null,
    mapDispatchToProps
)(DragAndDrop);
