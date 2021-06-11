import React, { ReactElement } from 'react';
import { SelectOption } from '@patternfly/react-core';
import { TableComposable, Tbody, Td, Thead, Th, Tr } from '@patternfly/react-table';

import { accessControl as accessTypeLabels } from 'messages/common';
import { AccessLevel } from 'services/RolesService';

import SelectSingle from '../SelectSingle'; // TODO import from where?
import { ReadAccessIcon, WriteAccessIcon } from './AccessIcons';

export type ResourcesTableProps = {
    resourceToAccess: Record<string, AccessLevel>;
    setResourceValue: (resource: string, value: string) => void;
    isDisabled: boolean;
};

function ResourcesTable({
    resourceToAccess,
    setResourceValue,
    isDisabled,
}: ResourcesTableProps): ReactElement {
    // TODO Access level does not need excess width.
    return (
        <TableComposable aria-label="Resources" variant="compact">
            <Thead>
                <Tr>
                    <Th key="resourceName">Resource</Th>
                    <Th key="read">Read</Th>
                    <Th key="write">Write</Th>
                    <Th key="accessLevel">Access level</Th>
                </Tr>
            </Thead>
            <Tbody>
                {Object.entries(resourceToAccess).map(([resource, accessType]) => (
                    <Tr key={resource}>
                        <Td key="resourceName" dataLabel="Resource">
                            {resource}
                        </Td>
                        <Td key="read" dataLabel="Read" data-testid="read">
                            <ReadAccessIcon accessType={accessType} />
                        </Td>
                        <Td key="write" dataLabel="Write" data-testid="write">
                            <WriteAccessIcon accessType={accessType} />
                        </Td>
                        <Td key="accessLevel" dataLabel="Access level">
                            <SelectSingle
                                id={resource}
                                value={accessType}
                                setFieldValue={setResourceValue}
                                isDisabled={isDisabled}
                            >
                                {Object.entries(accessTypeLabels).map(([id, name]) => (
                                    <SelectOption key={id} value={id}>
                                        {name}
                                    </SelectOption>
                                ))}
                            </SelectSingle>
                        </Td>
                    </Tr>
                ))}
            </Tbody>
        </TableComposable>
    );
}

export default ResourcesTable;
