import React, { ReactElement } from 'react';
import {
    Divider,
    Flex,
    FlexItem,
    Stack,
    StackItem,
    Text,
    TextContent,
    TextVariants,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';

import { getNodeById } from '../utils/networkGraphUtils';
import {
    filterNetworkFlows,
    getAllUniquePorts,
    getNetworkFlows,
    getNumFlows,
} from '../utils/flowUtils';
import { AdvancedFlowsFilterType } from '../common/AdvancedFlowsFilter/types';
import { CustomEdgeModel, CustomNodeModel } from '../types/topology.type';

import { ExternalEntitiesIcon } from '../common/NetworkGraphIcons';
import AdvancedFlowsFilter, {
    defaultAdvancedFlowsFilters,
} from '../common/AdvancedFlowsFilter/AdvancedFlowsFilter';
import EntityNameSearchInput from '../common/EntityNameSearchInput';
import FlowsTable from '../common/FlowsTable';
import FlowsTableHeaderText from '../common/FlowsTableHeaderText';

type ExternalEntitiesSideBarProps = {
    id: string;
    nodes: CustomNodeModel[];
    edges: CustomEdgeModel[];
    onNodeSelect: (id: string) => void;
};

// eslint-disable-next-line @typescript-eslint/no-unused-vars
function ExternalEntitiesSideBar({
    id,
    nodes,
    edges,
    onNodeSelect,
}: ExternalEntitiesSideBarProps): ReactElement {
    // component state
    const [entityNameFilter, setEntityNameFilter] = React.useState<string>('');
    const [advancedFilters, setAdvancedFilters] = React.useState<AdvancedFlowsFilterType>(
        defaultAdvancedFlowsFilters
    );
    const flows = getNetworkFlows(nodes, edges, id);
    const filteredFlows = filterNetworkFlows(flows, entityNameFilter, advancedFilters);
    const initialExpandedRows = filteredFlows
        .filter((row) => row.children && !!row.children.length)
        .map((row) => row.id); // Default to all expanded
    const [expandedRows, setExpandedRows] = React.useState<string[]>(initialExpandedRows);
    const [selectedRows, setSelectedRows] = React.useState<string[]>([]);

    // derived data
    const externalEntitiesNode = getNodeById(nodes, id);
    const numFlows = getNumFlows(filteredFlows);
    const allUniquePorts = getAllUniquePorts(filteredFlows);

    const onSelectFlow = (entityId: string) => {
        onNodeSelect(entityId);
    };

    return (
        <Stack>
            <StackItem>
                <Flex direction={{ default: 'row' }} className="pf-u-p-md pf-u-mb-0">
                    <FlexItem>
                        <ExternalEntitiesIcon />
                    </FlexItem>
                    <FlexItem>
                        <TextContent>
                            <Text component={TextVariants.h1} className="pf-u-font-size-xl">
                                {externalEntitiesNode?.label}
                            </Text>
                        </TextContent>
                        <TextContent>
                            <Text
                                component={TextVariants.h2}
                                className="pf-u-font-size-sm pf-u-color-200"
                            >
                                Connected Entities Outside Your Cluster
                            </Text>
                        </TextContent>
                    </FlexItem>
                </Flex>
            </StackItem>
            <Divider component="hr" />
            <StackItem isFilled style={{ overflow: 'auto' }}>
                <Stack className="pf-u-p-md">
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
                    <Divider component="hr" className="pf-u-py-md" />
                    <StackItem className="pf-u-pb-md">
                        <Toolbar className="pf-u-p-0">
                            <ToolbarContent className="pf-u-px-0">
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
            </StackItem>
        </Stack>
    );
}

export default ExternalEntitiesSideBar;
