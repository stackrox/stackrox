import React from 'react';
import {
    Button,
    Flex,
    FlexItem,
    Pagination,
    SearchInput,
    Stack,
    StackItem,
    Text,
    TextContent,
} from '@patternfly/react-core';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import usePagination from 'hooks/patternfly/usePagination';
import { DeploymentIcon } from '../common/NetworkGraphIcons';

const columnNames = {
    DEPLOYMENT: 'Deployment',
    ACTIVE_TRAFFIC: 'Active traffic',
};

type NamespaceDeploymentsProps = {
    deployments: { name: string; numFlows: number }[];
};

function NamespaceDeployments({ deployments }: NamespaceDeploymentsProps) {
    const [searchValue, setSearchValue] = React.useState('');
    const { page, perPage, onSetPage, onPerPageSelect } = usePagination();

    const onChange = (newValue: string) => {
        setSearchValue(newValue);
    };

    const filteredDeployments = deployments.filter((deployment) => {
        return deployment.name.includes(searchValue);
    });

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
                    <SearchInput
                        placeholder="Find by deployment name"
                        value={searchValue}
                        onChange={onChange}
                        onClear={() => onChange('')}
                    />
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
                            {filteredDeployments.map((deployment) => (
                                <Tr key={deployment.name}>
                                    <Td dataLabel={columnNames.DEPLOYMENT}>
                                        <Flex spaceItems={{ default: 'spaceItemsMd' }}>
                                            <FlexItem>
                                                <DeploymentIcon />
                                            </FlexItem>
                                            <FlexItem>
                                                <Button variant="link" isInline>
                                                    {deployment.name}
                                                </Button>
                                            </FlexItem>
                                        </Flex>
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
