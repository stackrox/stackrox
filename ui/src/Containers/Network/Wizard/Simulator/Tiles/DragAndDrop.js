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
        uploadMessage: PropTypes.string.isRequired,

        setNetworkPolicyModificationSuccess: PropTypes.func.isRequired,
        setNetworkPolicyModificationFailure: PropTypes.func.isRequired,
        setNetworkPolicyModificationName: PropTypes.func.isRequired,
        setWizardStage: PropTypes.func.isRequired,

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
            this.props.setWizardStage(wizardStages.simulator);
        });
    };

    showToast = () => {
        const errorMessage = 'Invalid file type. Try again.';
        this.props.addToast(errorMessage);
        setTimeout(this.props.removeToast, 500);
    };

    render() {
        return (
            <section
                data-test-id="upload-yaml-panel"
                className="bg-base-100 min-h-32 m-3 mt-4 mb-0 flex h-full border border-dashed border-base-300 hover:border-base-500 cursor-pointer"
            >
                <ReactDropzone
                    onDrop={this.onDrop}
                    className="flex w-full h-full flex-col self-center uppercase p-5 hover:bg-warning-100 shadow justify-center"
                >
                    <div
                        className="h-18 w-18 self-center rounded-full flex items-center justify-center flex-no-shrink"
                        style={{ background: '#faecd2', color: '#b39357' }}
                    >
                        <Icon.Upload className="h-8 w-8" strokeWidth="1.5px" />
                    </div>

                    <div className="text-center pt-5 font-700">{this.props.uploadMessage}</div>
                </ReactDropzone>
            </section>
        );
    }
}

const mapDispatchToProps = {
    setNetworkPolicyModificationSuccess: backendActions.fetchNetworkPolicyModification.success,
    setNetworkPolicyModificationFailure: backendActions.fetchNetworkPolicyModification.failure,
    setNetworkPolicyModificationName: wizardActions.setNetworkPolicyModificationName,

    setWizardStage: wizardActions.setNetworkWizardStage,

    addToast: notificationActions.addNotification,
    removeToast: notificationActions.removeOldestNotification
};

export default connect(
    null,
    mapDispatchToProps
)(DragAndDrop);
