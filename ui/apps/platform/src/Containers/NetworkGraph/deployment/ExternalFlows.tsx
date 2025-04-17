import React, { useCallback } from 'react';
import {
    Stack,
    StackItem,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
    ExpandableSection,
    Pagination,
} from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import useRestQuery from 'hooks/useRestQuery';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { getNetworkBaselineExternalStatus } from 'services/NetworkService';
import { NetworkBaselineExternalStatusResponse } from 'types/networkBaseline.proto';
import { getTableUIState } from 'utils/getTableUIState';

import pluralize from 'pluralize';
import useURLPagination from 'hooks/useURLPagination';
import FlowsTableHeaderText from '../common/FlowsTableHeaderText';

type ExternalFlowsProps = {
    deploymentId: string;
};

function ExternalFlows({ deploymentId }: ExternalFlowsProps) {
    const { isOpen: isAnomalousFlowsExpanded, onToggle: toggleAnomalousFlowsExpandable } =
        useSelectToggle(true);
    const { isOpen: isBaselineFlowsExpanded, onToggle: toggleBaselineFlowsExpandable } =
        useSelectToggle(true);

    const {
        page: anomalousPage,
        perPage: anomalousPerPage,
        setPage: anomalousSetPage,
        setPerPage: anomalousSetPerPage,
    } = useURLPagination(10, 'anomalous');

    const {
        page: baselinePage,
        perPage: baselinePerPage,
        setPage: baselineSetPage,
        setPerPage: baselineSetPerPage,
    } = useURLPagination(10, 'baseline');

    const fetchExternalFlowsAnomalous = useCallback(
        (): Promise<NetworkBaselineExternalStatusResponse> =>
            getNetworkBaselineExternalStatus(deploymentId, {
                sortOption: {},
                page: anomalousPage,
                perPage: anomalousPerPage,
                searchFilter: {},
            }),
        [anomalousPage, anomalousPerPage, deploymentId]
    );
    const {
        data: responseAnomalous,
        isLoading: isLoadingAnomalous,
        error: anomalousError,
    } = useRestQuery(fetchExternalFlowsAnomalous);

    const fetchExternalFlowsBaseline = useCallback(
        (): Promise<NetworkBaselineExternalStatusResponse> =>
            getNetworkBaselineExternalStatus(deploymentId, {
                sortOption: {},
                page: baselinePage,
                perPage: baselinePerPage,
                searchFilter: {},
            }),
        [baselinePage, baselinePerPage, deploymentId]
    );
    const {
        data: responseBaseline,
        isLoading: isLoadingBaseline,
        error: baselineError,
    } = useRestQuery(fetchExternalFlowsBaseline);

    const anomalousTableState = getTableUIState({
        isLoading: isLoadingAnomalous,
        data: responseAnomalous?.anomalous,
        error: anomalousError,
        searchFilter: {},
    });

    const baselineTableState = getTableUIState({
        isLoading: isLoadingBaseline,
        data: responseBaseline?.baseline,
        error: baselineError,
        searchFilter: {},
    });

    const totalAnomalous = responseAnomalous?.totalAnomalous ?? 0;
    const totalBaseline = responseBaseline?.totalBaseline ?? 0;

    const totalFlows = totalAnomalous + totalBaseline;

    return (
        <Stack>
            <StackItem>
                <Toolbar className="pf-v5-u-p-0">
                    <ToolbarContent className="pf-v5-u-px-0">
                        <ToolbarItem>
                            <FlowsTableHeaderText type={'total'} numFlows={totalFlows} />
                        </ToolbarItem>
                    </ToolbarContent>
                </Toolbar>
            </StackItem>
            <StackItem>
                <Stack hasGutter>
                    <StackItem>
                        <ExpandableSection
                            toggleText={`Anomalous ${pluralize('flow', totalAnomalous)}`}
                            onToggle={(e, isExpanded) => toggleAnomalousFlowsExpandable(isExpanded)}
                            isExpanded={isAnomalousFlowsExpanded}
                        >
                            <ToolbarContent>
                                <ToolbarItem variant="pagination" align={{ default: 'alignRight' }}>
                                    <Pagination
                                        itemCount={totalAnomalous}
                                        page={anomalousPage}
                                        perPage={anomalousPerPage}
                                        onSetPage={(_, newPage) => anomalousSetPage(newPage)}
                                        onPerPageSelect={(_, newPerPage) =>
                                            anomalousSetPerPage(newPerPage)
                                        }
                                        isCompact
                                    />
                                </ToolbarItem>
                            </ToolbarContent>
                            <Table variant="compact">
                                <Thead>
                                    <Tr>
                                        <Th>Entity</Th>
                                        <Th>Direction</Th>
                                        <Th>Port / protocol</Th>
                                    </Tr>
                                </Thead>
                                <TbodyUnified
                                    tableState={anomalousTableState}
                                    colSpan={3}
                                    errorProps={{
                                        title: 'There was an error',
                                    }}
                                    emptyProps={{
                                        message: 'No anomalous flows.',
                                    }}
                                    renderer={({ data }) => (
                                        <Tbody>
                                            {data.map((flow) => {
                                                return (
                                                    <Tr key={flow.peer.entity.id}>
                                                        <Td dataLabel="Entity">
                                                            {flow.peer.entity.name}
                                                        </Td>
                                                        <Td dataLabel="Direction">
                                                            {flow.peer.ingress
                                                                ? `Ingress`
                                                                : `Egress`}
                                                        </Td>
                                                        <Td dataLabel="Port / protocol">
                                                            {`${flow.peer.port} / ${flow.peer.protocol === 'L4_PROTOCOL_TCP' ? 'TCP' : 'UDP'}`}
                                                        </Td>
                                                    </Tr>
                                                );
                                            })}
                                        </Tbody>
                                    )}
                                />
                            </Table>
                        </ExpandableSection>
                    </StackItem>
                    <StackItem>
                        <ExpandableSection
                            toggleText={`Baseline ${pluralize('flow', totalBaseline)}`}
                            onToggle={(e, isExpanded) => toggleBaselineFlowsExpandable(isExpanded)}
                            isExpanded={isBaselineFlowsExpanded}
                        >
                            <ToolbarContent>
                                <ToolbarItem variant="pagination" align={{ default: 'alignRight' }}>
                                    <Pagination
                                        itemCount={totalBaseline}
                                        page={baselinePage}
                                        perPage={baselinePerPage}
                                        onSetPage={(_, newPage) => baselineSetPage(newPage)}
                                        onPerPageSelect={(_, newPerPage) =>
                                            baselineSetPerPage(newPerPage)
                                        }
                                        isCompact
                                    />
                                </ToolbarItem>
                            </ToolbarContent>
                            <Table variant="compact">
                                <Thead>
                                    <Tr>
                                        <Th>Entity</Th>
                                        <Th>Direction</Th>
                                        <Th>Port / protocol</Th>
                                    </Tr>
                                </Thead>
                                <TbodyUnified
                                    tableState={baselineTableState}
                                    colSpan={3}
                                    errorProps={{
                                        title: 'There was an error',
                                    }}
                                    emptyProps={{
                                        message: 'No baseline flows.',
                                    }}
                                    renderer={({ data }) => (
                                        <Tbody>
                                            {data.map((flow) => {
                                                return (
                                                    <Tr key={flow.peer.entity.id}>
                                                        <Td dataLabel="Entity">
                                                            {flow.peer.entity.name}
                                                        </Td>
                                                        <Td dataLabel="Direction">
                                                            {flow.peer.ingress
                                                                ? `Ingress`
                                                                : `Egress`}
                                                        </Td>
                                                        <Td dataLabel="Port / protocol">
                                                            {`${flow.peer.port} / ${flow.peer.protocol === 'L4_PROTOCOL_TCP' ? 'TCP' : 'UDP'}`}
                                                        </Td>
                                                    </Tr>
                                                );
                                            })}
                                        </Tbody>
                                    )}
                                />
                            </Table>
                        </ExpandableSection>
                    </StackItem>
                </Stack>
            </StackItem>
        </Stack>
    );
}

export default ExternalFlows;
