import React from 'react';
import entityTypes from 'constants/entityTypes';
import { IMAGES as QUERY } from 'queries/image';
import { format } from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';

import { sortDate } from 'sorters/sorters';
import { entityListPropTypes, entityListDefaultprops } from 'constants/entityPageProps';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import queryService from 'modules/queryService';
import List from './List';

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
            if (!metadata) return null;
            return format(metadata.v1.created, dateTimeFormat);
        },
        accessor: 'metadata.v1.created',
        sortMethod: sortDate
    }
];

const createTableRows = data => data.images;

const Images = ({ className, selectedRowId, onRowClick, query, data }) => {
    const queryText = queryService.objectToWhereClause(query);
    const variables = queryText ? { query: queryText } : null;
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
