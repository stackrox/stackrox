import { useCallback, useState } from 'react';
import type { Dispatch, SetStateAction } from 'react';
import { Link } from 'react-router-dom-v5-compat';
import { Bullseye, Card, Content, Pagination, Spinner } from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import { riskWorkloadsBasePath } from 'routePaths';
import { getListeningEndpointsForDeployment } from 'services/ProcessListeningOnPortsService';
import type { ProcessListeningOnPort } from 'services/ProcessListeningOnPortsService';
import { l4ProtocolLabels } from 'constants/networkFlow';
import type { ListDeployment } from 'types/deployment.proto';
import useSet from 'hooks/useSet';
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
    areAllRowsExpanded: boolean;
    setAllRowsExpanded: Dispatch<SetStateAction<boolean>>;
};

function ListeningEndpointsTable({
    deployments,
    getSortParams,
    areAllRowsExpanded,
    setAllRowsExpanded,
}: ListeningEndpointsTableProps) {
    // This is used to track which table rows are in the -opposite- state
    // of the passed expanded state for the entire table.
    const invertedExpansionRowSet = useSet<string>();

    return (
        <Table variant="compact" aria-label="Deployment results">
            <Thead noWrap>
                <Tr>
                    <Th
                        expand={{
                            // Possible PF bug? This boolean seems to need to be inverted based on the render output
                            areAllExpanded: !areAllRowsExpanded,
                            // TODO Awkward type assertion here is fixed in PF 5 https://github.com/patternfly/patternfly-react/issues/8330
                            collapseAllAriaLabel: 'Expand or collapse all rows' as '',
                            onToggle: () => {
                                setAllRowsExpanded((prev) => !prev);
                                invertedExpansionRowSet.clear();
                            },
                        }}
                        width={10}
                    />
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
                    // A row is expanded if
                    //   - the "are all rows expanded" toggle is on and the row is not in the toggled set
                    //   - the "are all rows expanded" toggle is off and the row is in the toggled set
                    const isExpanded = areAllRowsExpanded
                        ? !invertedExpansionRowSet.has(id)
                        : invertedExpansionRowSet.has(id);
                    return (
                        <Tbody key={id} isExpanded={isExpanded}>
                            <Tr>
                                <Td
                                    expand={{
                                        rowIndex,
                                        isExpanded,
                                        onToggle: () => invertedExpansionRowSet.toggle(id),
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
