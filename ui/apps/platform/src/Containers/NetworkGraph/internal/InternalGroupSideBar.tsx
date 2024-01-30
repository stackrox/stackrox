import React, { ReactElement } from 'react';
import {
    Button,
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

import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { getEdgesByNodeId } from '../utils/networkGraphUtils';
import {
    CustomEdgeModel,
    CustomNodeModel,
    InternalGroupNodeModel,
    isOfType,
} from '../types/topology.type';

import EntityNameSearchInput from '../common/EntityNameSearchInput';

type InternalGroupSideBarProps = {
    selectedNode: InternalGroupNodeModel;
    nodes: CustomNodeModel[];
    edges: CustomEdgeModel[];
    onNodeSelect: (id: string) => void;
};

const columnNames = {
    entity: 'Entity',
    address: 'Address',
    activeTraffic: 'Active traffic',
};

function InternalGroupSideBar({
    selectedNode,
    nodes,
    edges,
    onNodeSelect,
}: InternalGroupSideBarProps): ReactElement {
    // component state
    const [entityNameFilter, setEntityNameFilter] = React.useState<string>('');

    // derived data
    const internalNodes = nodes.filter(isOfType('INTERNAL_ENTITIES'));

    const filteredInternalNodes = entityNameFilter
        ? internalNodes.filter(({ label }) => {
              if (label) {
                  return label.includes(entityNameFilter);
              }
              return false;
          })
        : internalNodes;

    return (
        <Stack>
            <StackItem>
                <Flex direction={{ default: 'row' }} className="pf-u-p-md pf-u-mb-0">
                    <FlexItem>
                        <TextContent>
                            <Text component={TextVariants.h2} className="pf-u-font-size-xl">
                                {selectedNode.label}
                            </Text>
                        </TextContent>
                        <TextContent>
                            <Text
                                component={TextVariants.h3}
                                className="pf-u-font-size-sm pf-u-color-200"
                            >
                                Unspecified connected entities within your cluster
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
                        </Flex>
                    </StackItem>
                    <Divider component="hr" className="pf-u-py-md" />
                    <StackItem className="pf-u-pb-md">
                        <Toolbar className="pf-u-p-0">
                            <ToolbarContent className="pf-u-px-0">
                                <ToolbarItem>
                                    <TextContent>
                                        <Text component={TextVariants.h3}>
                                            {filteredInternalNodes.length} results found
                                        </Text>
                                    </TextContent>
                                </ToolbarItem>
                            </ToolbarContent>
                        </Toolbar>
                    </StackItem>
                    <StackItem>
                        <TableComposable aria-label="Internal to cluster table" variant="compact">
                            <Thead>
                                <Tr>
                                    <Th width={50}>{columnNames.entity}</Th>
                                    <Th>{columnNames.activeTraffic}</Th>
                                </Tr>
                            </Thead>
                            <Tbody>
                                {filteredInternalNodes.map((node) => {
                                    const entityName = node.label;
                                    const relevantEdges = getEdgesByNodeId(edges, node.id);
                                    return (
                                        <Tr key={node.id}>
                                            <Td dataLabel={columnNames.entity}>
                                                <Button
                                                    variant="link"
                                                    isInline
                                                    onClick={() => onNodeSelect(node.id)}
                                                >
                                                    {entityName}
                                                </Button>
                                            </Td>
                                            <Td dataLabel={columnNames.activeTraffic}>
                                                {relevantEdges.length}
                                            </Td>
                                        </Tr>
                                    );
                                })}
                            </Tbody>
                        </TableComposable>
                    </StackItem>
                </Stack>
            </StackItem>
        </Stack>
    );
}

export default InternalGroupSideBar;
