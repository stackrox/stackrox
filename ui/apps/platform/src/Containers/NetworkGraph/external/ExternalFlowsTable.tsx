import React, { ReactElement, useState } from 'react';
import {
    Divider,
    Flex,
    FlexItem,
    Stack,
    StackItem,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';

import AdvancedFlowsFilter, {
    defaultAdvancedFlowsFilters,
} from '../common/AdvancedFlowsFilter/AdvancedFlowsFilter';
import { AdvancedFlowsFilterType } from '../common/AdvancedFlowsFilter/types';
import EntityNameSearchInput from '../common/EntityNameSearchInput';
import FlowsTable from '../common/FlowsTable';
import FlowsTableHeaderText from '../common/FlowsTableHeaderText';
import { CustomEdgeModel, CustomNodeModel } from '../types/topology.type';
import {
    filterNetworkFlows,
    getAllUniquePorts,
    getNetworkFlows,
    getNumFlows,
} from '../utils/flowUtils';

type ExternalFlowsTableProps = {
    nodes: CustomNodeModel[];
    edges: CustomEdgeModel[];
    id: string;
    onNodeSelect: (id: string) => void;
};

function ExternalFlowsTable({
    nodes,
    edges,
    id,
    onNodeSelect,
}: ExternalFlowsTableProps): ReactElement {
    const [entityNameFilter, setEntityNameFilter] = useState<string>('');
    const [advancedFilters, setAdvancedFilters] = useState<AdvancedFlowsFilterType>(
        defaultAdvancedFlowsFilters
    );

    const flows = getNetworkFlows(nodes, edges, id);
    const filteredFlows = filterNetworkFlows(flows, entityNameFilter, advancedFilters);
    const initialExpandedRows = filteredFlows
        .filter((row) => row.children && !!row.children.length)
        .map((row) => row.id);

    const [expandedRows, setExpandedRows] = useState<string[]>(initialExpandedRows);
    const [selectedRows, setSelectedRows] = useState<string[]>([]);

    const numFlows = getNumFlows(filteredFlows);
    const allUniquePorts = getAllUniquePorts(filteredFlows);

    const onSelectFlow = (entityId: string) => {
        onNodeSelect(entityId);
    };

    return (
        <Stack>
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
            <StackItem className="pf-v5-u-pb-md">
                <Toolbar className="pf-v5-u-p-0">
                    <ToolbarContent className="pf-v5-u-px-0">
                        <ToolbarItem>
                            <FlowsTableHeaderText type="active" numFlows={numFlows} />
                        </ToolbarItem>
                    </ToolbarContent>
                </Toolbar>
            </StackItem>
            <StackItem>
                <FlowsTable
                    label="External entities flows"
                    flows={filteredFlows}
                    numFlows={numFlows}
                    expandedRows={expandedRows}
                    setExpandedRows={setExpandedRows}
                    selectedRows={selectedRows}
                    setSelectedRows={setSelectedRows}
                    onSelectFlow={onSelectFlow}
                />
            </StackItem>
        </Stack>
    );
}

export default ExternalFlowsTable;
