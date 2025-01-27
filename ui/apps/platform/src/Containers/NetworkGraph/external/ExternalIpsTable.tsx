import React, { useCallback } from 'react';
import { Toolbar, ToolbarContent, ToolbarItem, Pagination } from '@patternfly/react-core';
import { InnerScrollContainer, Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import useMetadata from 'hooks/useMetadata';
import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import { getExternalNetworkFlowsMetadata } from 'services/NetworkService';
import { getTableUIState } from 'utils/getTableUIState';
import { getVersionedDocs } from 'utils/versioning';
import { ExternalNetworkFlowsMetadata } from 'types/networkFlow.proto';

import { NetworkScopeHierarchy } from '../types/networkScopeHierarchy';

export type ExternalIpsTableProps = {
    scopeHierarchy: NetworkScopeHierarchy;
};

type ExternalNetworkFlowsMetadataResponse = {
    data: ExternalNetworkFlowsMetadata[];
};

function ExternalIpsTable({ scopeHierarchy }: ExternalIpsTableProps) {
    const { version } = useMetadata();
    const pagination = useURLPagination(10);
    const { page, perPage, setPage, setPerPage } = pagination;
    const clusterId = scopeHierarchy.cluster.id;
    const { namespaces, deployments } = scopeHierarchy;
    const fetchExternalNetworkFlowsMetadata =
        useCallback((): Promise<ExternalNetworkFlowsMetadataResponse> => {
            return getExternalNetworkFlowsMetadata(clusterId, namespaces, deployments, {
                sortOption: {},
                page,
                perPage,
                searchFilter: {},
            });
        }, [page, perPage, clusterId, deployments, namespaces]);

    const {
        data: externalNetworkFlowsMetadata,
        isLoading,
        error,
    } = useRestQuery(fetchExternalNetworkFlowsMetadata);

    const tableState = getTableUIState({
        isLoading,
        data: externalNetworkFlowsMetadata?.data,
        error,
        searchFilter: {},
    });

    return (
        <>
            <Toolbar>
                <ToolbarContent>
                    <ToolbarItem variant="pagination" align={{ default: 'alignRight' }}>
                        <Pagination
                            toggleTemplate={({ firstIndex, lastIndex }) => (
                                <span>
                                    <b>
                                        {firstIndex} - {lastIndex}
                                    </b>{' '}
                                    of <b>many</b>
                                </span>
                            )}
                            page={page}
                            perPage={perPage}
                            onSetPage={(_, newPage) => setPage(newPage)}
                            onPerPageSelect={(_, newPerPage) => setPerPage(newPerPage)}
                            isCompact
                        />
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
            <InnerScrollContainer>
                <Table>
                    <Thead>
                        <Tr>
                            <Th>Entity</Th>
                            <Th>Internal flows</Th>
                        </Tr>
                    </Thead>
                    <TbodyUnified
                        tableState={tableState}
                        colSpan={7}
                        errorProps={{
                            title: 'There was an error loading external ips',
                        }}
                        emptyProps={{
                            message: 'No external ips found. This feature might not be enabled.',
                            children: (
                                <ExternalLink>
                                    <a
                                        href={getVersionedDocs(
                                            version,
                                            'operating/visualizing-external-entities'
                                        )}
                                        target="_blank"
                                        rel="noopener noreferrer"
                                    >
                                        Enabling external ip collection
                                    </a>
                                </ExternalLink>
                            ),
                        }}
                        renderer={({ data }) => (
                            <Tbody>
                                {data.map(({ entity, flowsCount }) => {
                                    return (
                                        <Tr key={entity.id}>
                                            <Td dataLabel="Entity">{entity.externalSource.name}</Td>
                                            <Td dataLabel="Internal flows">{flowsCount}</Td>
                                        </Tr>
                                    );
                                })}
                            </Tbody>
                        )}
                    />
                </Table>
            </InnerScrollContainer>
        </>
    );
}

export default ExternalIpsTable;
