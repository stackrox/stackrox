import React from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { IMAGES as QUERY } from 'queries/image';
import { format } from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';

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
        }
    }
];

const createTableRows = data => data.images;

const Images = ({ onRowClick }) => (
    <List
        query={QUERY}
        entityType={entityTypes.IMAGE}
        tableColumns={tableColumns}
        createTableRows={createTableRows}
        onRowClick={onRowClick}
    />
);

Images.propTypes = {
    onRowClick: PropTypes.func.isRequired
};

export default Images;
