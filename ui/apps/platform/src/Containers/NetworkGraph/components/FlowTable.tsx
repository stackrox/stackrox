import { ToolbarContent, ToolbarItem, Pagination } from '@patternfly/react-core';
import { Table, Thead, Tbody, Tr, Th, Td, ActionsColumn } from '@patternfly/react-table';
import type { IAction } from '@patternfly/react-table';

import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import type { UseURLPaginationResult } from 'hooks/useURLPagination';
import type { NetworkBaselinePeerStatus } from 'types/networkBaseline.proto';
import type { TableUIState } from 'utils/getTableUIState';

import type { BaselineStatusType } from '../types/flow.type';
import { getFlowKey } from '../utils/flowUtils';

import { useSearchFilterSidePanel } from '../NetworkGraphURLStateContext';

type FlowTableProps = {
    pagination: UseURLPaginationResult;
    flowCount: number;
    statusType: BaselineStatusType;
    tableState: TableUIState<NetworkBaselinePeerStatus>;
    areAllRowsSelected: boolean;
    onSelectAll: (sel: boolean) => void;
    rowActions: (flow: NetworkBaselinePeerStatus) => IAction[];
    isFlowSelected: (flow: NetworkBaselinePeerStatus) => boolean;
    onRowSelect: (flow: NetworkBaselinePeerStatus, rowIndex: number, select: boolean) => void;
};

export function FlowTable({
    pagination,
    flowCount,
    statusType,
    tableState,
    areAllRowsSelected,
    onSelectAll,
    rowActions,
    isFlowSelected,
    onRowSelect,
}: FlowTableProps) {
    const { page, perPage, setPage, setPerPage } = pagination;
    const { setSearchFilter } = useSearchFilterSidePanel();
    return (
        <>
            <ToolbarContent>
                <ToolbarItem variant="pagination" align={{ default: 'alignRight' }}>
                    <Pagination
                        itemCount={flowCount}
                        page={page}
                        perPage={perPage}
                        onSetPage={(_, newPage) => setPage(newPage)}
                        onPerPageSelect={(_, newPerPage) => setPerPage(newPerPage)}
                        isCompact
                    />
                </ToolbarItem>
            </ToolbarContent>
            <Table variant="compact">
                <Thead>
                    <Tr>
                        <Th
                            select={{
                                isSelected: areAllRowsSelected,
                                onSelect: (_e, s) => onSelectAll(s),
                            }}
                        />
                        <Th>Entity</Th>
                        <Th>Direction</Th>
                        <Th>Port / protocol</Th>
                        <Th>
                            <span className="pf-v5-screen-reader">Row actions</span>
                        </Th>
                    </Tr>
                </Thead>

                <TbodyUnified
                    tableState={tableState}
                    colSpan={5}
                    emptyProps={{ title: `No ${statusType.toLowerCase()} flows`, message: '' }}
                    filteredEmptyProps={{
                        title: `No ${statusType.toLowerCase()} flows found`,
                        onClearFilters: () => {
                            setSearchFilter({});
                        },
                    }}
                    renderer={({ data }) => (
                        <Tbody>
                            {data.map((flow, idx) => (
                                <Tr key={getFlowKey(flow)}>
                                    <Td
                                        select={{
                                            rowIndex: idx,
                                            isSelected: isFlowSelected(flow),
                                            onSelect: (_e, isSelecting) =>
                                                onRowSelect(flow, idx, isSelecting),
                                        }}
                                    />
                                    <Td>{flow.peer.entity.name}</Td>
                                    <Td>{flow.peer.ingress ? 'Ingress' : 'Egress'}</Td>
                                    <Td>{`${flow.peer.port} / ${
                                        flow.peer.protocol === 'L4_PROTOCOL_TCP' ? 'TCP' : 'UDP'
                                    }`}</Td>
                                    <Td isActionCell>
                                        <ActionsColumn items={rowActions(flow)} />
                                    </Td>
                                </Tr>
                            ))}
                        </Tbody>
                    )}
                />
            </Table>
        </>
    );
}
