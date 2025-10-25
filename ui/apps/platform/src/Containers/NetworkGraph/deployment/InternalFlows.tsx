import { useState } from 'react';
import {
    Alert,
    Bullseye,
    Divider,
    EmptyState,
    ExpandableSection,
    Flex,
    FlexItem,
    Spinner,
    Stack,
    StackItem,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';
import pluralize from 'pluralize';

import usePermissions from 'hooks/usePermissions';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import useModifyBaselineStatuses from '../api/useModifyBaselineStatuses';
import type { AdvancedFlowsFilterType } from '../common/AdvancedFlowsFilter/types';
import {
    filterNetworkFlows,
    getAllUniquePorts,
    getNumExtraneousEgressFlows,
    getNumExtraneousIngressFlows,
    getNumFlows,
} from '../utils/flowUtils';
import type { CustomNodeModel } from '../types/topology.type';
import type { EdgeState } from '../components/EdgeStateSelect';
import type { Flow } from '../types/flow.type';

import AdvancedFlowsFilter, {
    defaultAdvancedFlowsFilters,
} from '../common/AdvancedFlowsFilter/AdvancedFlowsFilter';
import EntityNameSearchInput from '../common/EntityNameSearchInput';
import FlowsTable from '../common/FlowsTable';
import FlowsTableHeaderText from '../common/FlowsTableHeaderText';
import FlowsBulkActions from '../common/FlowsBulkActions';

import './DeploymentFlows.css';

type InternalFlowsProps = {
    deploymentId: string;
    nodes: CustomNodeModel[];
    edgeState: EdgeState;
    onNodeSelect: (id: string) => void;
    isLoadingNetworkFlows: boolean;
    networkFlowsError: string;
    networkFlows: Flow[];
    refetchFlows: () => void;
};

function InternalFlows({
    deploymentId,
    nodes,
    edgeState,
    onNodeSelect,
    isLoadingNetworkFlows,
    networkFlowsError,
    networkFlows,
    refetchFlows,
}: InternalFlowsProps) {
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForActions = hasReadWriteAccess('DeploymentExtension');

    // component state
    const [entityNameFilter, setEntityNameFilter] = useState<string>('');
    const [advancedFilters, setAdvancedFilters] = useState<AdvancedFlowsFilterType>(
        defaultAdvancedFlowsFilters
    );
    const { isOpen: isAnomalousFlowsExpanded, onToggle: toggleAnomalousFlowsExpandable } =
        useSelectToggle(true);
    const { isOpen: isBaselineFlowsExpanded, onToggle: toggleBaselineFlowsExpandable } =
        useSelectToggle(true);

    const {
        isModifying,
        error: modifyError,
        modifyBaselineStatuses,
    } = useModifyBaselineStatuses(deploymentId);
    const filteredFlows = filterNetworkFlows(networkFlows, entityNameFilter, advancedFilters);

    const initialExpandedRows = filteredFlows
        .filter((row) => row.children && !!row.children.length)
        .map((row) => row.id); // Default to all expanded
    const [expandedRows, setExpandedRows] = useState<string[]>(initialExpandedRows);

    const [selectedAnomalousRows, setSelectedAnomalousRows] = useState<string[]>([]);
    const [selectedBaselineRows, setSelectedBaselineRows] = useState<string[]>([]);

    // derived data
    const anomalousFlows = filteredFlows.filter((flow) => flow.isAnomalous);
    const baselineFlows = filteredFlows.filter((flow) => !flow.isAnomalous);

    const numFlows = getNumFlows(filteredFlows);
    const numAnomalousFlows = getNumFlows(anomalousFlows);
    const numBaselineFlows = getNumFlows(baselineFlows);

    const allUniquePorts = getAllUniquePorts(networkFlows);
    const numExtraneousEgressFlows = getNumExtraneousEgressFlows(nodes);
    const numExtraneousIngressFlows = getNumExtraneousIngressFlows(nodes);
    const totalFlows = numFlows + numExtraneousEgressFlows + numExtraneousIngressFlows;

    const selectedRows = [...selectedAnomalousRows, ...selectedBaselineRows];

    const onSelectFlow = (entityId: string) => {
        onNodeSelect(entityId);
    };

    function addToBaseline(flow: Flow) {
        modifyBaselineStatuses([flow], 'BASELINE', refetchFlows);
    }

    function markAsAnomalous(flow: Flow) {
        modifyBaselineStatuses([flow], 'ANOMALOUS', refetchFlows);
    }

    function addSelectedToBaseline() {
        const selectedFlows = filteredFlows.filter((networkBaseline) => {
            return (
                selectedAnomalousRows.includes(networkBaseline.id) ||
                selectedBaselineRows.includes(networkBaseline.id)
            );
        });
        modifyBaselineStatuses(selectedFlows, 'BASELINE', refetchFlows);
    }

    function markSelectedAsAnomalous() {
        const selectedFlows = filteredFlows.filter((networkBaseline) => {
            return (
                selectedAnomalousRows.includes(networkBaseline.id) ||
                selectedBaselineRows.includes(networkBaseline.id)
            );
        });
        modifyBaselineStatuses(selectedFlows, 'ANOMALOUS', refetchFlows);
    }

    if (isLoadingNetworkFlows || isModifying) {
        return (
            <Bullseye>
                <Spinner size="lg" />
            </Bullseye>
        );
    }

    return (
        <Stack>
            {(networkFlowsError || modifyError) && (
                <StackItem>
                    <Alert
                        isInline
                        variant="danger"
                        title={networkFlowsError || modifyError}
                        component="p"
                        className="pf-v5-u-mb-sm"
                    />
                </StackItem>
            )}
            <StackItem>
                <Flex>
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <EntityNameSearchInput
                            value={entityNameFilter}
                            setValue={setEntityNameFilter}
                        />
                    </FlexItem>
                    <FlexItem>
                        <AdvancedFlowsFilter
                            filters={advancedFilters}
                            setFilters={setAdvancedFilters}
                            allUniquePorts={allUniquePorts}
                        />
                    </FlexItem>
                </Flex>
            </StackItem>
            <Divider component="hr" className="pf-v5-u-py-md" />
            <StackItem>
                <Toolbar className="pf-v5-u-p-0">
                    <ToolbarContent className="pf-v5-u-px-0">
                        <ToolbarItem>
                            <FlowsTableHeaderText type={edgeState} numFlows={totalFlows} />
                        </ToolbarItem>
                        {hasWriteAccessForActions && (
                            <ToolbarItem align={{ default: 'alignRight' }}>
                                <FlowsBulkActions
                                    type="active"
                                    selectedRows={selectedRows}
                                    onClearSelectedRows={() => {
                                        setSelectedAnomalousRows([]);
                                        setSelectedBaselineRows([]);
                                    }}
                                    markSelectedAsAnomalous={markSelectedAsAnomalous}
                                    addSelectedToBaseline={addSelectedToBaseline}
                                />
                            </ToolbarItem>
                        )}
                    </ToolbarContent>
                </Toolbar>
            </StackItem>
            <StackItem>
                <Stack hasGutter>
                    <StackItem>
                        <ExpandableSection
                            toggleText={`${numAnomalousFlows} anomalous ${pluralize(
                                'flow',
                                numAnomalousFlows
                            )}`}
                            onToggle={(e, isExpanded) => toggleAnomalousFlowsExpandable(isExpanded)}
                            isExpanded={isAnomalousFlowsExpanded}
                        >
                            {numAnomalousFlows > 0 ? (
                                <FlowsTable
                                    label="Deployment flows"
                                    flows={anomalousFlows}
                                    numFlows={numAnomalousFlows}
                                    expandedRows={expandedRows}
                                    setExpandedRows={setExpandedRows}
                                    selectedRows={selectedAnomalousRows}
                                    setSelectedRows={setSelectedAnomalousRows}
                                    addToBaseline={addToBaseline}
                                    markAsAnomalous={markAsAnomalous}
                                    numExtraneousEgressFlows={numExtraneousEgressFlows}
                                    numExtraneousIngressFlows={numExtraneousIngressFlows}
                                    isEditable={hasWriteAccessForActions}
                                    onSelectFlow={onSelectFlow}
                                />
                            ) : (
                                <EmptyState>No anomalous flows</EmptyState>
                            )}
                        </ExpandableSection>
                    </StackItem>
                    <StackItem>
                        <ExpandableSection
                            toggleText={`${numBaselineFlows} baseline ${pluralize(
                                'flow',
                                numBaselineFlows
                            )}`}
                            onToggle={(e, isExpanded) => toggleBaselineFlowsExpandable(isExpanded)}
                            isExpanded={isBaselineFlowsExpanded}
                        >
                            {numBaselineFlows > 0 ? (
                                <FlowsTable
                                    label="Deployment flows"
                                    flows={baselineFlows}
                                    numFlows={numBaselineFlows}
                                    expandedRows={expandedRows}
                                    setExpandedRows={setExpandedRows}
                                    selectedRows={selectedBaselineRows}
                                    setSelectedRows={setSelectedBaselineRows}
                                    addToBaseline={addToBaseline}
                                    markAsAnomalous={markAsAnomalous}
                                    numExtraneousEgressFlows={numExtraneousEgressFlows}
                                    numExtraneousIngressFlows={numExtraneousIngressFlows}
                                    isEditable={hasWriteAccessForActions}
                                    onSelectFlow={onSelectFlow}
                                />
                            ) : (
                                <EmptyState>No anomalous flows</EmptyState>
                            )}
                        </ExpandableSection>
                    </StackItem>
                </Stack>
            </StackItem>
        </Stack>
    );
}

export default InternalFlows;
