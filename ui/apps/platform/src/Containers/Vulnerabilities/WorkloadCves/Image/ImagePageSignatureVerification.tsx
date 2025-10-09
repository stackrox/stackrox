import React, { useEffect } from 'react';
import { Divider, Flex, FlexItem, Label, PageSection, Text } from '@patternfly/react-core';
import { Table, TableText, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';
import { CheckCircleIcon, ExclamationCircleIcon } from '@patternfly/react-icons';
import { Link } from 'react-router-dom';

import DateDistance from 'Components/DateDistance';
import { SignatureVerificationResult, VerifiedStatus } from '../../types';
import useIntegrations from '../../../Integrations/hooks/useIntegrations';
import useFetchIntegrations from '../../../Integrations/hooks/useFetchIntegrations';
import useIntegrationPermissions from '../../../Integrations/hooks/useIntegrationPermissions';
import { integrationsPath } from 'routePaths';

export type ImagePageSignatureVerificationProps = {
    results?: SignatureVerificationResult[];
};

const renderedStatus = new Map<VerifiedStatus, string>([
    ['CORRUPTED_SIGNATURE', 'Corrupted signature'],
    ['FAILED_VERIFICATION', 'Failed verification'],
    ['GENERIC_ERROR', 'Generic error'],
    ['INVALID_SIGNATURE_ALGO', 'Invalid signature algorithm'],
    ['UNSET', 'Unset'],
]);

function getStatusMessage({ status, description }: SignatureVerificationResult) {
    if (status === 'VERIFIED') {
        return (
            <Flex direction={{ default: 'column' }}>
                <FlexItem>
                    <Label color="green" icon={<CheckCircleIcon />}>
                        Verified
                    </Label>
                </FlexItem>
            </Flex>
        );
    }

    return (
        <Flex direction={{ default: 'column' }}>
            <FlexItem>
                <Label color="red" icon={<ExclamationCircleIcon />}>
                    {renderedStatus.get(status) ?? status}
                </Label>
            </FlexItem>
            <FlexItem>
                <TableText wrapModifier="wrap">{description}</TableText>
            </FlexItem>
        </Flex>
    );
}

function ImagePageSignatureVerification({ results }: ImagePageSignatureVerificationProps) {
    const signatureIntegrations = useIntegrations({
        source: 'signatureIntegrations',
        type: 'signature',
    });
    const fetchSignatureIntegrations = useFetchIntegrations('signatureIntegrations');
    const { signatureIntegrations: integrationPermissions } = useIntegrationPermissions();

    useEffect(() => {
        if (integrationPermissions.read) {
            fetchSignatureIntegrations();
        }
    }, [fetchSignatureIntegrations, integrationPermissions.read]);

    const getIntegrationDetailsUrl = (verifierId: string): string => {
        return `${integrationsPath}/signatureIntegrations/signature/view/${verifierId}`;
    };

    const getIntegrationDisplayName = (verifierId: string): string => {
        if (!integrationPermissions.read) {
            return verifierId;
        }

        const integration = signatureIntegrations.find(
            (integration) => integration.id === verifierId
        );
        return integration?.name || verifierId;
    };

    const renderIntegrationCell = (result: SignatureVerificationResult) => {
        const displayName = getIntegrationDisplayName(result.verifierId);
        const hasReadAccess = integrationPermissions.read;
        const integration = signatureIntegrations.find(
            (integration) => integration.id === result.verifierId
        );

        // Show as link only if user has permissions and we found the integration
        if (hasReadAccess && integration) {
            return <Link to={getIntegrationDetailsUrl(result.verifierId)}>{displayName}</Link>;
        }

        // Fallback to plain text
        return displayName;
    };

    return (
        <>
            <PageSection component="div" variant="light" className="pf-v5-u-py-md pf-v5-u-px-xl">
                <Text>Review the signature verification results for this image</Text>
            </PageSection>
            <Divider component="div" />
            <PageSection
                className="pf-v5-u-display-flex pf-v5-u-flex-direction-column pf-v5-u-flex-grow-1"
                component="div"
            >
                <div className="pf-v5-u-background-color-100 pf-v5-u-pt-sm">
                    <Table borders={false} variant="compact">
                        <Thead noWrap>
                            <Tr>
                                <Th>Integration</Th>
                                <Th>Status</Th>
                                <Th>Verification time</Th>
                            </Tr>
                        </Thead>

                        {results?.map((result) => {
                            return (
                                <Tbody
                                    key={result.verifierId}
                                    style={{
                                        borderBottom: '1px solid var(--pf-v5-c-table--BorderColor)',
                                    }}
                                >
                                    <Tr>
                                        <Td dataLabel="Integration">
                                            {renderIntegrationCell(result)}
                                        </Td>
                                        <Td dataLabel="Status">{getStatusMessage(result)}</Td>
                                        <Td dataLabel="Verification time">
                                            <DateDistance date={result.verificationTime} />
                                        </Td>
                                    </Tr>
                                </Tbody>
                            );
                        })}
                    </Table>
                </div>
            </PageSection>
        </>
    );
}

export default ImagePageSignatureVerification;
