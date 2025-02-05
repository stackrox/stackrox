import React, { useCallback } from 'react';
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
import { UseURLPaginationResult } from 'hooks/useURLPagination';
import { UseUrlSearchReturn } from 'hooks/useURLSearch';
import { getExternalNetworkFlows } from 'services/NetworkService';
import { getTableUIState } from 'utils/getTableUIState';
import { ExternalNetworkFlowsResponse } from 'types/networkFlow.proto';

import { getDeploymentInfoForExternalEntity, protocolLabel } from '../utils/flowUtils';
import { NetworkScopeHierarchy } from '../types/networkScopeHierarchy';

export type EntityDetailsTableProps = {
    entityId: string;
    scopeHierarchy: NetworkScopeHierarchy;
    urlPagination: UseURLPaginationResult;
    urlSearchFiltering: UseUrlSearchReturn;
};

function EntityDetailsTable({
    entityId,
    scopeHierarchy,
    urlPagination,
    urlSearchFiltering,
}: EntityDetailsTableProps) {
    const { page, perPage, setPage, setPerPage } = urlPagination;
    const { searchFilter } = urlSearchFiltering;
    const clusterId = scopeHierarchy.cluster.id;
    const { deployments, namespaces } = scopeHierarchy;
    const fetchExternalNetworkFlows = useCallback((): Promise<ExternalNetworkFlowsResponse> => {
        return getExternalNetworkFlows(clusterId, entityId, namespaces, deployments, {
            sortOption: {},
            page,
            perPage,
            advancedFilters: searchFilter,
        });
    }, [page, perPage, clusterId, deployments, entityId, namespaces, searchFilter]);

    const {
        data: externalNetworkFlows,
        isLoading,
        error,
    } = useRestQuery(fetchExternalNetworkFlows);

    const tableState = getTableUIState({
        isLoading,
        data: externalNetworkFlows?.flows,
        error,
        searchFilter,
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
                                    const deploymentInfo = getDeploymentInfoForExternalEntity(
                                        flow.props
                                    );

                                    if (!deploymentInfo) {
                                        return null;
                                    }

                                    const { l4protocol, dstPort } = flow.props;
                                    const { deployment, direction } = deploymentInfo;

                                    return (
                                        <Tr key={`${deployment.name}-${dstPort}-${l4protocol}`}>
                                            <Td dataLabel="Entity">
                                                {deployment.name}
                                                <div>
                                                    <Text
                                                        component="small"
                                                        className="pf-v5-u-color-200 pf-v5-u-text-truncate"
                                                    >
                                                        in &quot;
                                                        {deployment.namespace}
                                                        &quot;
                                                    </Text>
                                                </div>
                                            </Td>
                                            <Td dataLabel="Direction">{direction}</Td>
                                            <Td dataLabel="Port/protocol">
                                                {dstPort} / {protocolLabel[l4protocol]}
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
