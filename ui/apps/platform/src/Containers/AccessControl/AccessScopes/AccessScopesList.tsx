import React, { ReactElement, useState } from 'react';
import { useParams } from 'react-router-dom';

import { defaultColumnClassName, nonSortableHeaderClassName } from 'Components/Table';

import AccessControlListPage from '../AccessControlListPage';
import { AccessScope, Column } from '../accessControlTypes';

// 3/6 + 2/5 + 1/10 = 0.5 + 0.4 + 0.1 = 1.0
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
        Header: 'Resources',
        accessor: 'TODO', // TODO link
        headerClassName: `w-1/6 ${nonSortableHeaderClassName}`,
        className: `w-1/6 ${defaultColumnClassName}`,
        sortable: false,
    },
    {
        Header: 'Roles',
        accessor: 'TODO', // TODO link
        headerClassName: `w-1/6 ${nonSortableHeaderClassName}`,
        className: `w-1/6 ${defaultColumnClassName}`,
        sortable: false,
    },
];

// Mock data
export const rows: AccessScope[] = [
    {
        id: '0',
        name: 'WalledGarden',
        description: 'Exclude all entities, only access select entities',
        type: 'User defined',
    },
    {
        id: '1',
        name: 'AllAccess',
        description: 'Users can access all entities',
        type: 'System default',
    },
    {
        id: '2',
        name: 'LimitedAccess',
        description: 'Users have access to limited entities',
        type: 'System default',
    },
    {
        id: '3',
        name: 'DenyAccess',
        description: 'Users have no access',
        type: 'System default',
    },
];

function AuthProvidersList(): ReactElement {
    const { entityId } = useParams();
    const [selectedRowId, setSelectedRowId] = useState<string | undefined>(entityId);

    // TODO request data

    return (
        <AccessControlListPage
            columns={columns}
            entityType="ACCESS_SCOPE"
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
