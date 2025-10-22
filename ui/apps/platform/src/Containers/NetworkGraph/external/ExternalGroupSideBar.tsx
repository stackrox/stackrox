import { useState } from 'react';
import type { ReactElement } from 'react';
import {
    Button,
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

import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { getEdgesByNodeId, getNodeById } from '../utils/networkGraphUtils';
import { isOfType } from '../types/topology.type';
import type {
    CIDRBlockNodeModel,
    CustomEdgeModel,
    CustomNodeModel,
    ExternalEntitiesNodeModel,
    ExternalGroupNodeModel,
} from '../types/topology.type';

import { CidrBlockIcon, ExternalEntitiesIcon } from '../common/NetworkGraphIcons';
import EntityNameSearchInput from '../common/EntityNameSearchInput';

type ExternalGroupSideBarProps = {
    labelledById: string; // corresponds to aria-labelledby prop of TopologySideBar
    id: string;
    nodes: CustomNodeModel[];
    edges: CustomEdgeModel[];
    onNodeSelect: (id: string) => void;
};

function ExternalGroupSideBar({
    labelledById,
    id,
    nodes,
    edges,
    onNodeSelect,
}: ExternalGroupSideBarProps): ReactElement {
    // component state
    const [entityNameFilter, setEntityNameFilter] = useState<string>('');

    // derived data
    const externalGroupNode = getNodeById(nodes, id) as ExternalGroupNodeModel;
    const externalNodes = [
        ...nodes.filter(isOfType('CIDR_BLOCK')),
        ...nodes.filter(isOfType('EXTERNAL_ENTITIES')),
    ];
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
                <Flex direction={{ default: 'row' }} className="pf-v5-u-p-md pf-v5-u-mb-0">
                    <FlexItem>
                        <Title headingLevel="h2" id={labelledById}>
                            {externalGroupNode?.label}
                        </Title>
                        <Text className="pf-v5-u-font-size-sm pf-v5-u-color-200">
                            Connected entities outside your cluster
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
                        </Flex>
                    </StackItem>
                    <Divider component="hr" className="pf-v5-u-py-md" />
                    <StackItem className="pf-v5-u-pb-md">
                        <Toolbar className="pf-v5-u-p-0">
                            <ToolbarContent className="pf-v5-u-px-0">
                                <ToolbarItem>
                                    <Title headingLevel="h3">
                                        {filteredExternalNodes.length} results found
                                    </Title>
                                </ToolbarItem>
                            </ToolbarContent>
                        </Toolbar>
                    </StackItem>
                    <StackItem>
                        <Table aria-label="External to cluster table" variant="compact">
                            <Thead>
                                <Tr>
                                    <Th width={50}>Entity</Th>
                                    <Th>Address</Th>
                                    <Th>Active traffic</Th>
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
                                        <Tr key={externalNode.id}>
                                            <Td dataLabel="Entity">
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
                                            <Td dataLabel="Address">{address}</Td>
                                            <Td dataLabel="Active traffic">
                                                {relevantEdges.length}
                                            </Td>
                                        </Tr>
                                    );
                                })}
                            </Tbody>
                        </Table>
                    </StackItem>
                </Stack>
            </StackItem>
        </Stack>
    );
}

export default ExternalGroupSideBar;
