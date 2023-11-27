import React from 'react';
import pluralize from 'pluralize';
import { format } from 'date-fns';

import {
    defaultHeaderClassName,
    defaultColumnClassName,
    nonSortableHeaderClassName,
} from 'Components/Table';
import TableCellLink from 'Components/TableCellLink';
import dateTimeFormat from 'constants/dateTimeFormat';
import { entityListPropTypes, entityListDefaultprops } from 'constants/entityPageProps';
import entityTypes from 'constants/entityTypes';
import { imageSortFields } from 'constants/sortFields';
import { IMAGES_QUERY } from 'queries/image';
import queryService from 'utils/queryService';
import URLService from 'utils/URLService';
import List from './List';

export const defaultImageSort = [
    {
        id: imageSortFields.NAME,
        desc: false,
    },
];

const buildTableColumns = (match, location, entityContext) => {
    const tableColumns = [
        {
            Header: 'Id',
            headerClassName: 'hidden',
            className: 'hidden',
            accessor: 'id',
        },
        {
            Header: `Image`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'name.fullName',
            id: imageSortFields.NAME,
            sortField: imageSortFields.NAME,
        },
        {
            Header: `Created`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { metadata } = original;
                if (!metadata) {
                    return '-';
                }
                return format(metadata.v1.created, dateTimeFormat);
            },
            id: imageSortFields.CREATED_TIME,
            sortField: imageSortFields.CREATED_TIME,
        },
        entityContext && entityContext[entityTypes.DEPLOYMENT]
            ? null
            : {
                  Header: `Deployments`,
                  headerClassName: `w-1/8 ${nonSortableHeaderClassName}`,
                  className: `w-1/8 ${defaultColumnClassName}`,
                  Cell: ({ original, pdf }) => {
                      const { deployments, id } = original;
                      const num = deployments.length;
                      const text = `${num} ${pluralize('deployment', num)}`;
                      if (num === 0) {
                          return text;
                      }
                      const url = URLService.getURL(match, location)
                          .push(id)
                          .push(entityTypes.DEPLOYMENT)
                          .url();
                      return (
                          <TableCellLink pdf={pdf} url={url}>
                              {text}
                          </TableCellLink>
                      );
                  },
                  accessor: 'deployments',
                  sortable: false,
              },
    ];

    return tableColumns.filter((col) => col);
};

const createTableRows = (data) => data.images;

const Images = ({
    className,
    selectedRowId,
    onRowClick,
    query,
    match,
    location,
    data,
    totalResults,
    entityContext,
}) => {
    const autoFocusSearchInput = !selectedRowId;
    const queryText = queryService.objectToWhereClause(query);
    const variables = queryText ? { query: queryText } : null;
    const tableColumns = buildTableColumns(match, location, entityContext);
    return (
        <List
            className={className}
            query={IMAGES_QUERY}
            variables={variables}
            entityType={entityTypes.IMAGE}
            tableColumns={tableColumns}
            createTableRows={createTableRows}
            onRowClick={onRowClick}
            selectedRowId={selectedRowId}
            idAttribute="id"
            defaultSorted={defaultImageSort}
            data={data}
            totalResults={totalResults}
            autoFocusSearchInput={autoFocusSearchInput}
        />
    );
};
Images.propTypes = entityListPropTypes;
Images.defaultProps = entityListDefaultprops;

export default Images;
