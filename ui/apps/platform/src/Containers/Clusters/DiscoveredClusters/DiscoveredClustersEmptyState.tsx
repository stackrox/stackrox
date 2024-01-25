import React, { ReactElement } from 'react';
import {
    Bullseye,
    EmptyState,
    EmptyStateBody,
    EmptyStateIcon,
    Flex,
    Text,
    Title,
} from '@patternfly/react-core';
import { SearchIcon } from '@patternfly/react-icons';
import { Tbody, Td, Tr } from '@patternfly/react-table';

export type DiscoveredClustersEmptyStateProps = {
    colSpan: number;
    hasFilter: boolean;
};

function DiscoveredClustersEmptyState({
    colSpan,
    hasFilter,
}: DiscoveredClustersEmptyStateProps): ReactElement {
    return (
        <Tbody>
            <Tr>
                <Td colSpan={colSpan}>
                    <Bullseye>
                        {hasFilter ? (
                            <EmptyState>
                                <EmptyStateIcon icon={SearchIcon} />
                                <EmptyStateBody>
                                    <Flex direction={{ default: 'column' }}>
                                        <Title headingLevel="h2">
                                            No discovered clusters found
                                        </Title>
                                        <Text>Modify filters and try again</Text>
                                    </Flex>
                                </EmptyStateBody>
                            </EmptyState>
                        ) : (
                            <EmptyState>
                                <EmptyStateBody>
                                    <Title headingLevel="h2">No discovered clusters</Title>
                                </EmptyStateBody>
                            </EmptyState>
                        )}
                    </Bullseye>
                </Td>
            </Tr>
        </Tbody>
    );
}

export default DiscoveredClustersEmptyState;
