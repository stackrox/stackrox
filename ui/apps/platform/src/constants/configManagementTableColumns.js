import React from 'react';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import { resourceTypes } from 'constants/entityTypes';
import { sortVersion } from 'sorters/sorters';

const getNameCell = (name) => <div data-testid="table-row-name">{name}</div>;

const controlColumns = [
    {
        accessor: 'id',
        Header: 'id',
        headerClassName: 'hidden',
        className: 'hidden',
    },
    {
        accessor: 'control.standardId',
        sortMethod: sortVersion,
        Header: 'Standard',
        headerClassName: `w-1/4 ${defaultHeaderClassName}`,
        className: `w-1/4 ${defaultColumnClassName}`,
    },
    {
        accessor: 'control',
        sortMethod: sortVersion,
        Header: 'Control',
        headerClassName: `w-1/4 ${defaultHeaderClassName}`,
        className: `w-1/4 ${defaultColumnClassName}`,
        Cell: ({ original }) =>
            getNameCell(`${original.control.name} - ${original.control.description}`),
    },
    {
        accessor: 'value.overallState',
        Header: 'State',
        headerClassName: `w-1/4 ${defaultHeaderClassName}`,
        className: `w-1/4 ${defaultColumnClassName}`,
        // eslint-disable-next-line react/prop-types
        Cell: ({ original }) => (
            <span className="bg-alert-200 border border-alert-400 px-2 rounded text-alert-800">
                {original.value.overallState === 'COMPLIANCE_STATE_FAILURE' && 'Fail'}
            </span>
        ),
    },
    {
        accessor: 'value.evidence',
        Header: 'Evidence',
        headerClassName: `w-1/4 ${defaultHeaderClassName}`,
        className: `w-1/4 ${defaultColumnClassName}`,
        Cell: ({ original }) => {
            const { evidence } = original.value;
            return `${evidence[0].message}  ${
                evidence.length > 1 ? `+${evidence.length - 1} more...` : ''
            }`;
        },
    },
];

const entityToColumns = {
    [resourceTypes.CONTROL]: controlColumns,
};

export default entityToColumns;
