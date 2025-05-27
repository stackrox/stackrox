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
import pluralize from 'pluralize';

import SearchFilterChips from 'Components/PatternFly/SearchFilterChips';
import { TimeWindow } from 'constants/timeWindows';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { UseURLPaginationResult } from 'hooks/useURLPagination';
import { UseUrlSearchReturn } from 'hooks/useURLSearch';
import { markNetworkBaselineStatuses } from 'services/NetworkService';
import { NetworkBaselinePeerStatus, PeerStatus } from 'types/networkBaseline.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import FlowsTableHeaderText from '../common/FlowsTableHeaderText';
import IPMatchFilter from '../common/IPMatchFilter';
import { FlowBulkDropdown } from '../components/FlowBulkDropdown';
import { FlowTable } from '../components/FlowTable';
import { useNetworkBaselineStatus } from '../hooks/useNetworkBaselineStatus';
import { EXTERNAL_SOURCE_ADDRESS_QUERY } from '../NetworkGraph.constants';
import { getFlowKey } from '../utils/flowUtils';

type ExternalFlowsProps = {
    deploymentId: string;
    timeWindow: TimeWindow;
    anomalousUrlPagination: UseURLPaginationResult;
    baselineUrlPagination: UseURLPaginationResult;
    urlSearchFiltering: UseUrlSearchReturn;
};

function ExternalFlows({
    deploymentId,
    timeWindow,
    anomalousUrlPagination,
    baselineUrlPagination,
    urlSearchFiltering,
}: ExternalFlowsProps) {
    const { searchFilter, setSearchFilter } = urlSearchFiltering;

    const anomalous = useNetworkBaselineStatus(
        deploymentId,
        timeWindow,
        anomalousUrlPagination,
        searchFilter,
        'ANOMALOUS'
    );
    const baseline = useNetworkBaselineStatus(
        deploymentId,
        timeWindow,
        baselineUrlPagination,
        searchFilter,
        'BASELINE'
    );

    const [selectedAnomalous, setSelectedAnomalous] = useState<NetworkBaselinePeerStatus[]>([]);
    const [selectedBaseline, setSelectedBaseline] = useState<NetworkBaselinePeerStatus[]>([]);

    const [isAnomalousBulkActionOpen, setIsAnomalousBulkActionOpen] = useState(false);
    const [isBaselineBulkActionOpen, setIsBaselineBulkActionOpen] = useState(false);

    const [networkFlowError, setNetworkFlowError] = useState('');

    const { setPage: setPageAnomalous } = anomalousUrlPagination;
    const { setPage: setPageBaseline } = baselineUrlPagination;

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
        const selected = Array.isArray(flows) ? flows : [flows];
        if (!selected.length) {
            return;
        }

        const payload = selected.map((f) => ({ ...f, status: targetStatus }));

        try {
            await markNetworkBaselineStatuses({ deploymentId, networkBaselines: payload });
            await Promise.all([anomalous.refetch(), baseline.refetch()]);
            setSelectedAnomalous([]);
            setSelectedBaseline([]);
            setNetworkFlowError('');
        } catch (err) {
            setNetworkFlowError(getAxiosErrorMessage(err));
        }
    }

    async function markSelectedAsAnomalous() {
        await updateFlowsStatus(selectedBaseline, 'ANOMALOUS');
    }

    async function addSelectedToBaseline() {
        await updateFlowsStatus(selectedAnomalous, 'BASELINE');
    }

    return (
        <Stack>
            {networkFlowError && (
                <StackItem>
                    <Alert
                        isInline
                        variant="danger"
                        title={networkFlowError}
                        component="p"
                        className="pf-v5-u-mb-sm"
                    />
                </StackItem>
            )}
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
                                <DropdownItem onClick={addSelectedToBaseline}>
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
                                urlSearchFiltering={urlSearchFiltering}
                                onSelectAll={selectAllAnomalousFlows}
                                isFlowSelected={isFlowSelected}
                                onRowSelect={onSelectFlow}
                                rowActions={(flow) => [
                                    {
                                        title: <span>Add to baseline</span>,
                                        onClick: async (e) => {
                                            e.preventDefault();
                                            await updateFlowsStatus(flow, 'BASELINE');
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
                                onToggle={(isExpanded) => toggleBaselineFlowsExpandable(isExpanded)}
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
                                <DropdownItem onClick={markSelectedAsAnomalous}>
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
                                urlSearchFiltering={urlSearchFiltering}
                                onSelectAll={selectAllBaselineFlows}
                                isFlowSelected={isFlowSelected}
                                onRowSelect={onSelectFlow}
                                rowActions={(flow) => [
                                    {
                                        title: <span>Mark as anomalous</span>,
                                        onClick: async (e) => {
                                            e.preventDefault();
                                            await updateFlowsStatus(flow, 'ANOMALOUS');
                                        },
                                    },
                                ]}
                            />
                        </ExpandableSection>
                    </StackItem>
                </Stack>
            </StackItem>
        </Stack>
    );
}

export default ExternalFlows;
