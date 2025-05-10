import React from 'react';
import { Divider, PageSection, Text } from '@patternfly/react-core';
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';

import DateDistance from 'Components/DateDistance';
import { SignatureVerificationResult } from '../../types';

export type ImagePageSignatureVerificationProps = {
    results?: SignatureVerificationResult[];
};

function ImagePageSignatureVerification({ results }: ImagePageSignatureVerificationProps) {
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
                                <Th>Description</Th>
                                <Th>Verificaton time</Th>
                            </Tr>
                        </Thead>

                        {results?.map((result) => {
                            return (
                                <Tbody
                                    key={result?.verifierId}
                                    style={{
                                        borderBottom: '1px solid var(--pf-v5-c-table--BorderColor)',
                                    }}
                                >
                                    <Tr>
                                        <Td dataLabel="Integration">{result?.verifierId}</Td>
                                        <Td dataLabel="Status">{result?.status}</Td>
                                        <Td dataLabel="Description">{result?.description}</Td>
                                        <Td dataLabel="Verificaton time">
                                            <DateDistance date={result?.verificationTime} />
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
