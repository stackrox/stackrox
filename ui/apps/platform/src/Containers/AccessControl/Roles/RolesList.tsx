import React, { ReactElement, useState } from 'react';
import { useParams } from 'react-router-dom';

import { defaultColumnClassName, nonSortableHeaderClassName } from 'Components/Table';

import AccessControlListPage from '../AccessControlListPage';
import { Column, Role } from '../accessControlTypes';

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
        headerClassName: `w-1/6 ${nonSortableHeaderClassName}`,
        className: `w-1/6 ${defaultColumnClassName}`,
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
        Header: 'Permission set',
        accessor: 'permissionSetName', // TODO link
        headerClassName: `w-1/6 ${nonSortableHeaderClassName}`,
        className: `w-1/6 ${defaultColumnClassName}`,
        sortable: false,
    },
    {
        Header: 'Access scope',
        accessor: 'accessScopeName', // TODO link
        headerClassName: `w-1/6 ${nonSortableHeaderClassName}`,
        className: `w-1/6 ${defaultColumnClassName}`,
        sortable: false,
    },
];

// Mock data
export const rows: Role[] = [
    {
        id: '0',
        name: 'GuestUser',
        displayName: 'GuestUser',
        description: 'Has access to selected entities',
        type: 'User defined',
        permissionSetName: 'GuestAccount',
        accessScopeName: 'WalledGarden',
    },
    {
        id: '1',
        name: 'Admin',
        displayName: 'Admin',
        description: 'Admin access to all entities',
        type: 'System default',
        permissionSetName: 'ReadWriteAll',
        accessScopeName: 'AllAccess',
    },
    {
        id: '2',
        name: 'SensorCreator',
        displayName: 'SensorCreator',
        description: 'Users can create sensors',
        type: 'System default',
        permissionSetName: 'WriteSpecific',
        accessScopeName: 'LimitedAccess',
    },
    {
        id: '3',
        name: 'NoAccess',
        displayName: 'NoAccess',
        description: 'No access',
        type: 'System default',
        permissionSetName: 'NoPermissions',
        accessScopeName: 'DenyAccess',
    },
    {
        id: '4',
        name: 'ContinuousIntegration',
        displayName: 'ContinuousIntegration',
        description: 'Users can manage integrations',
        type: 'System default',
        permissionSetName: 'WriteSpecific',
        accessScopeName: 'LimitedAccess',
    },
    {
        id: '5',
        name: 'Analyst',
        displayName: 'Analyst',
        description: 'Users can view and create reports',
        type: 'System default',
        permissionSetName: 'ReadOnly',
        accessScopeName: 'Limited Access',
    },
];

function AuthProvidersList(): ReactElement {
    const { entityId } = useParams();
    const [selectedRowId, setSelectedRowId] = useState<string | undefined>(entityId);

    // TODO request data

    return (
        <AccessControlListPage
            columns={columns}
            entityType="ROLE"
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
