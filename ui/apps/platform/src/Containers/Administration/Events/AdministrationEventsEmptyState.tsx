import type { ReactElement } from 'react';
import { Bullseye, Content, EmptyState, EmptyStateBody, Flex, Title } from '@patternfly/react-core';
import { SearchIcon } from '@patternfly/react-icons';
import { Tbody, Td, Tr } from '@patternfly/react-table';

export type AdministrationEventsEmptyStateProps = {
    colSpan: number;
    hasFilter: boolean;
};

function AdministrationEventsEmptyState({
    colSpan,
    hasFilter,
}: AdministrationEventsEmptyStateProps): ReactElement {
    return (
        <Tbody>
            <Tr>
                <Td colSpan={colSpan}>
                    <Bullseye>
                        {hasFilter ? (
                            <EmptyState icon={SearchIcon}>
                                <EmptyStateBody>
                                    <Flex direction={{ default: 'column' }}>
                                        <Title headingLevel="h2">
                                            No administration events found
                                        </Title>
                                        <Content component="p">
                                            Modify filters and try again
                                        </Content>
                                    </Flex>
                                </EmptyStateBody>
                            </EmptyState>
                        ) : (
                            <EmptyState>
                                <EmptyStateBody>
                                    <Title headingLevel="h2">No administration events</Title>
                                </EmptyStateBody>
                            </EmptyState>
                        )}
                    </Bullseye>
                </Td>
            </Tr>
        </Tbody>
    );
}

export default AdministrationEventsEmptyState;
