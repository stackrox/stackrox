import React from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { IMAGES as QUERY } from 'queries/image';
import { format } from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';

import { sortDate } from 'sorters/sorters';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
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

const Images = ({ className, selectedRowId, onRowClick }) => (
    <List
        className={className}
        query={QUERY}
        entityType={entityTypes.IMAGE}
        tableColumns={tableColumns}
        createTableRows={createTableRows}
        onRowClick={onRowClick}
        selectedRowId={selectedRowId}
        idAttribute="id"
    />
);

Images.propTypes = {
    className: PropTypes.string,
    selectedRowId: PropTypes.string,
    onRowClick: PropTypes.func.isRequired
};

Images.defaultProps = {
    className: '',
    selectedRowId: null
};

export default Images;
