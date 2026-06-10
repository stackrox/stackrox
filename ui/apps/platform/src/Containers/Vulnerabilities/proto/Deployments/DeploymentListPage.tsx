import { Link } from 'react-router-dom-v5-compat';
import {
    Bullseye,
    Label,
    Pagination,
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
import { usePagination } from '../usePagination';
import { useSort } from '../useSort';
import {
    SEVERITY_WIDTH,
    COUNT_WIDTH,
    TABLE_HEADER_STYLE,
    TABLE_CELL_STYLE,
} from '../utils/tableDefaults';
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

// Column keys: name, cluster (not sortable), namespace (not sortable), images (not sortable),
// cveCount, severity, fixable (not sortable)
const depSortColumns = ['name', '', '', '', 'cveCount', 'severity', ''];

function DeploymentListPage() {
    const { sortBy, sortDir, getThSortProps } = useSort(depSortColumns, 5);
    const { page, perPage, offset, onSetPage, onPerPageSelect } = usePagination(20);
    const { data, loading, error } = useDeploymentList(perPage, offset, sortBy, sortDir);

    const deployments: ProtoDeploymentListItem[] = data?.deployments ?? [];
    const totalCount = data?.totalCount ?? 0;

    return (
        <>
            <PageSection hasBodyWrapper={false}>
                <Title headingLevel="h1">Vuln Management V5</Title>
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

                <Table aria-label="Vuln Management V5 deployment list" variant="compact">
                    <Thead style={{ borderBottom: '2px solid var(--pf-v5-global--BorderColor--100)' }}>
                        <Tr>
                            <Th {...getThSortProps(0)} style={TABLE_HEADER_STYLE}>Deployment</Th>
                            <Th style={TABLE_HEADER_STYLE}>Cluster</Th>
                            <Th style={TABLE_HEADER_STYLE}>Namespace</Th>
                            <Th style={{ ...TABLE_HEADER_STYLE, width: `${COUNT_WIDTH}px` }}>Images</Th>
                            <Th {...getThSortProps(4)} style={{ ...TABLE_HEADER_STYLE, width: `${COUNT_WIDTH}px` }}>CVEs</Th>
                            <Th {...getThSortProps(5)} style={{ ...TABLE_HEADER_STYLE, width: `${SEVERITY_WIDTH}px` }}>Top Severity</Th>
                            <Th style={TABLE_HEADER_STYLE}>Fixable</Th>
                        </Tr>
                    </Thead>
                    <Tbody>
                        {deployments.map((dep) => (
                            <Tr key={dep.id}>
                                <Td dataLabel="Deployment" style={TABLE_CELL_STYLE}>
                                    <Link
                                        to={`${vulnerabilitiesPrototypeDeploymentsPath}/${encodeURIComponent(dep.id)}`}
                                    >
                                        {dep.name}
                                    </Link>
                                </Td>
                                <Td dataLabel="Cluster" style={TABLE_CELL_STYLE}>{dep.cluster}</Td>
                                <Td dataLabel="Namespace" style={TABLE_CELL_STYLE}>{dep.namespace}</Td>
                                <Td dataLabel="Images" style={TABLE_CELL_STYLE}>{dep.imageCount}</Td>
                                <Td dataLabel="CVEs" style={TABLE_CELL_STYLE}>{dep.cveCount}</Td>
                                <Td dataLabel="Top Severity" style={TABLE_CELL_STYLE}>
                                    <Label color={severityColor(dep.topSeverity)}>
                                        {severityLabel(dep.topSeverity)}
                                    </Label>
                                </Td>
                                <Td dataLabel="Fixable" style={TABLE_CELL_STYLE}>
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
                <Pagination
                    itemCount={totalCount}
                    perPage={perPage}
                    page={page}
                    onSetPage={onSetPage}
                    onPerPageSelect={onPerPageSelect}
                />
            </PageSection>
        </>
    );
}

export default DeploymentListPage;
