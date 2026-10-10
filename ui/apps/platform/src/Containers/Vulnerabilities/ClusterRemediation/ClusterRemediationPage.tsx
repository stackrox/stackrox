import { useEffect, useState } from 'react';
import type { ReactElement } from 'react';
import {
    Alert,
    Button,
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

import PageTitle from 'Components/PageTitle';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import SeverityCountLabels from 'Containers/Vulnerabilities/components/SeverityCountLabels';
import type { ImageRemediation, SeverityCounts } from 'services/OperatorRemediationService';
import {
    createClusterRemediationReport,
    getClusterRemediationReport,
    type ClusterRemediationReport,
    type JobStatus,
} from 'services/ClusterRemediationService';

const POLL_INTERVAL_MS = 5000;

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
    return image.name || image.repository;
}

function ClusterRemediationPage(): ReactElement {
    const [runningOnly, setRunningOnly] = useState(true);
    const [jobId, setJobId] = useState<string | null>(null);
    const [status, setStatus] = useState<JobStatus | null>(null);
    const [report, setReport] = useState<ClusterRemediationReport | null>(null);
    const [error, setError] = useState<string | null>(null);
    const [starting, setStarting] = useState(false);

    const isRunning = starting || status === 'RUNNING';

    async function onGenerate() {
        setError(null);
        setReport(null);
        setStatus(null);
        setStarting(true);
        try {
            const job = await createClusterRemediationReport(runningOnly);
            setJobId(job.id);
            setStatus(job.status);
        } catch (e) {
            setError(getAxiosErrorMessage(e));
        } finally {
            setStarting(false);
        }
    }

    useEffect(() => {
        if (!jobId || status !== 'RUNNING') {
            return undefined;
        }
        const timer = setInterval(() => {
            getClusterRemediationReport(jobId)
                .then((job) => {
                    setStatus(job.status);
                    if (job.status === 'COMPLETE') {
                        setReport(job.report ?? null);
                    } else if (job.status === 'FAILED') {
                        setError(job.error || 'Report generation failed');
                    }
                })
                .catch((e) => {
                    setError(getAxiosErrorMessage(e));
                    setStatus('FAILED');
                });
        }, POLL_INTERVAL_MS);
        return () => clearInterval(timer);
    }, [jobId, status]);

    const header = report
        ? report.targetVersion
            ? `OpenShift ${report.currentVersion} → ${report.targetVersion}`
            : `OpenShift ${report.currentVersion} — ${report.noUpdateReason || 'no update available'}`
        : 'Cluster upgrade CVE fixes per image';

    return (
        <PageSection>
            <PageTitle title="Cluster upgrade remediation" />
            <Stack hasGutter>
                <StackItem>
                    <Title headingLevel="h1">{header}</Title>
                    <Content component="p">
                        Generate a report of which OpenShift component images the next update changes,
                        and the CVEs each image fixes or introduces.
                    </Content>
                </StackItem>
                <StackItem>
                    <Flex alignItems={{ default: 'alignItemsCenter' }} spaceItems={{ default: 'spaceItemsLg' }}>
                        <Switch
                            id="cluster-remediation-running-only"
                            label="Currently running only"
                            isChecked={runningOnly}
                            isDisabled={isRunning}
                            onChange={(_e, checked) => setRunningOnly(checked)}
                        />
                        <Button variant="primary" onClick={onGenerate} isLoading={isRunning} isDisabled={isRunning}>
                            {isRunning ? 'Generating…' : 'Generate report'}
                        </Button>
                    </Flex>
                </StackItem>
                <StackItem>
                    {isRunning && (
                        <Flex alignItems={{ default: 'alignItemsCenter' }} spaceItems={{ default: 'spaceItemsSm' }}>
                            <Spinner size="md" />
                            <Content component="small">
                                Generating report — this can take a few minutes while candidate images are scanned.
                            </Content>
                        </Flex>
                    )}
                    {error && (
                        <Alert variant="danger" isInline title="Could not generate report">
                            {error}
                        </Alert>
                    )}
                    {!isRunning && report && (
                        <Table aria-label="Cluster upgrade remediation" variant="compact">
                            <Thead>
                                <Tr>
                                    <Th>Component image</Th>
                                    <Th>Current CVEs by severity</Th>
                                    <Th>Changes (fixed / new)</Th>
                                </Tr>
                            </Thead>
                            <Tbody>
                                {(report.images ?? []).map((image) => (
                                    <Tr key={`${image.repository}/${image.name}`}>
                                        <Td dataLabel="Component image">{imageLabel(image)}</Td>
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
                    {!isRunning && report && (report.images ?? []).length === 0 && (
                        <Content component="p">No component images to display.</Content>
                    )}
                </StackItem>
            </Stack>
        </PageSection>
    );
}

export default ClusterRemediationPage;
