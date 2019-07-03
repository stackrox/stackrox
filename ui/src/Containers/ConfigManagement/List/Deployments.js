import React from 'react';
import entityTypes from 'constants/entityTypes';
import { DEPLOYMENTS_QUERY as QUERY } from 'queries/deployment';
import URLService from 'modules/URLService';
import { entityListPropTypes, entityListDefaultprops } from 'constants/entityPageProps';

import { sortValueByLength } from 'sorters/sorters';
import queryService from 'modules/queryService';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import LabelChip from 'Components/LabelChip';
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
            Header: `Deployment`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'name'
        },
        {
            Header: `Cluster`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'clusterName'
        },
        {
            Header: `Namespace`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'namespace'
        },
        {
            Header: `Service Account`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'serviceAccount',
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { serviceAccount, id } = original;
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.SERVICE_ACCOUNT)
                    .url();
                return <TableCellLink pdf={pdf} url={url} text={serviceAccount} />;
            }
        },
        {
            Header: `Policies Violated`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'alertsCount',
            // eslint-disable-next-line
            Cell: ({ original }) => {
                const { alertsCount } = original;
                if (alertsCount === 0) return 'No alerts';
                return <LabelChip text={`${alertsCount} Alerts`} type="alert" />;
            },
            sortMethod: sortValueByLength
        }
    ];
    return tableColumns;
};

const createTableRows = data => data.results;

const Deployments = ({ match, location, className, selectedRowId, onRowClick, query }) => {
    const tableColumns = buildTableColumns(match, location);
    const queryText = queryService.objectToWhereClause(query);
    const variables = queryText ? { query: queryText } : null;
    return (
        <List
            className={className}
            query={QUERY}
            variables={variables}
            entityType={entityTypes.DEPLOYMENT}
            tableColumns={tableColumns}
            createTableRows={createTableRows}
            onRowClick={onRowClick}
            selectedRowId={selectedRowId}
            idAttribute="id"
        />
    );
};
Deployments.propTypes = entityListPropTypes;
Deployments.defaultProps = entityListDefaultprops;

export default Deployments;
