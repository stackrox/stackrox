import React from 'react';
import { Alert, Button, Flex, Modal, Text } from '@patternfly/react-core';

import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

export type GenerateSbomModalProps = {
    onClose: () => void;
    onGenerateSbom: () => void;
    imageFullName: string;
};

function GenerateSbomModal(props: GenerateSbomModalProps) {
    const { onClose, onGenerateSbom, imageFullName } = props;

    const isLoading = false;
    const isSuccess = false;
    const isError = false;
    const error = undefined;

    function onClickGenerateSbom() {
        // TODO Implement API call
        onGenerateSbom();
    }

    return (
        <Modal
            isOpen
            onClose={onClose}
            title="Generate Software Bill of Materials (SBOM)"
            variant="medium"
            actions={[
                <Button
                    key="generate-sbom-action"
                    isLoading={isLoading}
                    isDisabled={isLoading}
                    onClick={onClickGenerateSbom}
                >
                    Generate SBOM
                </Button>,
                <Button key="close-modal" variant="link" onClick={onClose}>
                    Cancel
                </Button>,
            ]}
        >
            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsLg' }}>
                <Text>
                    Generate and download the Software Bill of Materials (SBOM) in SBDX 2.3 format.
                    This file contains a detailed list of all components and dependencies included
                    in the image.
                </Text>
                <Text>
                    Selected image: <em>{imageFullName}</em>
                </Text>
                {isSuccess && (
                    <Alert
                        isInline
                        component="p"
                        variant="success"
                        title="Software Bill of Materials (SBOM) generated successfully"
                    />
                )}
                {isLoading && (
                    <Alert
                        isInline
                        component="p"
                        variant="info"
                        title="Generating, please do not navigate away from this modal"
                    />
                )}
                {isError && (
                    <Alert
                        isInline
                        component="p"
                        variant="danger"
                        title="There was an error generating the Software Bill of Materials (SBOM)"
                    >
                        {getAxiosErrorMessage(error)}
                    </Alert>
                )}
            </Flex>
        </Modal>
    );
}

export default GenerateSbomModal;
