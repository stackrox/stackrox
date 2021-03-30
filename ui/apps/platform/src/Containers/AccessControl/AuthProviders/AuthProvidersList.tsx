import React, { ReactElement, useCallback } from 'react';
import { useHistory, useLocation, useParams } from 'react-router-dom';

import { PanelNew, PanelBody } from 'Components/Panel';
import SidePanelAbsoluteArea from 'Components/SidePanelAbsoluteArea';
import { defaultColumnClassName, nonSortableHeaderClassName } from 'Components/Table';
import TableCellLink from 'Components/TableCellLink';
import { AccessControlEntityType } from 'constants/entityTypes';

import { AccessControlSidePanelHead } from '../AccessControlComponents';
import AccessControlPage from '../AccessControlPage';
import { getEntityPath, getQueryObject } from '../accessControlPaths';
import { AuthProvider, Column, authProviders, rolesMap } from '../accessControlTypes';

import AuthProviderForm from './AuthProviderForm';

// The total of column width ratios must be less than or equal to 1.0
// 1/5 + 1/5 + 1/5 + 2/5 = 0.2 + 0.2 + 0.2 + 0.4 = 1.0
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
        Header: 'Auth Provider',
        accessor: 'authProvider',
        headerClassName: `w-1/5 ${nonSortableHeaderClassName}`,
        className: `w-1/5 ${defaultColumnClassName}`,
        sortable: false,
    },
    {
        Header: 'Minimum access role',
        accessor: 'minimumAccessRole',
        Cell: ({ original }) => {
            const { minimumAccessRole } = original; // TODO verify it is id not name
            const url = getEntityPath('ROLE', minimumAccessRole);
            const text = rolesMap[minimumAccessRole]?.name ?? '';
            return <TableCellLink url={url}>{text}</TableCellLink>;
        },
        headerClassName: `w-1/5 ${nonSortableHeaderClassName}`,
        className: `w-1/5 ${defaultColumnClassName}`,
        sortable: false,
    },
    {
        Header: 'Assigned rules',
        accessor: 'TODO',
        headerClassName: `w-2/5 ${nonSortableHeaderClassName}`,
        className: `w-2/5 ${defaultColumnClassName}`,
        sortable: false,
    },
];

const authProviderNew: AuthProvider = {
    id: '',
    name: '',
    authProvider: '',
    minimumAccessRole: '',
};

const entityType: AccessControlEntityType = 'AUTH_PROVIDER';

function AuthProvidersList(): ReactElement {
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

    const authProvider = authProviders.find(({ id }) => id === entityId) || authProviderNew;
    const isEditing = Boolean(queryObject.action);
    const isSidePanelOpen = isEditing || Boolean(entityId);

    return (
        <AccessControlPage
            columns={columns}
            entityType={entityType}
            onClickNew={onCreate}
            rows={authProviders}
            selectedRowId={entityId}
            setSelectedRowId={setEntityId}
        >
            {isSidePanelOpen && (
                <SidePanelAbsoluteArea>
                    <PanelNew testid="side-panel">
                        <AccessControlSidePanelHead
                            entityType={entityType}
                            isEditable
                            isEditing={isEditing}
                            name={authProvider.name}
                            onClickCancel={onCancel}
                            onClickClose={onClose}
                            onClickEdit={onUpdate}
                            onClickSave={onSave}
                        />
                        <PanelBody>
                            <AuthProviderForm authProvider={authProvider} isEditing={isEditing} />
                        </PanelBody>
                    </PanelNew>
                </SidePanelAbsoluteArea>
            )}
        </AccessControlPage>
    );
}

export default AuthProvidersList;
