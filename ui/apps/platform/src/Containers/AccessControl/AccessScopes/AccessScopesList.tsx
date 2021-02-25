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
import { AccessScope, Column } from '../accessControlTypes';

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

const entityType: AccessControlEntityType = 'ACCESS_SCOPE';

function AccessScopesList(): ReactElement {
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

export default AccessScopesList;
