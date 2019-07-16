import React from 'react';
import entityTypes from 'constants/entityTypes';
import { SECRETS as QUERY } from 'queries/secret';
import uniq from 'lodash/uniq';
import { format } from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';
import URLService from 'modules/URLService';
import { sortValueByLength, sortDate } from 'sorters/sorters';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import { entityListPropTypes, entityListDefaultprops } from 'constants/entityPageProps';
import queryService from 'modules/queryService';
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
            Header: `Secret`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'name'
        },
        {
            Header: `Created`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { createdAt } = original;
                return format(createdAt, dateTimeFormat);
            },
            accessor: 'createdAt',
            sortMethod: sortDate
        },
        {
            Header: `File Types`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'files',
            // eslint-disable-next-line
            Cell: ({ original }) => {
                const { files } = original;
                if (!files.length) return 'No File Types';
                return (
                    <span className="capitalize">
                        {uniq(files.map(file => file.type))
                            .join(', ')
                            .replace(/_/g, ' ')
                            .toLowerCase()}
                    </span>
                );
            },
            sortMethod: sortValueByLength
        },
        {
            Header: `Namespace`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'namespace',
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { namespace, id } = original;
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.NAMESPACE)
                    .url();
                return <TableCellLink pdf={pdf} url={url} text={namespace} />;
            }
        },
        {
            Header: `Deployments`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'deployments',
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { deployments, id } = original;
                if (!deployments.length) return 'No Deployments';
                if (deployments.length === 1) return deployments[0].name;
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.DEPLOYMENT)
                    .url();
                return <TableCellLink pdf={pdf} url={url} text={`${deployments.length} matches`} />;
            },
            sortMethod: sortValueByLength
        }
    ];
    return tableColumns;
};

const createTableRows = data => data.secrets;

const Secrets = ({ match, location, className, selectedRowId, onRowClick, query }) => {
    const tableColumns = buildTableColumns(match, location);
    const queryText = queryService.objectToWhereClause(query);
    const variables = queryText ? { query: queryText } : null;
    return (
        <List
            className={className}
            query={QUERY}
            variables={variables}
            entityType={entityTypes.SECRET}
            tableColumns={tableColumns}
            createTableRows={createTableRows}
            onRowClick={onRowClick}
            selectedRowId={selectedRowId}
            idAttribute="id"
            defaultSorted={[
                {
                    id: 'deployments',
                    desc: true
                }
            ]}
        />
    );
};
Secrets.propTypes = entityListPropTypes;
Secrets.defaultProps = entityListDefaultprops;

export default Secrets;
