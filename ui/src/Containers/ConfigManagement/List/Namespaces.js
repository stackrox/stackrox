import React from 'react';
import entityTypes from 'constants/entityTypes';
import URLService from 'modules/URLService';

import { sortValueByLength } from 'sorters/sorters';
import { NAMESPACES_QUERY as QUERY } from 'queries/namespace';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import { entityListPropTypes, entityListDefaultprops } from 'constants/entityPageProps';
import queryService from 'modules/queryService';

import LabelChip from 'Components/LabelChip';
import List from './List';
import TableCellLink from './Link';

const buildTableColumns = (match, location) => {
    const tableColumns = [
        {
            Header: 'Id',
            headerClassName: 'hidden',
            className: 'hidden',
            accessor: 'metadata.id'
        },
        {
            Header: `Namespace`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'metadata.name'
        },
        {
            Header: `Cluster`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'metadata.clusterName',
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { metadata } = original;
                if (!metadata) return '-';
                const { clusterName, clusterId, id } = metadata;
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.CLUSTER, clusterId)
                    .url();
                return <TableCellLink pdf={pdf} url={url} text={clusterName} />;
            }
        },
        {
            Header: `Policies`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { policyCount, metadata } = original;
                if (!policyCount || policyCount === 0) return 'No matches';
                const { id } = metadata;
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.POLICY)
                    .url();
                return <TableCellLink pdf={pdf} url={url} text={`${policyCount} matches`} />;
            },
            accessor: 'policyCount'
        },
        {
            Header: `Policy Status`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original }) => {
                const { policyStatus } = original;
                return policyStatus ? 'Pass' : <LabelChip text="Fail" type="alert" />;
            },
            accessor: 'policyStatus'
        },
        {
            Header: `Secrets`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { numSecrets, metadata } = original;
                if (!metadata || numSecrets === 0) return 'No matches';
                const { id } = metadata;
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.SECRET)
                    .url();
                return <TableCellLink pdf={pdf} url={url} text={`${numSecrets} matches`} />;
            },
            id: 'numSecrets',
            accessor: d => d.numSecrets,
            sortMethod: sortValueByLength
        },
        {
            Header: `Users & Groups`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { subjectsCount, metadata } = original;
                if (!subjectsCount || subjectsCount === 0) return 'No matches';
                const { id } = metadata;
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.SUBJECT)
                    .url();
                return <TableCellLink pdf={pdf} url={url} text={`${subjectsCount} matches`} />;
            },
            accessor: 'subjectCount'
        },
        {
            Header: `Service Accounts`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { serviceAccountCount, metadata } = original;
                if (!serviceAccountCount || serviceAccountCount === 0) return 'No matches';
                const { id } = metadata;
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.SERVICE_ACCOUNT)
                    .url();
                return (
                    <TableCellLink pdf={pdf} url={url} text={`${serviceAccountCount} matches`} />
                );
            },
            accessor: 'serviceAccountCount'
        },
        {
            Header: `Roles granted`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { k8sroleCount, metadata } = original;
                if (!k8sroleCount || k8sroleCount === 0) return 'No matches';
                const { id } = metadata;
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.ROLE)
                    .url();
                return <TableCellLink pdf={pdf} url={url} text={`${k8sroleCount} matches`} />;
            },
            accessor: 'k8sroleCount'
        }
    ];
    return tableColumns;
};

const createTableRows = data => data.results;

const Namespaces = ({ match, location, className, selectedRowId, onRowClick, query }) => {
    const tableColumns = buildTableColumns(match, location);
    const queryText = queryService.objectToWhereClause(query);
    const variables = queryText ? { query: queryText } : null;
    return (
        <List
            className={className}
            query={QUERY}
            variables={variables}
            entityType={entityTypes.NAMESPACE}
            tableColumns={tableColumns}
            createTableRows={createTableRows}
            onRowClick={onRowClick}
            selectedRowId={selectedRowId}
            idAttribute="metadata.id"
        />
    );
};
Namespaces.propTypes = entityListPropTypes;
Namespaces.defaultProps = entityListDefaultprops;

export default Namespaces;
