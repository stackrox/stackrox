import React from 'react';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import { resourceTypes } from 'constants/entityTypes';
import { sortVersion } from 'sorters/sorters';
import LabelChip from 'Components/LabelChip';

const getNameCell = name => <div data-test-id="table-row-name">{name}</div>;

const controlColumns = [
    {
        accessor: 'id',
        Header: 'id',
        headerClassName: 'hidden',
        className: 'hidden'
    },
    {
        accessor: 'control.standardId',
        sortMethod: sortVersion,
        Header: 'Standard',
        headerClassName: `w-1/4 ${defaultHeaderClassName}`,
        className: `w-1/4 ${defaultColumnClassName}`
    },
    {
        accessor: 'control',
        sortMethod: sortVersion,
        Header: 'Control',
        headerClassName: `w-1/4 ${defaultHeaderClassName}`,
        className: `w-1/4 ${defaultColumnClassName}`,
        Cell: ({ original }) =>
            getNameCell(`${original.control.name} - ${original.control.description}`)
    },
    {
        accessor: 'value.overallState',
        Header: 'State',
        headerClassName: `w-1/4 ${defaultHeaderClassName}`,
        className: `w-1/4 ${defaultColumnClassName}`,
        // eslint-disable-next-line react/prop-types
        Cell: ({ original }) => {
            const text = original.value.overallState === 'COMPLIANCE_STATE_FAILURE' && 'Fail';
            return <LabelChip text={text} type="alert" />;
        }
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
        }
    }
];

const nodesAcrossControlsColumns = [
    {
        Header: 'Id',
        headerClassName: 'hidden',
        className: 'hidden',
        accessor: 'id'
    },
    {
        Header: `Node`,
        headerClassName: `w-1/3 ${defaultHeaderClassName}`,
        className: `w-1/3 ${defaultColumnClassName}`,
        accessor: 'name'
    },
    {
        Header: `Cluster`,
        headerClassName: `w-1/3 ${defaultHeaderClassName}`,
        className: `w-1/3 ${defaultColumnClassName}`,
        accessor: 'clusterName'
    },
    {
        Header: `Control Status`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        // eslint-disable-next-line
        Cell: ({ original }) => {
            return !original.passing ? <LabelChip text="Fail" type="alert" /> : 'Pass';
        }
    }
];

export const entityToColumns = {
    [resourceTypes.CONTROL]: controlColumns
};

export const entityAcrossControlsColumns = {
    [resourceTypes.NODE]: nodesAcrossControlsColumns
};
