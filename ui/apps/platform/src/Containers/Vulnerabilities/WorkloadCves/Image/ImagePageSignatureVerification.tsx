import { Content, Divider, Flex, FlexItem, Label, PageSection } from '@patternfly/react-core';
import { Table, TableText, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { CheckCircleIcon, ExclamationCircleIcon } from '@patternfly/react-icons';

import DateDistance from 'Components/DateDistance';
import type { SignatureVerificationResult, VerifiedStatus } from '../../types';

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
    return (
        <>
            <PageSection component="div">
                <Content component="p">
                    Review the signature verification results for this image
                </Content>
            </PageSection>
            <Divider component="div" />
            <PageSection component="div">
                <Table variant="compact">
                    <Thead noWrap>
                        <Tr>
                            <Th>Integration</Th>
                            <Th>Status</Th>
                            <Th>Verification time</Th>
                        </Tr>
                    </Thead>

                    {results?.map((result) => {
                        return (
                            <Tbody key={result.verifierId}>
                                <Tr>
                                    <Td dataLabel="Integration">{result.verifierId}</Td>
                                    <Td dataLabel="Status">{getStatusMessage(result)}</Td>
                                    <Td dataLabel="Verification time">
                                        <DateDistance date={result.verificationTime} />
                                    </Td>
                                </Tr>
                            </Tbody>
                        );
                    })}
                </Table>
            </PageSection>
        </>
    );
}

export default ImagePageSignatureVerification;
