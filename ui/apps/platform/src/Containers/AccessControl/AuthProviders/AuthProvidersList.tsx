import React, { ReactElement, useState } from 'react';
import { useParams } from 'react-router-dom';

import { defaultColumnClassName, nonSortableHeaderClassName } from 'Components/Table';

import AccessControlListPage from '../AccessControlListPage';
import { AuthProvider, Column } from '../accessControlTypes';

// 4/4 = 1.0
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
        Header: 'Auth Provider',
        accessor: 'authProvider',
        headerClassName: `w-1/4 ${nonSortableHeaderClassName}`,
        className: `w-1/4 ${defaultColumnClassName}`,
        sortable: false,
    },
    {
        Header: 'Minimum access role',
        accessor: 'minimumAccessRole', // TODO link
        headerClassName: `w-1/4 ${nonSortableHeaderClassName}`,
        className: `w-1/4 ${defaultColumnClassName}`,
        sortable: false,
    },
    {
        Header: 'Assigned rules',
        accessor: 'TODO',
        headerClassName: `w-1/4 ${nonSortableHeaderClassName}`,
        className: `w-1/4 ${defaultColumnClassName}`,
        sortable: false,
    },
];

// Mock data
export const rows: AuthProvider[] = [
    {
        id: '0',
        name: 'Read-Only Auth0',
        authProvider: 'Auth0',
        minimumAccessRole: 'Analyst',
    },
    {
        id: '1',
        name: 'Read-Write OpenID',
        authProvider: 'OpenID Connect',
        minimumAccessRole: 'Analyst',
    },
    {
        id: '2',
        name: 'SeriousSAML',
        authProvider: 'SAML 2.0',
        minimumAccessRole: '',
    },
];

function AuthProvidersList(): ReactElement {
    const { entityId } = useParams();
    const [selectedRowId, setSelectedRowId] = useState<string | undefined>(entityId);

    // TODO request data

    return (
        <AccessControlListPage
            columns={columns}
            entityType="AUTH_PROVIDER"
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
