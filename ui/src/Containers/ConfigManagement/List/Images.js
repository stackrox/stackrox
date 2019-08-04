import React from 'react';
import entityTypes from 'constants/entityTypes';
import { IMAGES as QUERY } from 'queries/image';
import { format } from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';
import { sortDate } from 'sorters/sorters';
import { entityListPropTypes, entityListDefaultprops } from 'constants/entityPageProps';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import queryService from 'modules/queryService';
import pluralize from 'pluralize';
import URLService from 'modules/URLService';
import Query from 'Components/ThrowingQuery';
import { DEPLOYMENT_NAME } from 'queries/deployment';
import TableCellLink from './Link';
import List from './List';

const buildTableColumns = (match, location) => [
    {
        Header: 'Id',
        headerClassName: 'hidden',
        className: 'hidden',
        accessor: 'id'
    },
    {
        Header: `Image`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'name.fullName'
    },
    {
        Header: `Created`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        Cell: ({ original }) => {
            const { metadata } = original;
            if (!metadata) return null;
            return format(metadata.v1.created, dateTimeFormat);
        },
        accessor: 'metadata.v1.created',
        sortMethod: sortDate
    },
    {
        Header: `Deployments`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        // eslint-disable-next-line
        Cell: ({ original, pdf }) => {
            const { deploymentIDs, id } = original;
            const num = deploymentIDs.length;
            const text = `${num} ${pluralize('deployment', num)}`;
            if (num === 0) return text;

            if (num === 1 && !pdf) {
                const deploymentId = deploymentIDs[0];
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.DEPLOYMENT, deploymentId)
                    .url();
                return (
                    <Query query={DEPLOYMENT_NAME} variables={{ id: deploymentId }}>
                        {({ loading, data }) => {
                            if (loading || !data || !data.result)
                                return <TableCellLink pdf={pdf} url={url} text={text} />;
                            return <TableCellLink pdf={pdf} url={url} text={data.result.name} />;
                        }}
                    </Query>
                );
            }

            const url = URLService.getURL(match, location)
                .push(id)
                .push(entityTypes.DEPLOYMENT)
                .url();
            return <TableCellLink pdf={pdf} url={url} text={text} />;
        },
        accessor: 'deploymentIDs'
    }
];

const createTableRows = data => data.images;

const Images = ({ className, selectedRowId, onRowClick, query, match, location, data }) => {
    const queryText = queryService.objectToWhereClause(query);
    const variables = queryText ? { query: queryText } : null;
    const tableColumns = buildTableColumns(match, location);
    return (
        <List
            className={className}
            query={QUERY}
            variables={variables}
            entityType={entityTypes.IMAGE}
            tableColumns={tableColumns}
            createTableRows={createTableRows}
            onRowClick={onRowClick}
            selectedRowId={selectedRowId}
            idAttribute="id"
            data={data}
        />
    );
};
Images.propTypes = entityListPropTypes;
Images.defaultProps = entityListDefaultprops;

export default Images;
