import React from 'react';
import entityTypes from 'constants/entityTypes';
import { SUBJECTS_QUERY } from 'queries/subject';
import URLService from 'modules/URLService';
import { entityListPropTypes, entityListDefaultprops } from 'constants/entityPageProps';
import PermissionCounts from 'Containers/ConfigManagement/Entity/widgets/PermissionCounts';

import { sortValueByLength } from 'sorters/sorters';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import queryService from 'modules/queryService';
import pluralize from 'pluralize';
import List from './List';
import TableCellLink from './Link';

const buildTableColumns = (match, location) => {
    const tableColumns = [
        {
            Header: 'Id',
            headerClassName: 'hidden',
            className: 'hidden',
            accessor: 'id'
        },
        {
            Header: 'Users & Groups',
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            accessor: 'subject.name'
        },
        {
            Header: 'Type',
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            accessor: 'type'
        },
        {
            Header: `Permissions`,
            headerClassName: `w-1/3 ${defaultHeaderClassName}`,
            className: `w-1/3 text-sm ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original }) => {
                const { scopedPermissions } = original;
                return <PermissionCounts scopedPermissions={scopedPermissions} />;
            },
            id: 'permissions',
            accessor: 'scopedPermissions[0].permissions',
            sortMethod: sortValueByLength
        },
        {
            Header: `Cluster Admin Role`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { clusterAdmin } = original;
                return clusterAdmin ? 'Enabled' : 'Disabled';
            },
            accessor: 'clusterAdmin'
        },
        {
            Header: `Permissions Scope`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { scopedPermissions } = original;
                if (!scopedPermissions.length) return 'No Permissions';
                const result = scopedPermissions
                    .map(({ scope, permissions }) => `${scope} (${permissions.length})`)
                    .join(', ');
                return result;
            },
            id: 'permissionsScope',
            accessor: 'scopedPermissions[0].permissions',
            sortMethod: sortValueByLength
        },
        {
            Header: `Roles`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { id, roles } = original;
                const { length } = roles;
                if (!length) return 'No Matches';
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.ROLE)
                    .url();
                if (length > 1)
                    return (
                        <TableCellLink
                            pdf={pdf}
                            url={url}
                            text={`${length} ${pluralize('Roles', length)}`}
                        />
                    );
                return original.roles[0].name;
            },
            accessor: 'roles',
            sortMethod: sortValueByLength
        }
    ];
    return tableColumns;
};

const createTableRows = data =>
    data.subjects.reduce((acc, curr) => [...acc, ...curr.subjectWithClusterID], []);

const Subjects = ({ match, location, selectedRowId, onRowClick, query, className, data }) => {
    const tableColumns = buildTableColumns(match, location);
    const queryText = queryService.objectToWhereClause(query);
    const variables = queryText ? { query: queryText } : null;
    return (
        <List
            className={className}
            query={SUBJECTS_QUERY}
            variables={variables}
            entityType={entityTypes.SUBJECT}
            tableColumns={tableColumns}
            createTableRows={createTableRows}
            selectedRowId={selectedRowId}
            onRowClick={onRowClick}
            idAttribute="id"
            defaultSorted={[
                {
                    id: 'clusterAdmin',
                    desc: true
                },
                {
                    id: 'scopedPermissions[0].permissions',
                    desc: true
                }
            ]}
            data={data}
        />
    );
};

Subjects.propTypes = entityListPropTypes;
Subjects.defaultProps = entityListDefaultprops;

export default Subjects;
