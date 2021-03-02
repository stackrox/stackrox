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
import { Column, authProviders, rolesMap } from '../accessControlTypes';

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
            return (
                <TableCellLink url={getEntityPath('ROLE', minimumAccessRole)}>
                    {rolesMap[minimumAccessRole]?.name ?? ''}
                </TableCellLink>
            );
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

const entityType: AccessControlEntityType = 'AUTH_PROVIDER';

function AuthProvidersList(): ReactElement {
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
    const authProvider = authProviders.find(({ id }) => id === entityId);

    function onClose() {
        setEntityId(undefined);
    }

    const borderColor = getSidePanelHeadBorderColor(isDarkMode);
    return (
        <AccessControlListPage
            columns={columns}
            entityType={entityType}
            isDarkMode={isDarkMode}
            rows={authProviders}
            selectedRowId={entityId}
            setSelectedRowId={setEntityId}
        >
            <PanelNew testid="side-panel">
                <PanelHead isDarkMode={isDarkMode} isSidePanel>
                    <PanelTitle2
                        entityName={authProvider?.name ?? ''}
                        entityTypeLabel={accessControlLabels[entityType]}
                    />
                    <PanelHeadEnd>
                        <CloseButton onClose={onClose} className={`${borderColor} border-l`} />
                    </PanelHeadEnd>
                </PanelHead>
                <PanelBody>
                    <code>{JSON.stringify(authProvider, null, 2)}</code>
                </PanelBody>
            </PanelNew>
        </AccessControlListPage>
    );
}

export default AuthProvidersList;
