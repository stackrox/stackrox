import React, { useCallback } from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import { useDropzone } from 'react-dropzone';
import Tooltip from 'Components/Tooltip';
import TooltipOverlay from 'Components/TooltipOverlay';
import { actions as wizardActions } from 'reducers/network/wizard';
import { actions as notificationActions } from 'reducers/notifications';

const Upload = (props) => {
    const showToast = useCallback(() => {
        const errorMessage = 'Invalid file type. Try again.';
        props.addToast(errorMessage);
        setTimeout(props.removeToast, 500);
    }, [props]);

    const onDrop = useCallback(
        (acceptedFiles) => {
            props.setNetworkPolicyModificationState('REQUEST');
            acceptedFiles.forEach((file) => {
                // check file type.
                if (file && !file.name.includes('.yaml')) {
                    showToast();
                    return;
                }

                props.setNetworkPolicyModificationName(file.name);
                const reader = new FileReader();
                reader.onload = () => {
                    const fileAsBinaryString = reader.result;
                    props.setNetworkPolicyModification({ applyYaml: fileAsBinaryString });
                    props.setNetworkPolicyModificationState('SUCCESS');
                };
                reader.onerror = () => {
                    props.setNetworkPolicyModificationState('ERROR');
                };
                reader.readAsBinaryString(file);
                props.setNetworkPolicyModificationSource('UPLOAD');
            });
        },
        [props, showToast]
    );

    const { getRootProps, getInputProps } = useDropzone({ onDrop });

    return (
        <Tooltip content={<TooltipOverlay>Upload a new YAML</TooltipOverlay>}>
            <div
                {...getRootProps()}
                className="inline-block px-2 py-2 border-r border-base-300 cursor-pointer outline-none"
            >
                <input {...getInputProps()} />
                <Icon.Upload className="h-4 w-4 text-base-500 hover:bg-base-200" />
            </div>
        </Tooltip>
    );
};

Upload.propTypes = {
    setNetworkPolicyModification: PropTypes.func.isRequired,
    setNetworkPolicyModificationState: PropTypes.func.isRequired,
    setNetworkPolicyModificationSource: PropTypes.func.isRequired,
    setNetworkPolicyModificationName: PropTypes.func.isRequired,

    addToast: PropTypes.func.isRequired,
    removeToast: PropTypes.func.isRequired,
};

const mapDispatchToProps = {
    setNetworkPolicyModification: wizardActions.setNetworkPolicyModification,
    setNetworkPolicyModificationState: wizardActions.setNetworkPolicyModificationState,
    setNetworkPolicyModificationSource: wizardActions.setNetworkPolicyModificationSource,
    setNetworkPolicyModificationName: wizardActions.setNetworkPolicyModificationName,

    addToast: notificationActions.addNotification,
    removeToast: notificationActions.removeOldestNotification,
};

export default connect(null, mapDispatchToProps)(Upload);
