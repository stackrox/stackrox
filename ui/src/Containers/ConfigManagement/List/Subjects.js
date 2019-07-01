import React from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import QUERY from 'queries/subject';
import URLService from 'modules/URLService';

import { sortValueByLength } from 'sorters/sorters';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
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
                    return <TableCellLink pdf={pdf} url={url} text={`${length} Matches`} />;
                return original.roles[0].name;
            },
            accessor: 'roles',
            sortMethod: sortValueByLength
        }
    ];
    return tableColumns;
};

const createTableRows = data => data.clusters.reduce((acc, curr) => [...acc, ...curr.subjects], []);

const Subjects = ({ match, location, onRowClick }) => {
    const tableColumns = buildTableColumns(match, location);
    return (
        <List
            query={QUERY}
            entityType={entityTypes.SUBJECT}
            tableColumns={tableColumns}
            createTableRows={createTableRows}
            onRowClick={onRowClick}
            idAttribute="id"
        />
    );
};

Subjects.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    onRowClick: PropTypes.func.isRequired
};

export default Subjects;
