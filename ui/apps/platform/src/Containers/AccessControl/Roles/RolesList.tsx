import React, { ReactElement, useCallback } from 'react';
import { useHistory, useParams } from 'react-router-dom';

import CloseButton from 'Components/CloseButton';
import {
    getSidePanelHeadBorderColor,
    PanelNew,
    PanelBody,
    PanelHead,
    PanelHeadEnd,
} from 'Components/Panel';
import { defaultColumnClassName, nonSortableHeaderClassName } from 'Components/Table';
import TableCellLink from 'Components/TableCellLink';
import { AccessControlEntityType } from 'constants/entityTypes';
import { useTheme } from 'Containers/ThemeProvider';
import { accessControlLabels } from 'messages/common';

import { PanelTitle2 } from '../AccessControlComponents';
import AccessControlListPage from '../AccessControlListPage';
import { getEntityPath } from '../accessControlPaths';
import { Column, accessScopesMap, permissionSetsMap, roles } from '../accessControlTypes';

// The total of column width ratios must be less than or equal to 1.0
// 1/5 + 2/5 + 1/5 + 1/5 = 0.2 + 0.4 + 0.2 + 0.2 = 1.0
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
        headerClassName: `w-1/5 ${nonSortableHeaderClassName}`,
        className: `w-1/5 ${defaultColumnClassName}`,
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
        Header: 'Permission set',
        accessor: 'permissionSetId',
        Cell: ({ original }) => {
            const { permissionSetId } = original;
            return (
                <TableCellLink url={getEntityPath('PERMISSION_SET', permissionSetId)}>
                    {permissionSetsMap[permissionSetId]?.name ?? ''}
                </TableCellLink>
            );
        },
        headerClassName: `w-1/5 ${nonSortableHeaderClassName}`,
        className: `w-1/5 ${defaultColumnClassName}`,
        sortable: false,
    },
    {
        Header: 'Access scope',
        accessor: 'accessScopeId',
        Cell: ({ original }) => {
            const { accessScopeId } = original;
            return (
                <TableCellLink url={getEntityPath('ACCESS_SCOPE', accessScopeId)}>
                    {accessScopesMap[accessScopeId]?.name ?? ''}
                </TableCellLink>
            );
        },
        headerClassName: `w-1/5 ${nonSortableHeaderClassName}`,
        className: `w-1/5 ${defaultColumnClassName}`,
        sortable: false,
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
    const role = roles.find(({ id }) => id === entityId);

    function onClose() {
        setEntityId(undefined);
    }

    const borderColor = getSidePanelHeadBorderColor(isDarkMode);
    return (
        <AccessControlListPage
            columns={columns}
            entityType={entityType}
            isDarkMode={isDarkMode}
            rows={roles}
            selectedRowId={entityId}
            setSelectedRowId={setEntityId}
        >
            <PanelNew testid="side-panel">
                <PanelHead isDarkMode={isDarkMode} isSidePanel>
                    <PanelTitle2
                        entityName={role?.name ?? ''}
                        entityTypeLabel={accessControlLabels[entityType]}
                    />
                    <PanelHeadEnd>
                        <CloseButton onClose={onClose} className={`${borderColor} border-l`} />
                    </PanelHeadEnd>
                </PanelHead>
                <PanelBody>
                    <code>{JSON.stringify(role, null, 2)}</code>
                </PanelBody>
            </PanelNew>
        </AccessControlListPage>
    );
}

export default RolesList;
