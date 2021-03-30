import React, { ReactElement, useCallback } from 'react';
import { useHistory, useLocation, useParams } from 'react-router-dom';
import pluralize from 'pluralize';

import { PanelNew, PanelBody } from 'Components/Panel';
import SidePanelAbsoluteArea from 'Components/SidePanelAbsoluteArea';
import { defaultColumnClassName, nonSortableHeaderClassName } from 'Components/Table';
import TableCellLink from 'Components/TableCellLink';
import { AccessControlEntityType } from 'constants/entityTypes';
import { accessControlLabels } from 'messages/common';

import { AccessControlSidePanelHead } from '../AccessControlComponents';
import AccessControlPage from '../AccessControlPage';
import { getEntityPath, getQueryObject } from '../accessControlPaths';
import { AccessScope, Column, accessScopes, roles } from '../accessControlTypes';

import AccessScopeForm from './AccessScopeForm';

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
        Header: 'Resources',
        accessor: 'TODO', // TODO link
        headerClassName: `w-1/5 ${nonSortableHeaderClassName}`,
        className: `w-1/5 ${defaultColumnClassName}`,
        sortable: false,
    },
    {
        Header: 'Roles',
        accessor: 'TODO',
        Cell: ({ original }) => {
            const { id } = original;
            const rolesFiltered = roles.filter(({ accessScopeId }) => accessScopeId === id);

            if (rolesFiltered.length === 0) {
                return 'No roles';
            }

            if (rolesFiltered.length === 1) {
                const role = rolesFiltered[0];
                const url = getEntityPath('ROLE', role.id);
                const text = role.name;
                return <TableCellLink url={url}>{text}</TableCellLink>;
            }

            const url = getEntityPath('ROLE', '', { s: { ACCESS_SCOPE: id } });
            const count = rolesFiltered.length;
            const text = `${count} ${pluralize(accessControlLabels.ROLE, count)}`;
            return <TableCellLink url={url}>{text}</TableCellLink>;
        },
        headerClassName: `w-1/5 ${nonSortableHeaderClassName}`,
        className: `w-1/5 ${defaultColumnClassName}`,
        sortable: false,
    },
];

const accessScopeNew: AccessScope = {
    id: '',
    name: '',
    description: '',
};

const entityType: AccessControlEntityType = 'ACCESS_SCOPE';

function AccessScopesList(): ReactElement {
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

    const accessScope = accessScopes.find(({ id }) => id === entityId) || accessScopeNew;
    const isEditing = Boolean(queryObject.action);
    const isSidePanelOpen = isEditing || Boolean(entityId);

    return (
        <AccessControlPage
            columns={columns}
            entityType={entityType}
            onClickNew={onCreate}
            rows={accessScopes}
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
                            name={accessScope.name}
                            onClickCancel={onCancel}
                            onClickClose={onClose}
                            onClickEdit={onUpdate}
                            onClickSave={onSave}
                        />
                        <PanelBody>
                            <AccessScopeForm accessScope={accessScope} isEditing={isEditing} />
                        </PanelBody>
                    </PanelNew>
                </SidePanelAbsoluteArea>
            )}
        </AccessControlPage>
    );
}

export default AccessScopesList;
