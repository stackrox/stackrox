import React, { useState } from 'react';
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

import { filterNetworkFlows, getAllUniquePorts, getNumFlows } from '../utils/flowUtils';

import FlowsTable from '../common/FlowsTable';
import FlowsTableHeaderText from '../common/FlowsTableHeaderText';
import useFetchSimulatedBaselines from '../api/useFetchSimulatedBaselines';
import AdvancedFlowsFilter, {
    defaultAdvancedFlowsFilters,
} from '../common/AdvancedFlowsFilter/AdvancedFlowsFilter';
import { AdvancedFlowsFilterType } from '../common/AdvancedFlowsFilter/types';
import EntityNameSearchInput from '../common/EntityNameSearchInput';

type DeploymentBaselinesSimulatedProps = {
    deploymentId: string;
    onNodeSelect: (id: string) => void;
};

// eslint-disable-next-line @typescript-eslint/no-unused-vars
function DeploymentBaselinesSimulated({
    deploymentId,
    onNodeSelect,
}: DeploymentBaselinesSimulatedProps) {
    // component state
    const {
        isLoading,
        data: { simulatedBaselines },
        error,
    } = useFetchSimulatedBaselines(deploymentId);

    const [entityNameFilter, setEntityNameFilter] = useState<string>('');
    const [advancedFilters, setAdvancedFilters] = useState<AdvancedFlowsFilterType>(
        defaultAdvancedFlowsFilters
    );

    const filteredSimulatedBaselines = filterNetworkFlows(
        simulatedBaselines,
        entityNameFilter,
        advancedFilters
    );

    const initialExpandedRows = filteredSimulatedBaselines
        .filter((row) => row.children && !!row.children.length)
        .map((row) => row.id); // Default to all expanded
    const [expandedRows, setExpandedRows] = useState<string[]>(initialExpandedRows);
    const [selectedRows, setSelectedRows] = useState<string[]>([]);

    // derived data
    const numBaselines = getNumFlows(filteredSimulatedBaselines);
    const allUniquePorts = getAllUniquePorts(filteredSimulatedBaselines);

    const onSelectFlow = (entityId: string) => {
        onNodeSelect(entityId);
    };

    if (isLoading) {
        return (
            <Bullseye>
                <Spinner isSVG size="lg" />
            </Bullseye>
        );
    }

    return (
        <div className="pf-u-h-100">
            {error && (
                <Alert
                    isInline
                    variant={AlertVariant.danger}
                    title={error}
                    className="pf-u-mb-sm"
                />
            )}
            <Stack hasGutter className="pf-u-p-md">
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
                                isBaseline
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
                                <FlowsTableHeaderText
                                    type="baseline simulated"
                                    numFlows={numBaselines}
                                />
                            </ToolbarItem>
                        </ToolbarContent>
                    </Toolbar>
                </StackItem>
                <Divider component="hr" />
                <StackItem>
                    <FlowsTable
                        label="Deployment simulated baselines"
                        flows={filteredSimulatedBaselines}
                        numFlows={numBaselines}
                        expandedRows={expandedRows}
                        setExpandedRows={setExpandedRows}
                        selectedRows={selectedRows}
                        setSelectedRows={setSelectedRows}
                        isEditable={false}
                        isBaselineSimulation
                        onSelectFlow={onSelectFlow}
                    />
                </StackItem>
            </Stack>
        </div>
    );
}

export default DeploymentBaselinesSimulated;
