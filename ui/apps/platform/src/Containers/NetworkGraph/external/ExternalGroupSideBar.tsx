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
import { getEdgesByNodeId, getNodeById } from '../utils/networkGraphUtils';
import {
    CIDRBlockNodeModel,
    CustomEdgeModel,
    CustomNodeModel,
    ExternalEntitiesNodeModel,
    ExternalGroupNodeModel,
} from '../types/topology.type';

import { CidrBlockIcon, ExternalEntitiesIcon } from '../common/NetworkGraphIcons';
import EntityNameSearchInput from '../common/EntityNameSearchInput';

type ExternalGroupSideBarProps = {
    id: string;
    nodes: CustomNodeModel[];
    edges: CustomEdgeModel[];
    onNodeSelect: (id: string) => void;
};

const columnNames = {
    entity: 'Entity',
    address: 'Address',
    activeTraffic: 'Active traffic',
};

// eslint-disable-next-line @typescript-eslint/no-unused-vars
function ExternalGroupSideBar({
    id,
    nodes,
    edges,
    onNodeSelect,
}: ExternalGroupSideBarProps): ReactElement {
    // component state
    const [entityNameFilter, setEntityNameFilter] = React.useState<string>('');

    // derived data
    const externalGroupNode = getNodeById(nodes, id) as ExternalGroupNodeModel;
    const externalNodes = nodes.filter(
        (node) => node.data.type === 'CIDR_BLOCK' || node.data.type === 'EXTERNAL_ENTITIES'
    ) as (ExternalEntitiesNodeModel | CIDRBlockNodeModel)[];

    const onNodeSelectHandler =
        (externalNode: ExternalEntitiesNodeModel | CIDRBlockNodeModel) => () => {
            onNodeSelect(externalNode.id);
        };

    const filteredExternalNodes = entityNameFilter
        ? externalNodes.filter((externalNode) => {
              if (externalNode.label) {
                  return externalNode.label.includes(entityNameFilter);
              }
              return false;
          })
        : externalNodes;

    return (
        <Stack>
            <StackItem>
                <Flex direction={{ default: 'row' }} className="pf-u-p-md pf-u-mb-0">
                    <FlexItem>
                        <TextContent>
                            <Text component={TextVariants.h2} className="pf-u-font-size-xl">
                                {externalGroupNode?.label}
                            </Text>
                        </TextContent>
                        <TextContent>
                            <Text
                                component={TextVariants.h3}
                                className="pf-u-font-size-sm pf-u-color-200"
                            >
                                Connected entities outside your cluster
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
                                            {filteredExternalNodes.length} results found
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
                                {filteredExternalNodes.map((externalNode) => {
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
                                                    <FlexItem>
                                                        <Button
                                                            variant="link"
                                                            isInline
                                                            onClick={onNodeSelectHandler(
                                                                externalNode
                                                            )}
                                                        >
                                                            {entityName}
                                                        </Button>
                                                    </FlexItem>
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

export default ExternalGroupSideBar;
