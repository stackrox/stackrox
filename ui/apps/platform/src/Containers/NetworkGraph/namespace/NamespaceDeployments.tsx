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

const columnNames = {
    DEPLOYMENT: 'Deployment',
    ACTIVE_TRAFFIC: 'Active traffic',
};

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
        <div className="pf-u-h-100 pf-u-p-md">
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
                            {filteredDeployments.map((deployment) => (
                                <Tr key={deployment.id}>
                                    <Td dataLabel={columnNames.DEPLOYMENT}>
                                        <Button
                                            variant="link"
                                            isInline
                                            onClick={onNodeSelectHandler(deployment)}
                                        >
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
