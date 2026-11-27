import { useCallback, useState } from 'react';
import type { ReactElement } from 'react';
import {
    Alert,
    Bullseye,
    Content,
    Flex,
    FlexItem,
    PageSection,
    Spinner,
    Stack,
    StackItem,
    Switch,
    Title,
} from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { Route, Routes, useParams } from 'react-router-dom-v5-compat';

import PageTitle from 'Components/PageTitle';
import useRestQuery from 'hooks/useRestQuery';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import SeverityCountLabels from 'Containers/Vulnerabilities/components/SeverityCountLabels';
import {
    getImageRemediation,
    type ImageRemediation,
    type SeverityCounts,
} from 'services/OperatorRemediationService';

const severityKeys = [
    { key: 'critical', label: 'Critical' },
    { key: 'important', label: 'Important' },
    { key: 'moderate', label: 'Moderate' },
    { key: 'low', label: 'Low' },
    { key: 'unknown', label: 'Unknown' },
] as const;

function Changes({ fixed, added }: { fixed: SeverityCounts; added: SeverityCounts }): ReactElement {
    const parts = severityKeys
        .map(({ key, label }) => {
            const f = fixed[key];
            const n = added[key];
            if (!f && !n) {
                return null;
            }
            const deltas = [f ? `-${f}` : null, n ? `+${n}` : null].filter(Boolean).join(' ');
            return (
                <FlexItem key={key}>
                    <Content component="small">{`${label}: ${deltas}`}</Content>
                </FlexItem>
            );
        })
        .filter(Boolean);
    if (parts.length === 0) {
        return <Content component="small">No change</Content>;
    }
    return <Flex spaceItems={{ default: 'spaceItemsMd' }}>{parts}</Flex>;
}

function imageLabel(image: ImageRemediation): string {
    return image.name ? `${image.repository} (${image.name})` : image.repository;
}

function OperatorRemediationView(): ReactElement {
    const { imageId } = useParams();
    const [runningOnly, setRunningOnly] = useState(false);

    const requestFn = useCallback(
        () => getImageRemediation(imageId ?? '', runningOnly),
        [imageId, runningOnly]
    );
    const { data, isLoading, error } = useRestQuery(requestFn);

    const title = data?.operatorPackage || 'Operator image remediation';
    const subtitle = data
        ? data.updateVersion
            ? `${data.installedVersion} → ${data.updateVersion}`
            : `${data.installedVersion} — ${data.noUpdateReason || 'no update available'}`
        : '';

    return (
        <PageSection>
            <PageTitle title="Operator image remediation" />
            <Stack hasGutter>
                <StackItem>
                    <Title headingLevel="h1">{title}</Title>
                    {subtitle && <Content component="p">{subtitle}</Content>}
                </StackItem>
                <StackItem>
                    <Switch
                        id="operator-remediation-running-only"
                        label="Currently running only"
                        isChecked={runningOnly}
                        isDisabled={isLoading}
                        onChange={(_e, checked) => setRunningOnly(checked)}
                    />
                </StackItem>
                <StackItem>
                    {isLoading && (
                        <Bullseye>
                            <Spinner />
                        </Bullseye>
                    )}
                    {error && (
                        <Alert variant="danger" isInline title="Could not load remediation data">
                            {getAxiosErrorMessage(error)}
                        </Alert>
                    )}
                    {!isLoading && !error && data && (
                        <Table aria-label="Operator image remediation" variant="compact">
                            <Thead>
                                <Tr>
                                    <Th>Image</Th>
                                    <Th>Current CVEs by severity</Th>
                                    <Th>Changes (fixed / new)</Th>
                                </Tr>
                            </Thead>
                            <Tbody>
                                {(data.images ?? []).map((image) => (
                                    <Tr key={`${image.repository}/${image.name}`}>
                                        <Td dataLabel="Image">{imageLabel(image)}</Td>
                                        <Td dataLabel="Current CVEs by severity">
                                            <SeverityCountLabels
                                                criticalCount={image.current.critical}
                                                importantCount={image.current.important}
                                                moderateCount={image.current.moderate}
                                                lowCount={image.current.low}
                                                unknownCount={image.current.unknown}
                                            />
                                        </Td>
                                        <Td dataLabel="Changes (fixed / new)">
                                            <Changes fixed={image.fixed} added={image.newCves} />
                                        </Td>
                                    </Tr>
                                ))}
                            </Tbody>
                        </Table>
                    )}
                    {!isLoading && !error && data && (data.images ?? []).length === 0 && (
                        <Content component="p">No operator images to display.</Content>
                    )}
                </StackItem>
            </Stack>
        </PageSection>
    );
}

function OperatorRemediationPage(): ReactElement {
    // The page is registered as a splat route; declare the :imageId segment so useParams() in the
    // view is populated (mirrors WorkloadCvesPage's nested routes).
    return (
        <Routes>
            <Route path=":imageId" element={<OperatorRemediationView />} />
        </Routes>
    );
}

export default OperatorRemediationPage;
