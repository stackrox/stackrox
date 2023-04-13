import React, { useCallback, ReactElement } from 'react';
import { connect } from 'react-redux';
import * as Icon from 'react-feather';
import { useDropzone } from 'react-dropzone';

import { fileUploadColors } from 'constants/visuals/colors';
import { actions as notificationActions } from 'reducers/notifications';
import { actions as sidepanelActions } from 'reducers/network/sidepanel';
import sidepanelStages from 'Containers/Network/SidePanel/sidepanelStages';

type UploadNetworkPolicySectionProps = {
    setNetworkPolicyModification: (policy) => void;
    setNetworkPolicyModificationState: (state) => void;
    setNetworkPolicyModificationSource: (source) => void;
    setNetworkPolicyModificationName: (name) => void;
    setSidePanelStage: (stage) => void;

    addToast: (message) => void;
    removeToast: () => void;
};

function UploadNetworkPolicySection({
    setNetworkPolicyModification,
    setNetworkPolicyModificationName,
    setNetworkPolicyModificationSource,
    setNetworkPolicyModificationState,
    setSidePanelStage,
    addToast,
    removeToast,
}: UploadNetworkPolicySectionProps): ReactElement {
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
                setSidePanelStage(sidepanelStages.simulator);
            });
        },
        [
            setNetworkPolicyModification,
            setNetworkPolicyModificationName,
            setNetworkPolicyModificationSource,
            setNetworkPolicyModificationState,
            setSidePanelStage,
            showToast,
        ]
    );

    const { getRootProps, getInputProps } = useDropzone({ onDrop });

    return (
        <div className="flex flex-col bg-base-100 rounded-sm shadow flex-grow flex-shrink-0 mb-4">
            <div className="flex p-3 border-b border-base-300 mb-2 items-center flex-shrink-0">
                <Icon.Upload size="20px" strokeWidth="1.5px" />

                <div className="pl-3 font-700 text-lg">Upload a network policy YAML</div>
            </div>
            <div className="mb-3 px-3 font-600 text-lg leading-loose text-base-600">
                Upload your network policies to quickly preview your environment under different
                policy configurations and time windows. When ready, apply the network policies
                directly or share them with your team.
            </div>
            <div
                {...getRootProps()}
                className="bg-warning-100 border border-dashed border-warning-500 cursor-pointer flex flex-col h-full hover:bg-warning-200 justify-center min-h-32 mt-3 outline-none py-3 self-center uppercase w-full"
            >
                <input {...getInputProps()} />
                <div className="flex flex-shrink-0 flex-col">
                    <div
                        className="mt-3 h-18 w-18 self-center rounded-full flex items-center justify-center flex-shrink-0"
                        style={{
                            background: fileUploadColors.BACKGROUND_COLOR,
                            color: fileUploadColors.ICON_COLOR,
                        }}
                    >
                        <Icon.Upload className="h-8 w-8" strokeWidth="1.5px" />
                    </div>
                    <span className="font-700 mt-3 text-center text-warning-800">
                        Upload and simulate network policy YAML
                    </span>
                </div>
            </div>
        </div>
    );
}

const mapDispatchToProps = {
    setNetworkPolicyModification: sidepanelActions.setNetworkPolicyModification,
    setNetworkPolicyModificationState: sidepanelActions.setNetworkPolicyModificationState,
    setNetworkPolicyModificationSource: sidepanelActions.setNetworkPolicyModificationSource,
    setNetworkPolicyModificationName: sidepanelActions.setNetworkPolicyModificationName,

    setSidePanelStage: sidepanelActions.setSidePanelStage,

    addToast: notificationActions.addNotification,
    removeToast: notificationActions.removeOldestNotification,
};

export default connect(null, mapDispatchToProps)(UploadNetworkPolicySection);
