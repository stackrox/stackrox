import React, { ReactElement, useCallback } from 'react';
import { useHistory, useParams } from 'react-router-dom';

import CloseButton from 'Components/CloseButton';
import {
    getSidePanelHeadBorderColor,
    PanelNew,
    PanelBody,
    PanelHead,
    PanelHeadEnd,
    PanelTitle,
} from 'Components/Panel';
import { defaultColumnClassName, nonSortableHeaderClassName } from 'Components/Table';
import { AccessControlEntityType } from 'constants/entityTypes';
import { useTheme } from 'Containers/ThemeProvider';

import AccessControlListPage from '../AccessControlListPage';
import { getEntityPath } from '../accessControlPaths';
import { Column, Role } from '../accessControlTypes';

// The total of column width ratios must be less than or equal to 1.0
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

const entityType: AccessControlEntityType = 'ROLE';

function RolesList(): ReactElement {
    const history = useHistory();
    // const { search } = useLocation();
    const { entityId } = useParams();
    const { isDarkMode } = useTheme();

    const setEntityId = useCallback(
        (id) => {
            const url = getEntityPath(entityType, id);
            history.push(url);
        },
        [history]
    );

    // TODO request data
    const row = rows.find(({ id }) => id === entityId);

    function onClose() {
        setEntityId(undefined);
    }

    const borderColor = getSidePanelHeadBorderColor(isDarkMode);
    return (
        <AccessControlListPage
            columns={columns}
            entityType={entityType}
            isDarkMode={isDarkMode}
            rows={rows}
            selectedRowId={entityId}
            setSelectedRowId={setEntityId}
        >
            <PanelNew testid="side-panel">
                <PanelHead isDarkMode={isDarkMode} isSidePanel>
                    <PanelTitle isUpperCase={false} testid="head-text" text={row?.name ?? ''} />
                    <PanelHeadEnd>
                        <CloseButton onClose={onClose} className={`${borderColor} border-l`} />
                    </PanelHeadEnd>
                </PanelHead>
                <PanelBody>
                    <code>{JSON.stringify(row, null, 2)}</code>
                </PanelBody>
            </PanelNew>
        </AccessControlListPage>
    );
}

export default RolesList;
