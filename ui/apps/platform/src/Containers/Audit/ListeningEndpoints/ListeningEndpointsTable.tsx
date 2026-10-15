import { useCallback, useState } from 'react';
import { Link } from 'react-router-dom-v5-compat';
import { Bullseye, Card, Content, Pagination, Spinner } from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import { riskWorkloadsBasePath } from 'routePaths';
import { getListeningEndpointsForDeployment } from 'services/ProcessListeningOnPortsService';
import type { ProcessListeningOnPort } from 'services/ProcessListeningOnPortsService';
import { l4ProtocolLabels } from 'constants/networkFlow';
import type { ListDeployment } from 'types/deployment.proto';
import ExpandRowTh from 'Components/ExpandRowTh';
import useRestQuery from 'hooks/useRestQuery';
import type { GetSortParams } from 'hooks/useURLSort';
import { DEFAULT_ENDPOINTS_PER_PAGE } from './hooks/useDeploymentListeningEndpoints';

function EmbeddedTable({
    deploymentId,
    preloadedEndpoints,
    totalEndpoints,
}: {
    deploymentId: string;
    preloadedEndpoints: ProcessListeningOnPort[];
    totalEndpoints: number;
}) {
    const [page, setPage] = useState(1);
    const [perPage, setPerPage] = useState(DEFAULT_ENDPOINTS_PER_PAGE);

    const queryFn = useCallback(() => {
        if (page === 1 && perPage === DEFAULT_ENDPOINTS_PER_PAGE) {
            return Promise.resolve(preloadedEndpoints);
        }
        const { request, cancel } = getListeningEndpointsForDeployment(deploymentId, {
            offset: (page - 1) * perPage,
            limit: perPage,
        });
        return {
            request: request.then((r) => r.listeningEndpoints),
            cancel,
        };
    }, [deploymentId, page, perPage, preloadedEndpoints]);

    const { data: endpoints, isLoading, error } = useRestQuery(queryFn);

    if (error) {
        return (
            <Content component="p" className="pf-v6-u-p-md">
                Error loading listening endpoints
            </Content>
        );
    }

    if (isLoading || !endpoints) {
        return (
            <Bullseye>
                <Spinner size="md" aria-label="Loading listening endpoints" />
            </Bullseye>
        );
    }

    return (
        <>
            <Table isNested aria-label="Listening endpoints for deployment">
                <Thead noWrap>
                    <Tr>
                        <Th width={30}>Exec file path</Th>
                        <Th>PID</Th>
                        <Th>Port</Th>
                        <Th>Protocol</Th>
                        <Th width={30}>Pod ID</Th>
                        <Th width={20}>Container name</Th>
                    </Tr>
                </Thead>
                <Tbody>
                    {endpoints.map(({ podId, endpoint, signal, containerName }) => (
                        <Tr key={`${deploymentId}/${podId}/${endpoint.port}`}>
                            <Td dataLabel="Exec file path">{signal.execFilePath}</Td>
                            <Td dataLabel="PID">{signal.pid}</Td>
                            <Td dataLabel="Port">{endpoint.port}</Td>
                            <Td dataLabel="Protocol">{l4ProtocolLabels[endpoint.protocol]}</Td>
                            <Td dataLabel="Pod ID">{podId}</Td>
                            <Td dataLabel="Container name">{containerName}</Td>
                        </Tr>
                    ))}
                </Tbody>
            </Table>
            {totalEndpoints > DEFAULT_ENDPOINTS_PER_PAGE && (
                <Pagination
                    itemCount={totalEndpoints}
                    page={page}
                    perPage={perPage}
                    onSetPage={(_, newPage) => setPage(newPage)}
                    onPerPageSelect={(_, newPerPage) => {
                        setPerPage(newPerPage);
                        setPage(1);
                    }}
                    variant="bottom"
                    isCompact
                />
            )}
        </>
    );
}

type DeploymentWithEndpoints = ListDeployment & {
    listeningEndpoints: ProcessListeningOnPort[];
    totalListeningEndpoints: number;
};

export type ListeningEndpointsTableProps = {
    deployments: DeploymentWithEndpoints[];
    getSortParams: GetSortParams;
};

function ListeningEndpointsTable({ deployments, getSortParams }: ListeningEndpointsTableProps) {
    // Only one deployment can be expanded at a time. This tracks the id of the
    // currently expanded deployment, or null when none is expanded.
    const [expandedDeploymentId, setExpandedDeploymentId] = useState<string | null>(null);

    return (
        <Table variant="compact" aria-label="Deployment results">
            <Thead noWrap>
                <Tr>
                    <ExpandRowTh />
                    <Th width={30} sort={getSortParams('Deployment')}>
                        Deployment
                    </Th>
                    <Th width={20} sort={getSortParams('Cluster')}>
                        Cluster
                    </Th>
                    <Th width={30} sort={getSortParams('Namespace')}>
                        Namespace
                    </Th>
                    <Th>Count</Th>
                </Tr>
            </Thead>
            {deployments.map(
                (
                    { id, name, namespace, cluster, listeningEndpoints, totalListeningEndpoints },
                    rowIndex
                ) => {
                    const isExpanded = expandedDeploymentId === id;
                    return (
                        <Tbody key={id} isExpanded={isExpanded}>
                            <Tr>
                                <Td
                                    expand={{
                                        rowIndex,
                                        isExpanded,
                                        onToggle: () =>
                                            setExpandedDeploymentId(isExpanded ? null : id),
                                    }}
                                />
                                <Td dataLabel="Deployment">
                                    <Link to={`${riskWorkloadsBasePath}/${id}`}>{name}</Link>
                                </Td>
                                <Td dataLabel="Cluster">{cluster}</Td>
                                <Td dataLabel="Namespace">{namespace}</Td>
                                <Td dataLabel="Count">{totalListeningEndpoints}</Td>
                            </Tr>
                            <Tr isExpanded={isExpanded}>
                                <Td colSpan={5}>
                                    <Card className="pf-v6-u-m-md">
                                        {totalListeningEndpoints > 0 ? (
                                            <EmbeddedTable
                                                deploymentId={id}
                                                preloadedEndpoints={listeningEndpoints}
                                                totalEndpoints={totalListeningEndpoints}
                                            />
                                        ) : (
                                            <Content component="p" className="pf-v6-u-p-md">
                                                This deployment does not have any reported listening
                                                endpoints
                                            </Content>
                                        )}
                                    </Card>
                                </Td>
                            </Tr>
                        </Tbody>
                    );
                }
            )}
        </Table>
    );
}

export default ListeningEndpointsTable;
