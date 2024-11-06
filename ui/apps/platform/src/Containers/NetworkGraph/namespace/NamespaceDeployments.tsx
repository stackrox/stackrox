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
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import usePagination from 'hooks/patternfly/usePagination';

type NamespaceDeploymentsProps = {
    deployments: { id: string; name: string; numFlows: number }[];
    onNodeSelect: (id: string) => void;
};

function NamespaceDeployments({ deployments, onNodeSelect }: NamespaceDeploymentsProps) {
    const [searchValue, setSearchValue] = React.useState('');
    const { page, perPage, onSetPage, onPerPageSelect } = usePagination();

    function onSearchInputChange(_event, value) {
        setSearchValue(value);
    }

    const onNodeSelectHandler = (deployment) => () => {
        onNodeSelect(deployment.id);
    };

    const filteredDeployments = deployments.filter((deployment) => {
        return deployment.name.includes(searchValue);
    });

    return (
        <div className="pf-v5-u-h-100 pf-v5-u-p-md">
            <Stack hasGutter>
                <StackItem>
                    <SearchInput
                        placeholder="Find by deployment name"
                        value={searchValue}
                        onChange={onSearchInputChange}
                        onClear={() => onSearchInputChange(null, '')}
                    />
                </StackItem>
                <StackItem>
                    <Flex
                        direction={{ default: 'row' }}
                        alignItems={{ default: 'alignItemsCenter' }}
                    >
                        <FlexItem flex={{ default: 'flex_1' }}>
                            <TextContent>
                                <Text component="h2">
                                    {filteredDeployments.length} results found
                                </Text>
                            </TextContent>
                        </FlexItem>
                        <FlexItem>
                            <Pagination
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
                    <Table aria-label="Simple table" variant="compact">
                        <Thead>
                            <Tr>
                                <Th>Deployment</Th>
                                <Th modifier="fitContent">Active traffic</Th>
                            </Tr>
                        </Thead>
                        <Tbody>
                            {filteredDeployments.map((deployment) => (
                                <Tr key={deployment.id}>
                                    <Td dataLabel="Deployment">
                                        <Button
                                            variant="link"
                                            isInline
                                            onClick={onNodeSelectHandler(deployment)}
                                        >
                                            {deployment.name}
                                        </Button>
                                    </Td>
                                    <Td dataLabel="Active traffic">{deployment.numFlows} flows</Td>
                                </Tr>
                            ))}
                        </Tbody>
                    </Table>
                </StackItem>
            </Stack>
        </div>
    );
}

export default NamespaceDeployments;
