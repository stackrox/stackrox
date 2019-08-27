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
import TableCellLink from './Link';
import List from './List';

const buildTableColumns = (match, location, entityContext) => {
    const tableColumns = [
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
                if (!metadata) return '-';
                return format(metadata.v1.created, dateTimeFormat);
            },
            sortMethod: sortDate
        },
        entityContext && entityContext[entityTypes.DEPLOYMENT]
            ? null
            : {
                  Header: `Deployments`,
                  headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                  className: `w-1/8 ${defaultColumnClassName}`,
                  // eslint-disable-next-line
        Cell: ({ original, pdf }) => {
                      const { deployments, id } = original;
                      const num = deployments.length;
                      const text = `${num} ${pluralize('deployment', num)}`;
                      if (num === 0) return text;
                      const url = URLService.getURL(match, location)
                          .push(id)
                          .push(entityTypes.DEPLOYMENT)
                          .url();
                      return <TableCellLink pdf={pdf} url={url} text={text} />;
                  },
                  accessor: 'deployments'
              }
    ];

    return tableColumns.filter(col => col);
};

const createTableRows = data => data.images;

const Images = ({
    className,
    selectedRowId,
    onRowClick,
    query,
    match,
    location,
    data,
    entityContext
}) => {
    const queryText = queryService.objectToWhereClause(query);
    const variables = queryText ? { query: queryText } : null;
    const tableColumns = buildTableColumns(match, location, entityContext);
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
