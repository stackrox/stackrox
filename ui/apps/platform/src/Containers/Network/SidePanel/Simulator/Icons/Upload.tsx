import React, { useCallback, ReactElement } from 'react';
import { connect } from 'react-redux';
import * as Icon from 'react-feather';
import { useDropzone } from 'react-dropzone';
import { Tooltip } from '@patternfly/react-core';

import { actions as sidepanelActions } from 'reducers/network/sidepanel';
import { actions as notificationActions } from 'reducers/notifications';

type UploadProps = {
    setNetworkPolicyModification: (policy) => void;
    setNetworkPolicyModificationState: (state) => void;
    setNetworkPolicyModificationSource: (source) => void;
    setNetworkPolicyModificationName: (name) => void;

    addToast: (message) => void;
    removeToast: () => void;
};

function Upload({
    setNetworkPolicyModification,
    setNetworkPolicyModificationName,
    setNetworkPolicyModificationSource,
    setNetworkPolicyModificationState,
    addToast,
    removeToast,
}: UploadProps): ReactElement {
    // TODO: factor out upload logic into custom hook
    //  nearly duplicate logic here and Network/SidePanel/Creator/Tiles/UploadNetworkPolicySection
    const showToast = useCallback(() => {
        const errorMessage = 'Invalid file type. Try again.';
        addToast(errorMessage);
        setTimeout(removeToast, 500);
    }, [addToast, removeToast]);

    const onDrop = useCallback(
        (acceptedFiles) => {
            setNetworkPolicyModificationState('REQUEST');
            acceptedFiles.forEach((file) => {
                // check file type.
                if (file && !file.name.includes('.yaml')) {
                    showToast();
                    return;
                }

                setNetworkPolicyModificationName(file.name);
                const reader = new FileReader();
                reader.onload = () => {
                    const fileAsBinaryString = reader.result;
                    setNetworkPolicyModification({ applyYaml: fileAsBinaryString });
                    setNetworkPolicyModificationState('SUCCESS');
                };
                reader.onerror = () => {
                    setNetworkPolicyModificationState('ERROR');
                };
                reader.readAsBinaryString(file);
                setNetworkPolicyModificationSource('UPLOAD');
            });
        },
        [
            setNetworkPolicyModification,
            setNetworkPolicyModificationName,
            setNetworkPolicyModificationSource,
            setNetworkPolicyModificationState,
            showToast,
        ]
    );

    const { getRootProps, getInputProps } = useDropzone({ onDrop });

    return (
        <Tooltip content="Upload a new YAML">
            <div
                {...getRootProps()}
                className="inline-block px-2 py-2 border-r border-base-300 cursor-pointer outline-none"
            >
                <input {...getInputProps()} />
                <Icon.Upload className="h-4 w-4 text-base-500 hover:bg-base-200" />
            </div>
        </Tooltip>
    );
}

const mapDispatchToProps = {
    setNetworkPolicyModification: sidepanelActions.setNetworkPolicyModification,
    setNetworkPolicyModificationState: sidepanelActions.setNetworkPolicyModificationState,
    setNetworkPolicyModificationSource: sidepanelActions.setNetworkPolicyModificationSource,
    setNetworkPolicyModificationName: sidepanelActions.setNetworkPolicyModificationName,

    addToast: notificationActions.addNotification,
    removeToast: notificationActions.removeOldestNotification,
};

export default connect(null, mapDispatchToProps)(Upload);
