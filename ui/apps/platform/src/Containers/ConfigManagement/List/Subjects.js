import React from 'react';
import { useLocation } from 'react-router-dom';
import pluralize from 'pluralize';

import {
    defaultHeaderClassName,
    defaultColumnClassName,
    nonSortableHeaderClassName,
} from 'Components/Table';
import TableCellLink from 'Components/TableCellLink';
import { entityListPropTypes, entityListDefaultprops } from 'constants/entityPageProps';
import entityTypes from 'constants/entityTypes';
import { subjectSortFields } from 'constants/sortFields';
import useWorkflowMatch from 'hooks/useWorkflowMatch';
import { SUBJECTS_QUERY } from 'queries/subject';
import queryService from 'utils/queryService';
import URLService from 'utils/URLService';
import { getConfigMgmtPathForEntitiesAndId } from '../entities';
import List from './List';

export const defaultSubjectSort = [
    {
        id: subjectSortFields.SUBJECT,
        desc: false,
    },
];

const buildTableColumns = (match, location) => {
    const tableColumns = [
        {
            Header: 'Id',
            headerClassName: 'hidden',
            className: 'hidden',
            accessor: 'id',
        },
        {
            Header: 'Users & Groups',
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const url = getConfigMgmtPathForEntitiesAndId('SUBJECT', original.id);
                return (
                    <TableCellLink pdf={pdf} url={url}>
                        {original.name}
                    </TableCellLink>
                );
            },
            accessor: 'name',
            id: subjectSortFields.SUBJECT,
            sortField: subjectSortFields.SUBJECT,
        },
        {
            Header: 'Cluster',
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            accessor: 'clusterName',
        },
        {
            Header: 'Type',
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            accessor: 'type',
            id: subjectSortFields.SUBJECT_KIND,
            sortField: subjectSortFields.SUBJECT_KIND,
        },
        {
            Header: `Cluster Admin Role`,
            headerClassName: `w-1/10 ${nonSortableHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { clusterAdmin } = original;
                return clusterAdmin ? 'Enabled' : 'Disabled';
            },
            accessor: 'clusterAdmin',
            sortable: false,
        },
        {
            Header: `Roles`,
            headerClassName: `w-1/10 ${nonSortableHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { id, k8sRoles } = original;
                const { length } = k8sRoles;
                if (!length) {
                    return 'No Roles';
                }
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.ROLE)
                    .url();
                const text =
                    length === 1 ? k8sRoles[0].name : `${length} ${pluralize('Role', length)}`;
                return (
                    <TableCellLink pdf={pdf} url={url}>
                        {text}
                    </TableCellLink>
                );
            },
            accessor: 'k8sRoles',
            sortable: false,
        },
    ];
    return tableColumns;
};

const createTableRows = (data) => data?.results || [];

const Subjects = ({ selectedRowId, onRowClick, query, className, data, totalResults }) => {
    const location = useLocation();
    const match = useWorkflowMatch();
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
            defaultSorted={defaultSubjectSort}
            data={data}
            totalResults={totalResults}
            autoFocusSearchInput={autoFocusSearchInput}
        />
    );
};

Subjects.propTypes = entityListPropTypes;
Subjects.defaultProps = entityListDefaultprops;

export default Subjects;
