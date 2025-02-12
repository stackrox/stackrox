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
    Title,
} from '@patternfly/react-core';
import Raven from 'raven-js';

import TechPreviewLabel from 'Components/PatternFly/TechPreviewLabel';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import useAnalytics, { IMAGE_SBOM_GENERATED } from 'hooks/useAnalytics';
import useRestMutation from 'hooks/useRestMutation';
import { generateAndSaveSbom } from 'services/ImageSbomService';

export function getSbomGenerationStatusMessage({
    isScannerV4Enabled,
    hasScanMessage,
}: {
    isScannerV4Enabled: boolean;
    hasScanMessage: boolean;
}): string | undefined {
    if (!isScannerV4Enabled) {
        return 'SBOM generation requires Scanner V4';
    }

    if (hasScanMessage) {
        return 'SBOM generation is unavailable due to incomplete scan data';
    }

    return undefined;
}

export type GenerateSbomModalProps = {
    onClose: () => void;
    imageName: string;
};

function GenerateSbomModal(props: GenerateSbomModalProps) {
    const { analyticsTrack } = useAnalytics();
    const { onClose, imageName } = props;
    const { mutate, isLoading, isSuccess, isError, error } = useRestMutation(generateAndSaveSbom, {
        onSuccess: () => analyticsTrack(IMAGE_SBOM_GENERATED),
        onError: (err) => Raven.captureException(err),
    });

    function onClickGenerateSbom() {
        mutate({ imageName });
    }

    return (
        <Modal
            isOpen
            onClose={onClose}
            variant="medium"
            header={
                <Flex
                    className="pf-v5-u-mr-md"
                    justifyContent={{ default: 'justifyContentSpaceBetween' }}
                    alignItems={{ default: 'alignItemsCenter' }}
                >
                    <Title headingLevel="h1">Generate Software Bill of Materials (SBOM)</Title>
                    <TechPreviewLabel />
                </Flex>
            }
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
                    Generate and download the Software Bill of Materials (SBOM) in SPDX 2.3 format.
                    This file contains a detailed list of all components and dependencies included
                    in the image.
                </Text>
                <Text>
                    (Generating SBOMs from scans delegated to secured clusters is currently not
                    supported.)
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
