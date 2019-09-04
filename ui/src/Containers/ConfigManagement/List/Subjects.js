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
            Header: `Roles`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { id, roles } = original;
                const { length } = roles;
                if (!length) return 'No Roles';
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.ROLE)
                    .url();
                const text =
                    length === 1
                        ? original.roles[0].name
                        : `${length} ${pluralize('Role', length)}`;
                return <TableCellLink pdf={pdf} url={url} text={text} />;
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
    const autoFocusSearchInput = !selectedRowId;
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
                    id: 'name',
                    desc: false
                }
            ]}
            data={data}
            autoFocusSearchInput={autoFocusSearchInput}
        />
    );
};

Subjects.propTypes = entityListPropTypes;
Subjects.defaultProps = entityListDefaultprops;

export default Subjects;
