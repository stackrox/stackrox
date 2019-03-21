import React, { Component } from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import Tooltip from 'rc-tooltip';
import * as Icon from 'react-feather';
import ReactDropzone from 'react-dropzone';
import { actions as wizardActions } from 'reducers/network/wizard';
import { actions as notificationActions } from 'reducers/notifications';

class Upload extends Component {
    static propTypes = {
        setNetworkPolicyModification: PropTypes.func.isRequired,
        setNetworkPolicyModificationState: PropTypes.func.isRequired,
        setNetworkPolicyModificationSource: PropTypes.func.isRequired,
        setNetworkPolicyModificationName: PropTypes.func.isRequired,

        addToast: PropTypes.func.isRequired,
        removeToast: PropTypes.func.isRequired
    };

    onDrop = acceptedFiles => {
        this.props.setNetworkPolicyModificationState('REQUEST');
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
                this.props.setNetworkPolicyModification({ applyYaml: fileAsBinaryString });
                this.props.setNetworkPolicyModificationState('SUCCESS');
            };
            reader.onerror = () => {
                this.props.setNetworkPolicyModificationState('ERROR');
            };
            reader.readAsBinaryString(file);
            this.props.setNetworkPolicyModificationSource('UPLOAD');
        });
    };

    showToast = () => {
        const errorMessage = 'Invalid file type. Try again.';
        this.props.addToast(errorMessage);
        setTimeout(this.props.removeToast, 500);
    };

    render() {
        return (
            <ReactDropzone
                onDrop={this.onDrop}
                className="inline-block px-2 py-2 border-r border-base-300 cursor-pointer"
            >
                <Tooltip
                    placement="top"
                    overlay={<div>Upload a new YAML</div>}
                    mouseEnterDelay={0.5}
                    mouseLeaveDelay={0}
                >
                    <Icon.Upload className="h-4 w-4 text-base-500 hover:bg-base-200" />
                </Tooltip>
            </ReactDropzone>
        );
    }
}

const mapDispatchToProps = {
    setNetworkPolicyModification: wizardActions.setNetworkPolicyModification,
    setNetworkPolicyModificationState: wizardActions.setNetworkPolicyModificationState,
    setNetworkPolicyModificationSource: wizardActions.setNetworkPolicyModificationSource,
    setNetworkPolicyModificationName: wizardActions.setNetworkPolicyModificationName,

    addToast: notificationActions.addNotification,
    removeToast: notificationActions.removeOldestNotification
};

export default connect(
    null,
    mapDispatchToProps
)(Upload);
