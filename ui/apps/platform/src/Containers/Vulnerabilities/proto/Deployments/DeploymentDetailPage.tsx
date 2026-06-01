import { useParams } from 'react-router-dom-v5-compat';
import {
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    Flex,
    FlexItem,
    Label,
    PageSection,
    Spinner,
    Title,
} from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { Link } from 'react-router-dom-v5-compat';

import { vulnerabilitiesPrototypeDeploymentsPath } from 'routePaths';

import { useDeploymentDetail } from './useDeploymentDetail';
import type { ProtoDeploymentImage } from './useDeploymentDetail';

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

    return (
        <>
            <PageSection hasBodyWrapper={false}>
                <Breadcrumb>
                    <BreadcrumbItem>
                        <Link to={vulnerabilitiesPrototypeDeploymentsPath}>
                            Deployments
                        </Link>
                    </BreadcrumbItem>
                    <BreadcrumbItem isActive>
                        {data?.name ?? deploymentId}
                    </BreadcrumbItem>
                </Breadcrumb>
            </PageSection>

            <PageSection hasBodyWrapper={false}>
                {loading && (
                    <Bullseye>
                        <Spinner />
                    </Bullseye>
                )}

                {error && <p>Error loading deployment: {error.message}</p>}

                {!loading && !error && data && (
                    <>
                        <Flex>
                            <FlexItem>
                                <Title headingLevel="h1">{data.name}</Title>
                            </FlexItem>
                        </Flex>
                        <Flex spaceItems={{ default: 'spaceItemsMd' }}>
                            <FlexItem>
                                <Label color="blue">{data.cluster}</Label>
                            </FlexItem>
                            <FlexItem>
                                <Label color="grey">{data.namespace}</Label>
                            </FlexItem>
                        </Flex>
                    </>
                )}
            </PageSection>

            <PageSection hasBodyWrapper={false}>
                <Title headingLevel="h2">Images</Title>
                <Table aria-label="Deployment images" variant="compact">
                    <Thead>
                        <Tr>
                            <Th>Image</Th>
                            <Th>CVEs</Th>
                            <Th>Top Severity</Th>
                            <Th>Fixable</Th>
                        </Tr>
                    </Thead>
                    <Tbody>
                        {images.map((img) => (
                            <Tr key={img.imageId}>
                                <Td dataLabel="Image">
                                    <Link
                                        to={`/main/vulnerabilities/prototype/images/${encodeURIComponent(img.imageId)}`}
                                    >
                                        {img.imageName || img.imageId}
                                    </Link>
                                </Td>
                                <Td dataLabel="CVEs">{img.cveCount}</Td>
                                <Td dataLabel="Top Severity">
                                    <Label
                                        color={severityColor(img.topSeverity)}
                                    >
                                        {severityLabel(img.topSeverity)}
                                    </Label>
                                </Td>
                                <Td dataLabel="Fixable">
                                    {img.fixable ? 'Yes' : 'No'}
                                </Td>
                            </Tr>
                        ))}
                        {!loading && images.length === 0 && (
                            <Tr>
                                <Td colSpan={4}>
                                    <Bullseye>
                                        No images found for this deployment
                                    </Bullseye>
                                </Td>
                            </Tr>
                        )}
                    </Tbody>
                </Table>
            </PageSection>
        </>
    );
}

export default DeploymentDetailPage;
