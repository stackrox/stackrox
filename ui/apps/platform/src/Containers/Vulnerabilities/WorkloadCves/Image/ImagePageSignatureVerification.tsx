import React from 'react';
import { Divider, Flex, FlexItem, Label, PageSection, Text } from '@patternfly/react-core';
import { Table, TableText, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';
import { CheckCircleIcon, ExclamationCircleIcon } from '@patternfly/react-icons';
import { Link } from 'react-router-dom';
import { integrationsPath } from 'routePaths';

import DateDistance from 'Components/DateDistance';
import useIntegrationPermissions from 'Containers/Integrations/hooks/useIntegrationPermissions';
import { SignatureVerificationResult, VerifiedStatus } from '../../types';

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
    const permissions = useIntegrationPermissions();
    const getIntegrationDetailsUrl = (verifierId: string): string => {
        return `${integrationsPath}/signatureIntegrations/signature/view/${verifierId}`;
    };

    const renderIntegrationCell = (result: SignatureVerificationResult) => {
        const displayName = result.verifierName || result.verifierId;

        // Show as link only if user has permissions.
        if (permissions.signatureIntegrations.read) {
            return <Link to={getIntegrationDetailsUrl(result.verifierId)}>{displayName}</Link>;
        }

        // Fallback to plain text.
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
