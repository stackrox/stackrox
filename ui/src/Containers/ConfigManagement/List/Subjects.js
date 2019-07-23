import React from 'react';
import entityTypes from 'constants/entityTypes';
import { SUBJECTS_QUERY } from 'queries/subject';
import URLService from 'modules/URLService';
import { entityListPropTypes, entityListDefaultprops } from 'constants/entityPageProps';

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
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'subject.name'
        },
        {
            Header: 'Type',
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'type'
        },
        {
            Header: `Permissions`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { scopedPermissions } = original;
                if (!scopedPermissions.length) return 'No Scoped Permissions';
                const result = scopedPermissions
                    .map(({ scope, permissions }) => `${scope} (${permissions.length})`)
                    .join(', ');
                return result;
            },
            accessor: 'scopedPermissions[0].permissions',
            sortMethod: sortValueByLength
        },
        {
            Header: `Cluster Admin Role`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { clusterAdmin } = original;
                return clusterAdmin ? 'Enabled' : 'Disabled';
            },
            accessor: 'clusterAdmin'
        },
        {
            Header: `Roles`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
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

const Subjects = ({ match, location, selectedRowId, onRowClick, query, className }) => {
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
        />
    );
};

Subjects.propTypes = entityListPropTypes;
Subjects.defaultProps = entityListDefaultprops;

Subjects.defaultProps = {
    className: '',
    selectedRowId: null
};

export default Subjects;
