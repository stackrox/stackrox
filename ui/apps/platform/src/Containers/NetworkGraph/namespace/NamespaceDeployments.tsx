import React from 'react';
import {
    Button,
    Flex,
    FlexItem,
    Pagination,
    Stack,
    StackItem,
    Text,
    TextContent,
} from '@patternfly/react-core';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import usePagination from 'hooks/patternfly/usePagination';

const columnNames = {
    DEPLOYMENT: 'Deployment',
    ACTIVE_TRAFFIC: 'Active traffic',
};

const deployments = [
    { name: 'Sensor', numFlows: '1' },
    { name: 'Central', numFlows: '1' },
];

function NamespaceDeployments() {
    const { page, perPage, onSetPage, onPerPageSelect } = usePagination();

    return (
        <div className="pf-u-h-100 pf-u-p-md">
            <Stack hasGutter>
                <StackItem>
                    <Flex
                        direction={{ default: 'row' }}
                        alignItems={{ default: 'alignItemsCenter' }}
                    >
                        <FlexItem flex={{ default: 'flex_1' }}>
                            <TextContent>
                                <Text component="h2">6 results found</Text>
                            </TextContent>
                        </FlexItem>
                        <FlexItem>
                            <Pagination
                                perPageComponent="button"
                                itemCount={deployments.length}
                                perPage={perPage}
                                page={page}
                                onSetPage={onSetPage}
                                widgetId="networkgraph-namespace-deployments-pagination"
                                onPerPageSelect={onPerPageSelect}
                                isCompact
                            />
                        </FlexItem>
                    </Flex>
                </StackItem>
                <StackItem>
                    <TableComposable aria-label="Simple table" variant="compact">
                        <Thead>
                            <Tr>
                                <Th>{columnNames.DEPLOYMENT}</Th>
                                <Th>{columnNames.ACTIVE_TRAFFIC}</Th>
                            </Tr>
                        </Thead>
                        <Tbody>
                            {deployments.map((deployment) => (
                                <Tr key={deployment.name}>
                                    <Td dataLabel={columnNames.DEPLOYMENT}>
                                        <Button variant="link" isInline>
                                            {deployment.name}
                                        </Button>
                                    </Td>
                                    <Td dataLabel={columnNames.ACTIVE_TRAFFIC}>
                                        {deployment.numFlows} flows
                                    </Td>
                                </Tr>
                            ))}
                        </Tbody>
                    </TableComposable>
                </StackItem>
            </Stack>
        </div>
    );
}

export default NamespaceDeployments;
