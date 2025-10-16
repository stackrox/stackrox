import type { ReactElement } from 'react';
import {
    Divider,
    Flex,
    FlexItem,
    Stack,
    StackItem,
    Text,
    Title,
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
import type { AdvancedFlowsFilterType } from '../common/AdvancedFlowsFilter/types';
import type { CustomEdgeModel, CustomNodeModel } from '../types/topology.type';

import AdvancedFlowsFilter, {
    defaultAdvancedFlowsFilters,
} from '../common/AdvancedFlowsFilter/AdvancedFlowsFilter';
import EntityNameSearchInput from '../common/EntityNameSearchInput';
import FlowsTable from '../common/FlowsTable';
import FlowsTableHeaderText from '../common/FlowsTableHeaderText';

export type GenericEntitiesSideBarProps = {
    labelledById: string; // corresponds to aria-labelledby prop of TopologySideBar
    id: string;
    nodes: CustomNodeModel[];
    edges: CustomEdgeModel[];
    onNodeSelect: (id: string) => void;
    EntityHeaderIcon: ReactElement;
    sidebarTitle: string;
    flowTableLabel: string;
};

function GenericEntitiesSideBar({
    labelledById,
    id,
    nodes,
    edges,
    onNodeSelect,
    EntityHeaderIcon,
    sidebarTitle,
    flowTableLabel,
}: GenericEntitiesSideBarProps): ReactElement {
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

    const entityNode = getNodeById(nodes, id);
    const numFlows = getNumFlows(filteredFlows);
    const allUniquePorts = getAllUniquePorts(filteredFlows);

    const onSelectFlow = (entityId: string) => {
        onNodeSelect(entityId);
    };

    return (
        <Stack>
            <StackItem>
                <Flex direction={{ default: 'row' }} className="pf-v5-u-p-md pf-v5-u-mb-0">
                    <FlexItem>{EntityHeaderIcon}</FlexItem>
                    <FlexItem>
                        <Title headingLevel="h2" id={labelledById}>
                            {entityNode?.label}
                        </Title>
                        <Text className="pf-v5-u-font-size-sm pf-v5-u-color-200">
                            {sidebarTitle}
                        </Text>
                    </FlexItem>
                </Flex>
            </StackItem>
            <Divider component="hr" />
            <StackItem isFilled style={{ overflow: 'auto' }}>
                <Stack className="pf-v5-u-p-md">
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
                            label={flowTableLabel}
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

export default GenericEntitiesSideBar;
