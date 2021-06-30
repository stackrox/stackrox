import React, { ReactElement } from 'react';
import { Badge, SelectOption } from '@patternfly/react-core';
import { TableComposable, Tbody, Td, Thead, Th, Tr } from '@patternfly/react-table';

import { accessControl as accessTypeLabels } from 'messages/common';
import { PermissionsMap } from 'services/RolesService';

import SelectSingle from '../SelectSingle'; // TODO import from where?
import { ReadAccessIcon, WriteAccessIcon } from './AccessIcons';
import { getReadAccessCount, getWriteAccessCount } from './permissionSets.utils';

export type PermissionsTableProps = {
    resourceToAccess: PermissionsMap;
    setResourceValue: (resource: string, value: string) => void;
    isDisabled: boolean;
};

function PermissionsTable({
    resourceToAccess,
    setResourceValue,
    isDisabled,
}: PermissionsTableProps): ReactElement {
    const resourceToAccessEntries = Object.entries(resourceToAccess);

    // TODO Access level does not need excess width.
    return (
        <TableComposable variant="compact" isStickyHeader>
            <Thead>
                <Tr>
                    <Th key="resourceName">
                        Resource
                        <Badge isRead className="pf-u-ml-sm">
                            {resourceToAccessEntries.length}
                        </Badge>
                    </Th>
                    <Th key="read">
                        Read
                        <Badge isRead className="pf-u-ml-sm">
                            {getReadAccessCount(resourceToAccess)}
                        </Badge>
                    </Th>
                    <Th key="write">
                        Write
                        <Badge isRead className="pf-u-ml-sm">
                            {getWriteAccessCount(resourceToAccess)}
                        </Badge>
                    </Th>
                    <Th key="accessLevel">Access level</Th>
                </Tr>
            </Thead>
            <Tbody>
                {resourceToAccessEntries.map(([resource, accessLevel]) => (
                    <Tr key={resource}>
                        <Td key="resourceName" dataLabel="Resource">
                            {resource}
                        </Td>
                        <Td key="read" dataLabel="Read" data-testid="read">
                            <ReadAccessIcon accessLevel={accessLevel} />
                        </Td>
                        <Td key="write" dataLabel="Write" data-testid="write">
                            <WriteAccessIcon accessLevel={accessLevel} />
                        </Td>
                        <Td key="accessLevel" dataLabel="Access level">
                            <SelectSingle
                                id={resource}
                                value={accessLevel}
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

export default PermissionsTable;
