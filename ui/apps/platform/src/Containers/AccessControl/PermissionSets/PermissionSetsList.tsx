import React, { ReactElement, useState } from 'react';
import { useParams } from 'react-router-dom';

import { defaultColumnClassName, nonSortableHeaderClassName } from 'Components/Table';

import AccessControlListPage from '../AccessControlListPage';
import { Column, PermissionSet } from '../accessControlTypes';

// 2/4 + 2/5 + 1/10 = 0.5 + 0.4 + 0.1 = 1.0
const columns: Column[] = [
    {
        Header: 'Id',
        accessor: 'id',
        headerClassName: 'hidden',
        className: 'hidden',
    },
    {
        Header: 'Name',
        accessor: 'name',
        headerClassName: `w-1/4 ${nonSortableHeaderClassName}`,
        className: `w-1/4 ${defaultColumnClassName}`,
        sortable: false,
    },
    {
        Header: 'Description',
        accessor: 'description',
        headerClassName: `w-2/5 ${nonSortableHeaderClassName}`,
        className: `w-2/5 ${defaultColumnClassName}`,
        sortable: false,
    },
    {
        Header: 'Type',
        accessor: 'type',
        headerClassName: `w-1/10 ${nonSortableHeaderClassName}`,
        className: `w-1/10 ${defaultColumnClassName}`,
        sortable: false,
    },
    {
        Header: 'Roles',
        accessor: 'TODO', // TODO link
        headerClassName: `w-1/4 ${nonSortableHeaderClassName}`,
        className: `w-1/4 ${defaultColumnClassName}`,
        sortable: false,
    },
];

// Mock data
export const rows: PermissionSet[] = [
    {
        id: '0',
        name: 'GuestAccount',
        displayName: 'GuestAccount',
        description: 'Limited write access to basic settings, cannot save changes',
        type: 'User defined',
        minimumAccessLevel: 'READ_ACCESS',
        permissions: [],
    },
    {
        id: '1',
        name: 'ReadWriteAll',
        displayName: 'ReadWriteAll',
        description: 'Full read and write access',
        type: 'System default',
        minimumAccessLevel: 'READ_WRITE_ACCESS',
        permissions: [],
    },
    {
        id: '2',
        name: 'WriteSpecific',
        displayName: 'WriteSpecific',
        description: 'Limited write access and full read access',
        type: 'System default',
        minimumAccessLevel: 'READ_ACCESS',
        permissions: [],
    },
    {
        id: '3',
        name: 'NoPermissions',
        displayName: 'NoPermissions',
        description: 'No read or write access',
        type: 'System default',
        minimumAccessLevel: 'NO_ACCESS',
        permissions: [],
    },
    {
        id: '4',
        name: 'ReadOnly',
        displayName: 'ReadOnly',
        description: 'Full read access, no write access',
        type: 'System default',
        minimumAccessLevel: 'READ_ACCESS',
        permissions: [],
    },
    {
        id: '5',
        name: 'TestSet',
        displayName: 'TestSet',
        description: 'Experimental set, do not use',
        type: 'User defined',
        minimumAccessLevel: 'NO_ACCESS',
        permissions: [],
    },
];

function AuthProvidersList(): ReactElement {
    const { entityId } = useParams();
    const [selectedRowId, setSelectedRowId] = useState<string | undefined>(entityId);

    // TODO request data

    return (
        <AccessControlListPage
            columns={columns}
            entityType="PERMISSION_SET"
            rows={rows}
            selectedRowId={selectedRowId}
            setSelectedRowId={setSelectedRowId}
        >
            <div className="flex h-full items-center justify-center">
                <code>
                    {JSON.stringify(
                        rows.find(({ id }) => id === selectedRowId),
                        null,
                        2
                    )}
                </code>
            </div>
        </AccessControlListPage>
    );
}

export default AuthProvidersList;
