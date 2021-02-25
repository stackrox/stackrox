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
import { Column, PermissionSet } from '../accessControlTypes';

// The total of column width ratios must be less than or equal to 1.0
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

const entityType: AccessControlEntityType = 'PERMISSION_SET';

function PermissionSetsList(): ReactElement {
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

export default PermissionSetsList;
