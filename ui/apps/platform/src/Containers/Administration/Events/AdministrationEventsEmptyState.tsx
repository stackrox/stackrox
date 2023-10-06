import React, { ReactElement } from 'react';
import { Bullseye, EmptyState, EmptyStateBody, EmptyStateIcon } from '@patternfly/react-core';
import { FilterIcon } from '@patternfly/react-icons';
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
                        <EmptyState>
                            {hasFilter && <EmptyStateIcon icon={FilterIcon} />}
                            <EmptyStateBody>
                                {hasFilter ? 'No events found' : 'No events'}
                            </EmptyStateBody>
                        </EmptyState>
                    </Bullseye>
                </Td>
            </Tr>
        </Tbody>
    );
}

export default AdministrationEventsEmptyState;
