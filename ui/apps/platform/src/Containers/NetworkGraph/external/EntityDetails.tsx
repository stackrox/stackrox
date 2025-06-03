import React, { useCallback } from 'react';
import {
    Breadcrumb,
    BreadcrumbItem,
    Button,
    Divider,
    Flex,
    FlexItem,
    Pagination,
    Stack,
    StackItem,
    Text,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';
import { InnerScrollContainer, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import { TimeWindow } from 'constants/timeWindows';
import useRestQuery from 'hooks/useRestQuery';
import { UseURLPaginationResult } from 'hooks/useURLPagination';
import { UseUrlSearchReturn } from 'hooks/useURLSearch';
import { getExternalNetworkFlows } from 'services/NetworkService';
import { getTableUIState } from 'utils/getTableUIState';
import { ExternalNetworkFlowsResponse } from 'types/networkFlow.proto';

import { ExternalEntitiesIcon } from '../common/NetworkGraphIcons';
import { NetworkScopeHierarchy } from '../types/networkScopeHierarchy';
import { getDeploymentInfoForExternalEntity, protocolLabel } from '../utils/flowUtils';
import { timeWindowToISO } from '../utils/timeWindow';

export type EntityDetailsProps = {
    labelledById: string;
    entityName: string;
    entityId: string;
    scopeHierarchy: NetworkScopeHierarchy;
    timeWindow: TimeWindow;
    urlPagination: UseURLPaginationResult;
    urlSearchFiltering: UseUrlSearchReturn;
    onNodeSelect: (id: string) => void;
    onExternalIPSelect: (externalIP: string | undefined) => void;
};

function EntityTitleText({ text, id }: { text: string | undefined; id: string }) {
    return (
        <Title headingLevel="h2" id={id}>
            {text}
        </Title>
    );
}

function EntityDetails({
    labelledById,
    entityName,
    entityId,
    scopeHierarchy,
    timeWindow,
    urlPagination,
    urlSearchFiltering,
    onNodeSelect,
    onExternalIPSelect,
}: EntityDetailsProps) {
    const { page, perPage, setPage, setPerPage } = urlPagination;
    const { searchFilter } = urlSearchFiltering;
    const clusterId = scopeHierarchy.cluster.id;
    const { deployments, namespaces } = scopeHierarchy;
    const fetchExternalNetworkFlows = useCallback((): Promise<ExternalNetworkFlowsResponse> => {
        const fromTimestamp = timeWindowToISO(timeWindow);
        return getExternalNetworkFlows(
            clusterId,
            entityId,
            namespaces,
            deployments,
            fromTimestamp,
            {
                sortOption: {},
                page,
                perPage,
                advancedFilters: searchFilter,
            }
        );
    }, [page, perPage, clusterId, deployments, entityId, namespaces, searchFilter, timeWindow]);

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

    const externalIPName = externalNetworkFlows?.entity.externalSource.name || '';

    return (
        <Stack>
            <StackItem>
                <Flex direction={{ default: 'row' }} className="pf-v5-u-p-md pf-v5-u-mb-0">
                    <FlexItem>
                        <ExternalEntitiesIcon />
                    </FlexItem>
                    <FlexItem>
                        <Breadcrumb>
                            <BreadcrumbItem to="#" onClick={() => onExternalIPSelect(undefined)}>
                                <EntityTitleText text={entityName} id={labelledById} />
                            </BreadcrumbItem>
                            <BreadcrumbItem isActive>
                                <EntityTitleText text={externalIPName} id={externalIPName} />
                            </BreadcrumbItem>
                        </Breadcrumb>
                        <Text className="pf-v5-u-font-size-sm pf-v5-u-color-200">
                            Connected entities outside your cluster
                        </Text>
                    </FlexItem>
                </Flex>
            </StackItem>
            <Divider component="hr" />
            <StackItem isFilled style={{ overflow: 'auto' }}>
                <Stack className="pf-v5-u-p-md">
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
                                            const deploymentInfo =
                                                getDeploymentInfoForExternalEntity(flow.props);

                                            if (!deploymentInfo) {
                                                return null;
                                            }

                                            const { l4protocol, dstPort } = flow.props;
                                            const { entity, direction } = deploymentInfo;
                                            const { deployment, id } = entity;

                                            const onEntitySelect = () => {
                                                onNodeSelect(id);
                                            };

                                            return (
                                                <Tr
                                                    key={`${deployment.name}-${dstPort}-${l4protocol}`}
                                                >
                                                    <Td dataLabel="Entity">
                                                        <Button
                                                            variant="link"
                                                            isInline
                                                            onClick={onEntitySelect}
                                                        >
                                                            {deployment.name}
                                                        </Button>
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
                                                </Tr>
                                            );
                                        })}
                                    </Tbody>
                                )}
                            />
                        </Table>
                    </InnerScrollContainer>
                </Stack>
            </StackItem>
        </Stack>
    );
}

export default EntityDetails;
