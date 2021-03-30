import React, { ReactElement, useCallback } from 'react';
import { useHistory, useLocation, useParams } from 'react-router-dom';

import { PanelNew, PanelBody } from 'Components/Panel';
import SidePanelAbsoluteArea from 'Components/SidePanelAbsoluteArea';
import { defaultColumnClassName, nonSortableHeaderClassName } from 'Components/Table';
import TableCellLink from 'Components/TableCellLink';
import { defaultRoles } from 'constants/accessControl';
import { AccessControlEntityType } from 'constants/entityTypes';

import { AccessControlSidePanelHead } from '../AccessControlComponents';
import AccessControlPage from '../AccessControlPage';
import { getEntityPath, getQueryObject } from '../accessControlPaths';
import { Column, Role, accessScopesMap, permissionSetsMap, roles } from '../accessControlTypes';

import RoleForm from './RoleForm';

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
            const url = getEntityPath('PERMISSION_SET', permissionSetId);
            const text = permissionSetsMap[permissionSetId]?.name ?? '';
            return <TableCellLink url={url}>{text}</TableCellLink>;
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
            const url = getEntityPath('ACCESS_SCOPE', accessScopeId);
            const text = accessScopesMap[accessScopeId]?.name ?? '';
            return <TableCellLink url={url}>{text}</TableCellLink>;
        },
        headerClassName: `w-1/5 ${nonSortableHeaderClassName}`,
        className: `w-1/5 ${defaultColumnClassName}`,
        sortable: false,
    },
];

const roleNew: Role = {
    id: '',
    name: '',
    description: '',
    permissionSetId: '',
    accessScopeId: '',
};

const entityType: AccessControlEntityType = 'ROLE';

function RolesList(): ReactElement {
    const history = useHistory();
    const { search } = useLocation();
    const { entityId } = useParams();

    const queryObject = getQueryObject(search);

    const setEntityId = useCallback(
        (id) => {
            const url = getEntityPath(entityType, id);
            history.push(url);
        },
        [history]
    );

    // TODO request data

    function onCancel() {
        const url = getEntityPath(entityType, entityId, { ...queryObject, action: undefined });
        history.push(url);
    }

    function onClose() {
        const url = getEntityPath(entityType);
        history.push(url);
    }

    function onCreate() {
        const url = getEntityPath(entityType, undefined, { ...queryObject, action: 'create' });
        history.push(url);
    }

    function onSave() {
        // TODO put change
    }

    function onUpdate() {
        const url = getEntityPath(entityType, entityId, { ...queryObject, action: 'update' });
        history.push(url);
    }

    const role = roles.find(({ id }) => id === entityId) || roleNew;
    const isEditable = !defaultRoles[role.name];
    const isEditing = Boolean(queryObject.action);
    const isSidePanelOpen = isEditing || Boolean(entityId);

    return (
        <AccessControlPage
            columns={columns}
            entityType={entityType}
            onClickNew={onCreate}
            rows={roles}
            selectedRowId={entityId}
            setSelectedRowId={setEntityId}
        >
            {isSidePanelOpen && (
                <SidePanelAbsoluteArea>
                    <PanelNew testid="side-panel">
                        <AccessControlSidePanelHead
                            entityType={entityType}
                            isEditable={isEditable}
                            isEditing={isEditing}
                            name={role.name}
                            onClickCancel={onCancel}
                            onClickClose={onClose}
                            onClickEdit={onUpdate}
                            onClickSave={onSave}
                        />
                        <PanelBody>
                            <RoleForm role={role} isEditing={isEditing} />
                        </PanelBody>
                    </PanelNew>
                </SidePanelAbsoluteArea>
            )}
        </AccessControlPage>
    );
}

export default RolesList;
