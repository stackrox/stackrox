import React, { useState } from 'react';
import {
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

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { NetworkBaselinePeerStatus } from 'types/networkBaseline.proto';

import FlowsTableHeaderText from '../common/FlowsTableHeaderText';
import { FlowBulkDropdown } from '../components/FlowBulkDropdown';
import { FlowTable } from '../components/FlowTable';
import { useNetworkBaselineStatus } from '../hooks/useNetworkBaselineStatus';
import { getFlowKey } from '../utils/flowUtils';

type ExternalFlowsProps = {
    deploymentId: string;
};

function ExternalFlows({ deploymentId }: ExternalFlowsProps) {
    const anomalous = useNetworkBaselineStatus(deploymentId, 'ANOMALOUS');
    const baseline = useNetworkBaselineStatus(deploymentId, 'BASELINE');

    const [selectedAnomalous, setSelectedAnomalous] = useState<NetworkBaselinePeerStatus[]>([]);
    const [selectedBaseline, setSelectedBaseline] = useState<NetworkBaselinePeerStatus[]>([]);

    const [isAnomalousBulkActionOpen, setIsAnomalousBulkActionOpen] = useState(false);
    const [isBaselineBulkActionOpen, setIsBaselineBulkActionOpen] = useState(false);

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
    function markSelectedAsAnomalous() {}
    function addSelectedToBaseline() {}

    return (
        <Stack>
            <StackItem>
                <Toolbar className="pf-v5-u-p-0">
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
                                pagination={anomalous.pagination}
                                flowCount={totalAnomalous}
                                emptyStateMessage="No anomalous flows."
                                tableState={anomalous.tableState}
                                selectedPageAll={areAllPageAnomalousSelected}
                                onSelectAll={selectAllAnomalousFlows}
                                isFlowSelected={isFlowSelected}
                                onRowSelect={onSelectFlow}
                                rowActions={[
                                    {
                                        title: <span>Add to baseline</span>,
                                        onClick: (event) => {
                                            event.preventDefault();
                                            // handle action
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
                                pagination={baseline.pagination}
                                flowCount={totalBaseline}
                                emptyStateMessage="No baseline flows."
                                tableState={baseline.tableState}
                                selectedPageAll={areAllPageBaselineSelected}
                                onSelectAll={selectAllBaselineFlows}
                                isFlowSelected={isFlowSelected}
                                onRowSelect={onSelectFlow}
                                rowActions={[
                                    {
                                        title: <span>Mark as anomalous</span>,
                                        onClick: (event) => {
                                            event.preventDefault();
                                            // handle action
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
