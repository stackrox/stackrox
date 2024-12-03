import React from 'react';
import {
    Alert,
    Button,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    Modal,
    Text,
} from '@patternfly/react-core';
import Raven from 'raven-js';

import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import useRestMutation from 'hooks/useRestMutation';
import { generateAndSaveSbom } from 'services/ImageSbomService';

export type GenerateSbomModalProps = {
    onClose: () => void;
    onGenerateSbom: () => void;
    imageName: string;
};

function GenerateSbomModal(props: GenerateSbomModalProps) {
    const { onClose, onGenerateSbom, imageName } = props;
    const { mutate, isLoading, isSuccess, isError, error } = useRestMutation(generateAndSaveSbom, {
        onSuccess: onGenerateSbom,
        onError: (err) => Raven.captureException(err),
    });

    function onClickGenerateSbom() {
        mutate({ imageName });
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
                    Close
                </Button>,
            ]}
        >
            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsLg' }}>
                <Text>
                    Generate and download the Software Bill of Materials (SBOM) in SBDX 2.3 format.
                    This file contains a detailed list of all components and dependencies included
                    in the image.
                </Text>
                <DescriptionList isHorizontal>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Selected image:</DescriptionListTerm>
                        <DescriptionListDescription>{imageName}</DescriptionListDescription>
                    </DescriptionListGroup>
                </DescriptionList>
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
