import React, { ReactElement, useCallback } from 'react';
import { useHistory, useParams } from 'react-router-dom';
import pluralize from 'pluralize';

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
import { Column, accessScopes, roles } from '../accessControlTypes';

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
                return (
                    <TableCellLink url={getEntityPath('ROLE', role.id)}>{role.name}</TableCellLink>
                );
            }

            const count = rolesFiltered.length;
            const text = `${count} ${pluralize(accessControlLabels.ROLE, count)}`;
            return (
                <TableCellLink url={getEntityPath('ROLE', '', { ACCESS_SCOPE: id })}>
                    {text}
                </TableCellLink>
            );
        },
        headerClassName: `w-1/5 ${nonSortableHeaderClassName}`,
        className: `w-1/5 ${defaultColumnClassName}`,
        sortable: false,
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
    const accessScope = accessScopes.find(({ id }) => id === entityId);

    function onClose() {
        setEntityId(undefined);
    }

    const borderColor = getSidePanelHeadBorderColor(isDarkMode);
    return (
        <AccessControlListPage
            columns={columns}
            entityType={entityType}
            isDarkMode={isDarkMode}
            rows={accessScopes}
            selectedRowId={entityId}
            setSelectedRowId={setEntityId}
        >
            <PanelNew testid="side-panel">
                <PanelHead isDarkMode={isDarkMode} isSidePanel>
                    <PanelTitle2
                        entityName={accessScope?.name ?? ''}
                        entityTypeLabel={accessControlLabels[entityType]}
                    />
                    <PanelHeadEnd>
                        <CloseButton onClose={onClose} className={`${borderColor} border-l`} />
                    </PanelHeadEnd>
                </PanelHead>
                <PanelBody>
                    <code>{JSON.stringify(accessScope, null, 2)}</code>
                </PanelBody>
            </PanelNew>
        </AccessControlListPage>
    );
}

export default AccessScopesList;
