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
import { AuthProvider, Column } from '../accessControlTypes';

// The total of column width ratios must be less than or equal to 1.0
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

export default AuthProvidersList;
