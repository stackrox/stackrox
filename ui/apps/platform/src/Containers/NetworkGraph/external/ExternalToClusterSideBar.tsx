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

import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { getEdgesByNodeId, getNodeById } from '../utils/networkGraphUtils';
import {
    CIDRBlockNodeModel,
    CustomNodeModel,
    ExternalEntitiesNodeModel,
    ExternalNodeModel,
} from '../types/topology.type';

import { CidrBlockIcon, ExternalEntitiesIcon } from '../common/NetworkGraphIcons';
import EntityNameSearchInput from '../common/EntityNameSearchInput';

type CidrBlockSideBarProps = {
    id: string;
    nodes: CustomNodeModel[];
    edges: EdgeModel[];
};

const columnNames = {
    entity: 'Entity',
    address: 'Address',
    activeTraffic: 'Active traffic',
};

// eslint-disable-next-line @typescript-eslint/no-unused-vars
function ExternalToClusterSideBar({ id, nodes, edges }: CidrBlockSideBarProps): ReactElement {
    // component state
    const [entityNameFilter, setEntityNameFilter] = React.useState<string>('');

    // derived data
    const externalToClusterNode = getNodeById(nodes, id) as ExternalNodeModel;
    const externalNodes = nodes.filter(
        (node) => node.data.type === 'CIDR_BLOCK' || node.data.type === 'EXTERNAL_ENTITIES'
    ) as (ExternalEntitiesNodeModel | CIDRBlockNodeModel)[];

    return (
        <Stack>
            <StackItem>
                <Flex direction={{ default: 'row' }} className="pf-u-p-md pf-u-mb-0">
                    <FlexItem>
                        <TextContent>
                            <Text component={TextVariants.h1} className="pf-u-font-size-xl">
                                {externalToClusterNode?.label}
                            </Text>
                        </TextContent>
                        <TextContent>
                            <Text
                                component={TextVariants.h2}
                                className="pf-u-font-size-sm pf-u-color-200"
                            >
                                Connected entities outside your cluster
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
                        </Flex>
                    </StackItem>
                    <Divider component="hr" />
                    <StackItem>
                        <Toolbar>
                            <ToolbarContent>
                                <ToolbarItem>
                                    <TextContent>
                                        <Text component={TextVariants.h3}>
                                            {externalNodes.length} results found
                                        </Text>
                                    </TextContent>
                                </ToolbarItem>
                            </ToolbarContent>
                        </Toolbar>
                    </StackItem>
                    <StackItem>
                        <TableComposable aria-label="External to cluster table" variant="compact">
                            <Thead>
                                <Tr>
                                    <Th width={50}>{columnNames.entity}</Th>
                                    <Th>{columnNames.address}</Th>
                                    <Th>{columnNames.activeTraffic}</Th>
                                </Tr>
                            </Thead>
                            <Tbody>
                                {externalNodes.map((externalNode) => {
                                    const entityIcon =
                                        externalNode.data.type === 'CIDR_BLOCK' ? (
                                            <CidrBlockIcon />
                                        ) : (
                                            <ExternalEntitiesIcon />
                                        );
                                    const entityName = externalNode.label;
                                    const address =
                                        externalNode.data.type === 'CIDR_BLOCK'
                                            ? externalNode.data.externalSource.cidr
                                            : '';
                                    const relevantEdges = getEdgesByNodeId(edges, externalNode.id);
                                    return (
                                        <Tr>
                                            <Td dataLabel={columnNames.entity}>
                                                <Flex>
                                                    <FlexItem>{entityIcon}</FlexItem>
                                                    <FlexItem>{entityName}</FlexItem>
                                                </Flex>
                                            </Td>
                                            <Td dataLabel={columnNames.address}>{address}</Td>
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

export default ExternalToClusterSideBar;
