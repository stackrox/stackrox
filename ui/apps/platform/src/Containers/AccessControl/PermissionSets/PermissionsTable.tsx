import React, { ReactElement } from 'react';
import { Badge, SelectOption } from '@patternfly/react-core';
import { TableComposable, Tbody, Td, Thead, Th, Tr } from '@patternfly/react-table';

import SelectSingle from 'Components/SelectSingle';
import { accessControl as accessTypeLabels } from 'messages/common';
import { PermissionsMap } from 'services/RolesService';

import { ReadAccessIcon, WriteAccessIcon } from './AccessIcons';
import { getReadAccessCount, getWriteAccessCount } from './permissionSets.utils';
import ResourceDescription from './ResourceDescription';

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

    return (
        <TableComposable variant="compact" isStickyHeader>
            <Thead>
                <Tr>
                    <Th width={20}>
                        Resource
                        <Badge isRead className="pf-u-ml-sm">
                            {resourceToAccessEntries.length}
                        </Badge>
                    </Th>
                    <Th width={40}>Description</Th>
                    <Th width={10}>
                        Read
                        <Badge isRead className="pf-u-ml-sm">
                            {getReadAccessCount(resourceToAccess)}
                        </Badge>
                    </Th>
                    <Th width={10}>
                        Write
                        <Badge isRead className="pf-u-ml-sm">
                            {getWriteAccessCount(resourceToAccess)}
                        </Badge>
                    </Th>
                    <Th width={20}>Access level</Th>
                </Tr>
            </Thead>
            <Tbody>
                {resourceToAccessEntries.map(([resource, accessLevel]) => (
                    <Tr key={resource}>
                        <Td dataLabel="Resource">{resource}</Td>
                        <Td dataLabel="Description">
                            <ResourceDescription resource={resource} />
                        </Td>
                        <Td dataLabel="Read" data-testid="read">
                            <ReadAccessIcon accessLevel={accessLevel} />
                        </Td>
                        <Td dataLabel="Write" data-testid="write">
                            <WriteAccessIcon accessLevel={accessLevel} />
                        </Td>
                        <Td dataLabel="Access level">
                            <SelectSingle
                                id={resource}
                                value={accessLevel}
                                handleSelect={setResourceValue}
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
