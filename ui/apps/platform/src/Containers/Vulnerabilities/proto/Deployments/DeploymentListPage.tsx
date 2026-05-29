import { Link } from 'react-router-dom-v5-compat';
import {
    Bullseye,
    Label,
    PageSection,
    Spinner,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
    Title,
} from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import { vulnerabilitiesPrototypeDeploymentsPath } from 'routePaths';

import ProtoNav from '../ProtoNav';
import { useDeploymentList } from './useDeploymentList';
import type { ProtoDeploymentListItem } from './useDeploymentList';

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

function DeploymentListPage() {
    const { data, loading, error } = useDeploymentList(50, 0);

    const deployments: ProtoDeploymentListItem[] = data?.deployments ?? [];
    const totalCount = data?.totalCount ?? 0;

    return (
        <>
            <PageSection hasBodyWrapper={false}>
                <Title headingLevel="h1">CVE Prototype</Title>
            </PageSection>
            <PageSection hasBodyWrapper={false}>
                <ProtoNav />
            </PageSection>
            <PageSection hasBodyWrapper={false}>
                <Toolbar>
                    <ToolbarContent>
                        <ToolbarItem>
                            {loading && <Spinner size="md" />}
                            {!loading &&
                                `${deployments.length} of ${totalCount} Deployments`}
                        </ToolbarItem>
                    </ToolbarContent>
                </Toolbar>

                {error && (
                    <Bullseye>
                        <p>Error loading deployments: {error.message}</p>
                    </Bullseye>
                )}

                <Table aria-label="Prototype deployment list" variant="compact">
                    <Thead>
                        <Tr>
                            <Th>Deployment</Th>
                            <Th>Cluster</Th>
                            <Th>Namespace</Th>
                            <Th>Images</Th>
                            <Th>CVEs</Th>
                            <Th>Top Severity</Th>
                            <Th>Fixable</Th>
                        </Tr>
                    </Thead>
                    <Tbody>
                        {deployments.map((dep) => (
                            <Tr key={dep.id}>
                                <Td dataLabel="Deployment">
                                    <Link
                                        to={`${vulnerabilitiesPrototypeDeploymentsPath}/${encodeURIComponent(dep.id)}`}
                                    >
                                        {dep.name}
                                    </Link>
                                </Td>
                                <Td dataLabel="Cluster">{dep.cluster}</Td>
                                <Td dataLabel="Namespace">{dep.namespace}</Td>
                                <Td dataLabel="Images">{dep.imageCount}</Td>
                                <Td dataLabel="CVEs">{dep.cveCount}</Td>
                                <Td dataLabel="Top Severity">
                                    <Label color={severityColor(dep.topSeverity)}>
                                        {severityLabel(dep.topSeverity)}
                                    </Label>
                                </Td>
                                <Td dataLabel="Fixable">
                                    {dep.fixable ? 'Yes' : 'No'}
                                </Td>
                            </Tr>
                        ))}
                        {!loading && deployments.length === 0 && (
                            <Tr>
                                <Td colSpan={7}>
                                    <Bullseye>No deployments found</Bullseye>
                                </Td>
                            </Tr>
                        )}
                    </Tbody>
                </Table>
            </PageSection>
        </>
    );
}

export default DeploymentListPage;
