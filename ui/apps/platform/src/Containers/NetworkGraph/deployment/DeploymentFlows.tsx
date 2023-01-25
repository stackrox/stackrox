import React from 'react';
import {
    Alert,
    AlertVariant,
    Bullseye,
    Divider,
    Flex,
    FlexItem,
    Spinner,
    Stack,
    StackItem,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';

import { AdvancedFlowsFilterType } from '../common/AdvancedFlowsFilter/types';
import {
    filterNetworkFlows,
    getAllUniquePorts,
    getNumAllowedEgressFlows,
    getNumAllowedIngressFlows,
    getNumFlows,
} from '../utils/flowUtils';
import { CustomEdgeModel, CustomNodeModel } from '../types/topology.type';

import AdvancedFlowsFilter, {
    defaultAdvancedFlowsFilters,
} from '../common/AdvancedFlowsFilter/AdvancedFlowsFilter';
import EntityNameSearchInput from '../common/EntityNameSearchInput';
import FlowsTable from '../common/FlowsTable';
import FlowsTableHeaderText from '../common/FlowsTableHeaderText';
import FlowsBulkActions from '../common/FlowsBulkActions';

import './DeploymentFlows.css';
import useFetchNetworkFlows from '../api/useFetchNetworkFlows';
import useModifyBaselineStatuses from '../api/useModifyBaselineStatuses';
import { Flow } from '../types/flow.type';
import { EdgeState } from '../components/EdgeStateSelect';

type DeploymentFlowsProps = {
    deploymentId: string;
    nodes: CustomNodeModel[];
    edges: CustomEdgeModel[];
    edgeState: EdgeState;
};

function DeploymentFlows({ deploymentId, nodes, edges, edgeState }: DeploymentFlowsProps) {
    // component state
    const [entityNameFilter, setEntityNameFilter] = React.useState<string>('');
    const [advancedFilters, setAdvancedFilters] = React.useState<AdvancedFlowsFilterType>(
        defaultAdvancedFlowsFilters
    );

    const {
        isLoading,
        error: fetchError,
        data: { networkFlows },
        refetchFlows,
    } = useFetchNetworkFlows({ edges, deploymentId, edgeState });
    const {
        isModifying,
        error: modifyError,
        modifyBaselineStatuses,
    } = useModifyBaselineStatuses(deploymentId);
    const filteredFlows = filterNetworkFlows(networkFlows, entityNameFilter, advancedFilters);

    const initialExpandedRows = filteredFlows
        .filter((row) => row.children && !!row.children.length)
        .map((row) => row.id); // Default to all expanded
    const [expandedRows, setExpandedRows] = React.useState<string[]>(initialExpandedRows);
    const [selectedRows, setSelectedRows] = React.useState<string[]>([]);

    // derived data
    const numFlows = getNumFlows(filteredFlows);
    const allUniquePorts = getAllUniquePorts(networkFlows);
    const numAllowedEgressFlows = getNumAllowedEgressFlows(nodes);
    const numAllowedIngressFlows = getNumAllowedIngressFlows(nodes);
    const totalFlows = numFlows + numAllowedEgressFlows + numAllowedIngressFlows;

    function addToBaseline(flow: Flow) {
        modifyBaselineStatuses([flow], 'BASELINE', refetchFlows);
    }

    function markAsAnomalous(flow: Flow) {
        modifyBaselineStatuses([flow], 'ANOMALOUS', refetchFlows);
    }

    function addSelectedToBaseline() {
        const selectedFlows = filteredFlows.filter((networkBaseline) => {
            return selectedRows.includes(networkBaseline.id);
        });
        modifyBaselineStatuses(selectedFlows, 'BASELINE', refetchFlows);
    }

    function markSelectedAsAnomalous() {
        const selectedFlows = filteredFlows.filter((networkBaseline) => {
            return selectedRows.includes(networkBaseline.id);
        });
        modifyBaselineStatuses(selectedFlows, 'ANOMALOUS', refetchFlows);
    }

    if (isLoading || isModifying) {
        return (
            <Bullseye>
                <Spinner isSVG size="lg" />
            </Bullseye>
        );
    }

    return (
        <div className="pf-u-h-100 pf-u-p-md">
            {(fetchError || modifyError) && (
                <Alert
                    isInline
                    variant={AlertVariant.danger}
                    title={fetchError || modifyError}
                    className="pf-u-mb-sm"
                />
            )}
            <Stack hasGutter>
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
                <Divider component="hr" />
                <StackItem>
                    <Toolbar>
                        <ToolbarContent>
                            <ToolbarItem>
                                <FlowsTableHeaderText type={edgeState} numFlows={totalFlows} />
                            </ToolbarItem>
                            <ToolbarItem alignment={{ default: 'alignRight' }}>
                                <FlowsBulkActions
                                    type="active"
                                    selectedRows={selectedRows}
                                    onClearSelectedRows={() => setSelectedRows([])}
                                    markSelectedAsAnomalous={markSelectedAsAnomalous}
                                    addSelectedToBaseline={addSelectedToBaseline}
                                />
                            </ToolbarItem>
                        </ToolbarContent>
                    </Toolbar>
                </StackItem>
                <StackItem>
                    <FlowsTable
                        label="Deployment flows"
                        flows={filteredFlows}
                        numFlows={numFlows}
                        expandedRows={expandedRows}
                        setExpandedRows={setExpandedRows}
                        selectedRows={selectedRows}
                        setSelectedRows={setSelectedRows}
                        addToBaseline={addToBaseline}
                        markAsAnomalous={markAsAnomalous}
                        numAllowedEgressFlows={numAllowedEgressFlows}
                        numAllowedIngressFlows={numAllowedIngressFlows}
                        isEditable
                    />
                </StackItem>
            </Stack>
        </div>
    );
}

export default DeploymentFlows;
