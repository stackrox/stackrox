import { useParams } from 'react-router-dom-v5-compat';
import {
    Bullseye,
    Label,
    PageSection,
    Spinner,
    Title,
} from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { Link } from 'react-router-dom-v5-compat';

import { vulnerabilitiesPrototypePath, vulnerabilitiesPrototypeDeploymentsPath } from 'routePaths';

import { useDeploymentDetail } from './useDeploymentDetail';
import type { ProtoDeploymentImage } from './useDeploymentDetail';
import { DetailPageLayout } from '../Components/DetailPageLayout';
import { TABLE_HEADER_STYLE, TABLE_CELL_STYLE } from '../utils/tableDefaults';

const severityNames: Record<number, string> = {
    0: 'Unknown',
    1: 'Low',
    2: 'Moderate',
    3: 'Important',
    4: 'Critical',
};

function severityColor(severity: number): 'red' | 'orange' | 'blue' | 'grey' {
    switch (severity) {
        case 4:
            return 'red';
        case 3:
            return 'orange';
        case 2:
            return 'blue';
        default:
            return 'grey';
    }
}

function severityLabel(severity: number): string {
    return severityNames[severity] ?? 'Unknown';
}

function DeploymentDetailPage() {
    const { deploymentId } = useParams<{ deploymentId: string }>();
    const { data, loading, error } = useDeploymentDetail(deploymentId ?? '');

    const images: ProtoDeploymentImage[] = data?.images ?? [];

    const breadcrumbs = [
        { label: 'Vulnerability Management V5', path: vulnerabilitiesPrototypePath },
        { label: 'Deployments', path: vulnerabilitiesPrototypeDeploymentsPath },
        { label: data?.name ?? deploymentId ?? '' },
    ];

    const subtitle = data ? `${data.cluster} / ${data.namespace}` : '';

    if (loading) {
        return (
            <PageSection hasBodyWrapper={false}>
                <Bullseye>
                    <Spinner />
                </Bullseye>
            </PageSection>
        );
    }

    if (error) {
        return (
            <PageSection hasBodyWrapper={false}>
                <p>Error loading deployment: {error.message}</p>
            </PageSection>
        );
    }

    if (!data) {
        return null;
    }

    return (
        <PageSection hasBodyWrapper={false}>
            <DetailPageLayout
                breadcrumbs={breadcrumbs}
                title={data.name}
                subtitle={subtitle}
            >
                <Title headingLevel="h2">Images</Title>
                <Table aria-label="Deployment images" variant="compact">
                    <Thead style={{ borderBottom: '2px solid var(--pf-global--BorderColor--100)' }}>
                        <Tr>
                            <Th style={TABLE_HEADER_STYLE}>Image</Th>
                            <Th style={TABLE_HEADER_STYLE}>CVEs</Th>
                            <Th style={TABLE_HEADER_STYLE}>Top Severity</Th>
                            <Th style={TABLE_HEADER_STYLE}>Fixable</Th>
                        </Tr>
                    </Thead>
                    <Tbody>
                        {images.map((img) => (
                            <Tr key={img.imageId}>
                                <Td dataLabel="Image" style={TABLE_CELL_STYLE}>
                                    <Link
                                        to={`/main/vulnerabilities/prototype/images/${encodeURIComponent(img.imageId)}`}
                                    >
                                        {img.imageName || img.imageId}
                                    </Link>
                                </Td>
                                <Td dataLabel="CVEs" style={TABLE_CELL_STYLE}>{img.cveCount}</Td>
                                <Td dataLabel="Top Severity" style={TABLE_CELL_STYLE}>
                                    <Label
                                        color={severityColor(img.topSeverity)}
                                    >
                                        {severityLabel(img.topSeverity)}
                                    </Label>
                                </Td>
                                <Td dataLabel="Fixable" style={TABLE_CELL_STYLE}>
                                    {img.fixable ? 'Yes' : 'No'}
                                </Td>
                            </Tr>
                        ))}
                        {images.length === 0 && (
                            <Tr>
                                <Td colSpan={4} style={TABLE_CELL_STYLE}>
                                    <Bullseye>
                                        No images found for this deployment
                                    </Bullseye>
                                </Td>
                            </Tr>
                        )}
                    </Tbody>
                </Table>
            </DetailPageLayout>
        </PageSection>
    );
}

export default DeploymentDetailPage;
