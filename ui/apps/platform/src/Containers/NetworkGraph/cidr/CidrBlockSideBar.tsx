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
import { EdgeModel } from '@patternfly/react-topology';

import { getNodeById } from '../utils/networkGraphUtils';
import { getAllUniquePorts, getNumFlows } from '../utils/flowUtils';
import { AdvancedFlowsFilterType } from '../common/AdvancedFlowsFilter/types';
import { Flow } from '../types/flow.type';
import { CIDRBlockNodeModel, CustomNodeModel } from '../types/topology.type';

import { CidrBlockIcon } from '../common/NetworkGraphIcons';
import AdvancedFlowsFilter, {
    defaultAdvancedFlowsFilters,
} from '../common/AdvancedFlowsFilter/AdvancedFlowsFilter';
import EntityNameSearchInput from '../common/EntityNameSearchInput';
import FlowsTable from '../common/FlowsTable';
import FlowsTableHeaderText from '../common/FlowsTableHeaderText';
import FlowsBulkActions from '../common/FlowsBulkActions';

type CidrBlockSideBarProps = {
    id: string;
    nodes: CustomNodeModel[];
    edges: EdgeModel[];
};

const flows: Flow[] = [
    {
        id: 'Deployment 1-naples-Ingress-Many-TCP',
        type: 'Deployment',
        entity: 'Deployment 1',
        namespace: 'naples',
        direction: 'Ingress',
        port: '9000',
        protocol: 'TCP',
        isAnomalous: true,
        children: [],
    },
    {
        id: 'Deployment 2-naples-Ingress-Many-UDP',
        type: 'Deployment',
        entity: 'Deployment 2',
        namespace: 'naples',
        direction: 'Ingress',
        port: '8080',
        protocol: 'UDP',
        isAnomalous: false,
        children: [],
    },
    {
        id: 'Deployment 3-naples-Egress-7777-UDP',
        type: 'Deployment',
        entity: 'Deployment 3',
        namespace: 'naples',
        direction: 'Egress',
        port: '7777',
        protocol: 'UDP',
        isAnomalous: false,
        children: [],
    },
];

// eslint-disable-next-line @typescript-eslint/no-unused-vars
function CidrBlockSideBar({ id, nodes, edges }: CidrBlockSideBarProps): ReactElement {
    // component state
    const [entityNameFilter, setEntityNameFilter] = React.useState<string>('');
    const [advancedFilters, setAdvancedFilters] = React.useState<AdvancedFlowsFilterType>(
        defaultAdvancedFlowsFilters
    );
    const initialExpandedRows = flows
        .filter((row) => row.children && !!row.children.length)
        .map((row) => row.id); // Default to all expanded
    const [expandedRows, setExpandedRows] = React.useState<string[]>(initialExpandedRows);
    const [selectedRows, setSelectedRows] = React.useState<string[]>([]);

    // derived data
    const cidrBlockNode = getNodeById(nodes, id) as CIDRBlockNodeModel;
    const numFlows = getNumFlows(flows);
    const allUniquePorts = getAllUniquePorts(flows);

    return (
        <Stack>
            <StackItem>
                <Flex direction={{ default: 'row' }} className="pf-u-p-md pf-u-mb-0">
                    <FlexItem>
                        <CidrBlockIcon />
                    </FlexItem>
                    <FlexItem>
                        <TextContent>
                            <Text component={TextVariants.h1} className="pf-u-font-size-xl">
                                {cidrBlockNode?.label}
                            </Text>
                        </TextContent>
                        <TextContent>
                            <Text
                                component={TextVariants.h2}
                                className="pf-u-font-size-sm pf-u-color-200"
                            >
                                {cidrBlockNode?.data.externalSource.cidr}
                            </Text>
                        </TextContent>
                    </FlexItem>
                </Flex>
            </StackItem>
            <StackItem isFilled style={{ overflow: 'auto' }} className="pf-u-p-md">
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
                                    <FlowsTableHeaderText type="active" numFlows={numFlows} />
                                </ToolbarItem>
                                <ToolbarItem alignment={{ default: 'alignRight' }}>
                                    <FlowsBulkActions
                                        type="active"
                                        selectedRows={selectedRows}
                                        onClearSelectedRows={() => setSelectedRows([])}
                                    />
                                </ToolbarItem>
                            </ToolbarContent>
                        </Toolbar>
                    </StackItem>
                    <StackItem>
                        <FlowsTable
                            label="Cidr block flows"
                            flows={flows}
                            numFlows={numFlows}
                            expandedRows={expandedRows}
                            setExpandedRows={setExpandedRows}
                            selectedRows={selectedRows}
                            setSelectedRows={setSelectedRows}
                            isEditable
                        />
                    </StackItem>
                </Stack>
            </StackItem>
        </Stack>
    );
}

export default CidrBlockSideBar;
