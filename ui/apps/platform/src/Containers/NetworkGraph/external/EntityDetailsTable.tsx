import React, { useCallback, useState } from 'react';
import { Pagination, Text, Toolbar, ToolbarContent, ToolbarItem } from '@patternfly/react-core';
import {
    ActionsColumn,
    InnerScrollContainer,
    Table,
    Tbody,
    Td,
    Th,
    Thead,
    Tr,
} from '@patternfly/react-table';

import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import useRestQuery from 'hooks/useRestQuery';
import { getExternalNetworkFlows } from 'services/NetworkService';
import { getTableUIState } from 'utils/getTableUIState';
import { ExternalNetworkFlowsResponse } from 'types/networkFlow.proto';

import { protocolLabel } from '../utils/flowUtils';
import { NetworkScopeHierarchy } from '../types/networkScopeHierarchy';

export type EntityDetailsTableProps = {
    entityId: string;
    scopeHierarchy: NetworkScopeHierarchy;
};

function EntityDetailsTable({ entityId, scopeHierarchy }: EntityDetailsTableProps) {
    const [page, setPage] = useState(1);
    const [perPage, setPerPage] = useState(10);
    const clusterId = scopeHierarchy.cluster.id;
    const { deployments, namespaces } = scopeHierarchy;
    const fetchExternalNetworkFlows = useCallback((): Promise<ExternalNetworkFlowsResponse> => {
        return getExternalNetworkFlows(clusterId, entityId, namespaces, deployments, {
            sortOption: {},
            page,
            perPage,
            advancedFilters: {},
        });
    }, [page, perPage, clusterId, deployments, entityId, namespaces]);

    const {
        data: externalNetworkFlows,
        isLoading,
        error,
    } = useRestQuery(fetchExternalNetworkFlows);

    const tableState = getTableUIState({
        isLoading,
        data: externalNetworkFlows?.flows,
        error,
        searchFilter: {},
    });

    return (
        <>
            <Toolbar>
                <ToolbarContent>
                    <ToolbarItem variant="pagination" align={{ default: 'alignRight' }}>
                        <Pagination
                            itemCount={externalNetworkFlows?.totalFlows ?? 0}
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
                <Table variant="compact">
                    <Thead>
                        <Tr>
                            <Th>Entity</Th>
                            <Th>Direction</Th>
                            <Th>Port/protocol</Th>
                            <Th>
                                <span className="pf-v5-screen-reader">Row actions</span>
                            </Th>
                        </Tr>
                    </Thead>
                    <TbodyUnified
                        tableState={tableState}
                        colSpan={7}
                        errorProps={{
                            title: 'There was an error loading connected entities',
                        }}
                        renderer={({ data }) => (
                            <Tbody>
                                {data.map((flow) => {
                                    const { srcEntity, l4protocol, dstPort } = flow.props;
                                    return (
                                        <Tr key={`${srcEntity.id}-${dstPort}-${l4protocol}`}>
                                            <Td dataLabel="Entity">
                                                {srcEntity.deployment.name}
                                                <div>
                                                    <Text
                                                        component="small"
                                                        className="pf-v5-u-color-200 pf-v5-u-text-truncate"
                                                    >
                                                        in &quot;{srcEntity.deployment.namespace}
                                                        &quot;
                                                    </Text>
                                                </div>
                                            </Td>
                                            <Td dataLabel="Direction">-</Td>
                                            <Td dataLabel="Port/protocol">
                                                {dstPort}/{protocolLabel[l4protocol]}
                                            </Td>
                                            <Td isActionCell>
                                                <ActionsColumn
                                                    items={[
                                                        {
                                                            title: 'Add to baseline',
                                                            onClick: () => {},
                                                        },
                                                    ]}
                                                />
                                            </Td>
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

export default EntityDetailsTable;
