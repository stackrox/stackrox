import React, { useEffect, useState } from 'react';
import {
    Alert,
    Divider,
    DropdownItem,
    ExpandableSection,
    ExpandableSectionToggle,
    Flex,
    Stack,
    StackItem,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';
import pluralize from 'pluralize';

import ConfirmationModal from 'Components/PatternFly/ConfirmationModal';
import SearchFilterChips from 'Components/PatternFly/SearchFilterChips';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { markNetworkBaselineStatuses } from 'services/NetworkService';
import { NetworkBaselinePeerStatus, PeerStatus } from 'types/networkBaseline.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import FlowsTableHeaderText from '../common/FlowsTableHeaderText';
import IPMatchFilter from '../common/IPMatchFilter';
import { FlowBulkDropdown } from '../components/FlowBulkDropdown';
import { FlowTable } from '../components/FlowTable';
import { useNetworkBaselineStatus } from '../hooks/useNetworkBaselineStatus';
import { EXTERNAL_SOURCE_ADDRESS_QUERY } from '../NetworkGraph.constants';
import {
    usePagination,
    usePaginationSecondary,
    useSearchFilterSidePanel,
    useTimeWindow,
} from '../NetworkGraphURLStateContext';
import { getFlowKey } from '../utils/flowUtils';

function getUniquePendingFlows(flows: NetworkBaselinePeerStatus[]) {
    const uniquePendingFlowsSet = new Set<string>();

    return flows.flatMap((flow) => {
        const {
            peer: { ingress, port, protocol },
        } = flow;

        const key = `${ingress}-${port}-${protocol}`;

        if (uniquePendingFlowsSet.has(key)) {
            return [];
        }

        uniquePendingFlowsSet.add(key);

        return {
            direction: ingress ? 'Ingress' : 'Egress',
            port,
            protocol: protocol === 'L4_PROTOCOL_TCP' ? 'TCP' : 'UDP',
            key,
        };
    });
}

type ExternalFlowsProps = {
    deploymentId: string;
};

type PendingStatusChange = {
    error: string | null;
    flows: NetworkBaselinePeerStatus[];
    isSubmitting: boolean;
    targetStatus: PeerStatus;
    uniqueFlows: ReturnType<typeof getUniquePendingFlows>;
};

function ExternalFlows({ deploymentId }: ExternalFlowsProps) {
    const { searchFilter, setSearchFilter } = useSearchFilterSidePanel();
    const { timeWindow } = useTimeWindow();

    const anomalousPagination = usePagination();
    const baselinePagination = usePaginationSecondary();

    const { setPage: setPageAnomalous } = anomalousPagination;
    const { setPage: setPageBaseline } = baselinePagination;

    const anomalous = useNetworkBaselineStatus(
        deploymentId,
        timeWindow,
        anomalousPagination,
        searchFilter,
        'ANOMALOUS'
    );
    const baseline = useNetworkBaselineStatus(
        deploymentId,
        timeWindow,
        baselinePagination,
        searchFilter,
        'BASELINE'
    );

    const [selectedAnomalous, setSelectedAnomalous] = useState<NetworkBaselinePeerStatus[]>([]);
    const [selectedBaseline, setSelectedBaseline] = useState<NetworkBaselinePeerStatus[]>([]);

    const [isAnomalousBulkActionOpen, setIsAnomalousBulkActionOpen] = useState(false);
    const [isBaselineBulkActionOpen, setIsBaselineBulkActionOpen] = useState(false);

    const [pendingStatusChange, setPendingStatusChange] = useState<PendingStatusChange | null>(
        null
    );

    useEffect(() => {
        setPageAnomalous(1);
        setPageBaseline(1);
    }, [setPageAnomalous, setPageBaseline, searchFilter]);

    const { isOpen: isAnomalousFlowsExpanded, onToggle: toggleAnomalousFlowsExpandable } =
        useSelectToggle(true);
    const { isOpen: isBaselineFlowsExpanded, onToggle: toggleBaselineFlowsExpandable } =
        useSelectToggle(true);

    function setFlowSelected(flow: NetworkBaselinePeerStatus, isSelecting = true) {
        const key = getFlowKey(flow);
        const setter = flow.status === 'ANOMALOUS' ? setSelectedAnomalous : setSelectedBaseline;

        setter((prev) => {
            const without = prev.filter((f) => getFlowKey(f) !== key);
            return isSelecting ? [...without, flow] : without;
        });
    }

    const totalAnomalous = anomalous.total;
    const totalBaseline = baseline.total;

    const anomalousFlows = anomalous.flows;
    const baselineFlows = baseline.flows;

    const areAllPageAnomalousSelected =
        anomalousFlows.length > 0 && anomalousFlows.every(isFlowSelected);
    const areAllPageBaselineSelected =
        baselineFlows.length > 0 && baselineFlows.every(isFlowSelected);

    function selectAllBaselineFlows(isSelecting = true) {
        togglePageFlows(baselineFlows, isSelecting);
    }

    function selectAllAnomalousFlows(isSelecting = true) {
        togglePageFlows(anomalousFlows, isSelecting);
    }

    function isFlowSelected(flow: NetworkBaselinePeerStatus) {
        const key = getFlowKey(flow);
        return (flow.status === 'ANOMALOUS' ? selectedAnomalous : selectedBaseline).some(
            (f) => getFlowKey(f) === key
        );
    }

    function onSelectFlow(
        flow: NetworkBaselinePeerStatus,
        _rowIndex: number,
        isSelecting: boolean
    ) {
        setFlowSelected(flow, isSelecting);
    }

    function togglePageFlows(flows: NetworkBaselinePeerStatus[], isSelecting = true) {
        if (!flows.length) {
            return;
        }

        const setter = flows[0].status === 'ANOMALOUS' ? setSelectedAnomalous : setSelectedBaseline;

        setter((prev) => {
            const pageKeys = new Set(flows.map(getFlowKey));
            const withoutPage = prev.filter((f) => !pageKeys.has(getFlowKey(f)));
            return isSelecting ? [...withoutPage, ...flows] : withoutPage;
        });
    }

    async function updateFlowsStatus(
        flows: NetworkBaselinePeerStatus | NetworkBaselinePeerStatus[],
        targetStatus: PeerStatus
    ) {
        const selectedFlows = Array.isArray(flows) ? flows : [flows];
        if (!selectedFlows.length) {
            return;
        }

        const payload = selectedFlows.map((flow) => ({ ...flow, status: targetStatus }));

        await markNetworkBaselineStatuses({ deploymentId, networkBaselines: payload });
        await Promise.all([anomalous.refetch(), baseline.refetch()]);
        setSelectedAnomalous([]);
        setSelectedBaseline([]);
    }

    function confirmStatusChange(flows: NetworkBaselinePeerStatus[], targetStatus: PeerStatus) {
        setPendingStatusChange({
            error: null,
            flows,
            isSubmitting: false,
            targetStatus,
            uniqueFlows: getUniquePendingFlows(flows),
        });
    }

    async function onConfirmStatusChange() {
        if (!pendingStatusChange) {
            return;
        }

        setPendingStatusChange((prev) => prev && { ...prev, isSubmitting: true, error: null });

        try {
            await updateFlowsStatus(pendingStatusChange.flows, pendingStatusChange.targetStatus);
            setPendingStatusChange(null);
        } catch (err) {
            setPendingStatusChange(
                (prev) => prev && { ...prev, isSubmitting: false, error: getAxiosErrorMessage(err) }
            );
        }
    }

    function onCancelStatusChange() {
        setPendingStatusChange(null);
    }

    return (
        <>
            <Stack>
                <StackItem>
                    <Toolbar className="pf-v5-u-pb-md pf-v5-u-pt-0">
                        <ToolbarContent className="pf-v5-u-px-0">
                            <ToolbarItem className="pf-v5-u-w-100 pf-v5-u-mr-0">
                                <IPMatchFilter
                                    searchFilter={searchFilter}
                                    setSearchFilter={setSearchFilter}
                                />
                            </ToolbarItem>
                            <ToolbarItem className="pf-v5-u-w-100">
                                <SearchFilterChips
                                    searchFilter={searchFilter}
                                    onFilterChange={setSearchFilter}
                                    filterChipGroupDescriptors={[
                                        {
                                            displayName: 'CIDR',
                                            searchFilterName: EXTERNAL_SOURCE_ADDRESS_QUERY,
                                        },
                                    ]}
                                />
                            </ToolbarItem>
                        </ToolbarContent>
                    </Toolbar>
                </StackItem>
                <Divider />
                <StackItem>
                    <Toolbar className="pf-v5-u-pt-md">
                        <ToolbarContent className="pf-v5-u-px-0">
                            <ToolbarItem>
                                <FlowsTableHeaderText
                                    type={'total'}
                                    numFlows={anomalous.total + baseline.total}
                                />
                            </ToolbarItem>
                        </ToolbarContent>
                    </Toolbar>
                </StackItem>
                <StackItem>
                    <Stack hasGutter>
                        <StackItem>
                            <Flex justifyContent={{ default: 'justifyContentSpaceBetween' }}>
                                <ExpandableSectionToggle
                                    isExpanded={isAnomalousFlowsExpanded}
                                    onToggle={(isExpanded) =>
                                        toggleAnomalousFlowsExpandable(isExpanded)
                                    }
                                    toggleId={'anomalous-expandable-toggle'}
                                    contentId={'anomalous-expandable-content'}
                                >
                                    {`${totalAnomalous} anomalous ${pluralize('flow', totalAnomalous)}`}
                                </ExpandableSectionToggle>
                                <FlowBulkDropdown
                                    selectedCount={selectedAnomalous.length}
                                    isOpen={isAnomalousBulkActionOpen}
                                    setOpen={setIsAnomalousBulkActionOpen}
                                    onClear={() => setSelectedAnomalous([])}
                                >
                                    <DropdownItem
                                        onClick={() =>
                                            confirmStatusChange(selectedAnomalous, 'BASELINE')
                                        }
                                    >
                                        Add to baseline
                                    </DropdownItem>
                                </FlowBulkDropdown>
                            </Flex>
                            <ExpandableSection
                                isExpanded={isAnomalousFlowsExpanded}
                                isDetached
                                toggleId={'anomalous-expandable-toggle'}
                                contentId={'anomalous-expandable-content'}
                            >
                                <FlowTable
                                    pagination={anomalous.urlPagination}
                                    flowCount={totalAnomalous}
                                    statusType="ANOMALOUS"
                                    tableState={anomalous.tableState}
                                    areAllRowsSelected={areAllPageAnomalousSelected}
                                    onSelectAll={selectAllAnomalousFlows}
                                    isFlowSelected={isFlowSelected}
                                    onRowSelect={onSelectFlow}
                                    rowActions={(flow) => [
                                        {
                                            title: <span>Add to baseline</span>,
                                            onClick: async (e) => {
                                                e.preventDefault();
                                                confirmStatusChange([flow], 'BASELINE');
                                            },
                                        },
                                    ]}
                                />
                            </ExpandableSection>
                        </StackItem>
                        <StackItem>
                            <Flex justifyContent={{ default: 'justifyContentSpaceBetween' }}>
                                <ExpandableSectionToggle
                                    isExpanded={isBaselineFlowsExpanded}
                                    onToggle={(isExpanded) =>
                                        toggleBaselineFlowsExpandable(isExpanded)
                                    }
                                    toggleId={'baseline-expandable-toggle'}
                                    contentId={'baseline-expandable-content'}
                                >
                                    {`${totalBaseline} baseline ${pluralize('flow', totalBaseline)}`}
                                </ExpandableSectionToggle>
                                <FlowBulkDropdown
                                    selectedCount={selectedBaseline.length}
                                    isOpen={isBaselineBulkActionOpen}
                                    setOpen={setIsBaselineBulkActionOpen}
                                    onClear={() => setSelectedBaseline([])}
                                >
                                    <DropdownItem
                                        onClick={() =>
                                            confirmStatusChange(selectedBaseline, 'ANOMALOUS')
                                        }
                                    >
                                        Mark as anomalous
                                    </DropdownItem>
                                </FlowBulkDropdown>
                            </Flex>
                            <ExpandableSection
                                isDetached
                                toggleId={'baseline-expandable-toggle'}
                                contentId={'baseline-expandable-content'}
                                isExpanded={isBaselineFlowsExpanded}
                            >
                                <FlowTable
                                    pagination={baseline.urlPagination}
                                    flowCount={totalBaseline}
                                    statusType="BASELINE"
                                    tableState={baseline.tableState}
                                    areAllRowsSelected={areAllPageBaselineSelected}
                                    onSelectAll={selectAllBaselineFlows}
                                    isFlowSelected={isFlowSelected}
                                    onRowSelect={onSelectFlow}
                                    rowActions={(flow) => [
                                        {
                                            title: <span>Mark as anomalous</span>,
                                            onClick: async (e) => {
                                                e.preventDefault();
                                                confirmStatusChange([flow], 'ANOMALOUS');
                                            },
                                        },
                                    ]}
                                />
                            </ExpandableSection>
                        </StackItem>
                    </Stack>
                </StackItem>
            </Stack>
            {pendingStatusChange && (
                <ConfirmationModal
                    title="Apply status change to multiple flows?"
                    ariaLabel="apply status change"
                    confirmText="Apply"
                    isLoading={pendingStatusChange.isSubmitting}
                    isOpen
                    onConfirm={onConfirmStatusChange}
                    onCancel={onCancelStatusChange}
                    isDestructive={false}
                >
                    {pendingStatusChange.error && (
                        <Alert
                            className="pf-v5-u-mb-sm"
                            component="p"
                            isInline
                            title={pendingStatusChange.error}
                            variant="danger"
                        />
                    )}
                    <p>
                        All flows that have the same combination of direction, port, and protocol
                        have the same status. This action will affect the status of all matching
                        flows, even flows that you did not select.
                    </p>
                    <div style={{ maxHeight: '300px', overflowY: 'auto' }}>
                        <Table variant="compact" borders={false} isStickyHeader>
                            <Thead>
                                <Tr>
                                    <Th>Direction</Th>
                                    <Th>Port / protocol</Th>
                                </Tr>
                            </Thead>
                            <Tbody>
                                {pendingStatusChange.uniqueFlows.map(
                                    ({ direction, key, port, protocol }) => (
                                        <Tr key={key}>
                                            <Td dataLabel="Direction">{direction}</Td>
                                            <Td dataLabel="Port / protocol">
                                                {port} / {protocol}
                                            </Td>
                                        </Tr>
                                    )
                                )}
                            </Tbody>
                        </Table>
                    </div>
                </ConfirmationModal>
            )}
        </>
    );
}

export default ExternalFlows;
