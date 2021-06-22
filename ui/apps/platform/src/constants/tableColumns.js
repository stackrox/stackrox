import React from 'react';

import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import { resourceTypes, standardBaseTypes } from 'constants/entityTypes';
import { sortVersion } from 'sorters/sorters';

const getColumnValue = (row, accessor) => (row[accessor] ? row[accessor] : 'N/A');
const getNameCell = (name) => <div data-testid="table-row-name">{name}</div>;

const columnsForStandard = (function getColumnsForStandards() {
    const ret = {};
    Object.entries(standardBaseTypes).forEach(([baseType, columnName]) => {
        ret[baseType] = {
            accessor: baseType,
            Header: columnName,
            Cell: ({ original }) => getColumnValue(original, baseType),
        };
    });
    return ret;
})();

function getColumnForStandard(id) {
    return {
        accessor: id,
        Header: standardBaseTypes[id] || id,
        Cell: ({ original }) => getColumnValue(original, id),
    };
}

const getClusterColumns = (standards) => [
    {
        accessor: 'id',
        Header: 'id',
        headerClassName: 'hidden',
        className: 'hidden',
    },
    {
        accessor: 'name',
        Header: 'Cluster',
        Cell: ({ original }) => getNameCell(original.name),
    },
    ...standards.map(({ id }) => getColumnForStandard(id)),
    {
        accessor: 'overall.average',
        Header: 'Overall',
    },
];

const getStandardColumns = (standard) => [
    {
        accessor: 'id',
        Header: 'id',
        headerClassName: 'hidden',
        className: 'hidden',
    },
    {
        accessor: 'control',
        sortMethod: sortVersion,
        Header: `${standard} Controls`,
        headerClassName: `w-5/6 ${defaultHeaderClassName}`,
        className: `w-5/6 ${defaultColumnClassName}`,
        Cell: ({ original }) => getNameCell(`${original.control} - ${original.description}`),
    },
    {
        accessor: 'compliance',
        Header: 'Compliance',
        headerClassName: `w-1/8 flex justify-end pr-4 ${defaultHeaderClassName}`,
        className: `w-1/8 justify-end pr-4 ${defaultColumnClassName}`,
    },
];

const getNodeColumns = (standards) => [
    {
        accessor: 'id',
        Header: 'id',
        headerClassName: 'hidden',
        className: 'hidden',
    },
    {
        accessor: 'name',
        Header: 'Node',
        Cell: ({ original }) => getNameCell(original.name),
    },
    {
        accessor: 'cluster',
        Header: 'Cluster',
    },
    ...standards.map(({ id }) => columnsForStandard[id]),
    {
        accessor: 'overall.average',
        Header: 'Overall',
    },
];

const getNamespaceColumns = (standards) => [
    {
        accessor: 'id',
        Header: 'id',
        headerClassName: 'hidden',
        className: 'hidden',
    },
    {
        accessor: 'name',
        Header: 'Namespace',
        Cell: ({ original }) => getNameCell(original.name),
    },
    {
        accessor: 'cluster',
        Header: 'Cluster',
    },
    ...standards.map(({ id }) => columnsForStandard[id]),
    {
        accessor: 'overall.average',
        Header: 'Overall',
    },
];

const getDeploymentColumns = (standards) => [
    {
        accessor: 'id',
        Header: 'id',
        headerClassName: 'hidden',
        className: 'hidden',
    },
    {
        accessor: 'name',
        Header: 'Name',
        Cell: ({ original }) => getNameCell(original.name),
    },
    {
        accessor: 'cluster',
        Header: 'Cluster Name',
        Cell: ({ original }) => getNameCell(original.cluster),
    },
    {
        accessor: 'namespace',
        Header: 'Namespace',
        Cell: ({ original }) => getNameCell(original.namespace),
    },
    ...standards.map(({ id }) => columnsForStandard[id]),
    {
        accessor: 'overall.average',
        Header: 'Overall',
    },
];

const controlColumns = [
    {
        accessor: 'id',
        Header: 'id',
        headerClassName: 'hidden',
        className: 'hidden',
    },
    {
        accessor: 'control',
        sortMethod: sortVersion,
        Header: `Control`,
        headerClassName: `w-5/6 ${defaultHeaderClassName}`,
        className: `w-5/6 ${defaultColumnClassName}`,
        Cell: ({ original }) => getNameCell(`${original.control} - ${original.description}`),
    },
    {
        accessor: 'compliance',
        Header: 'Compliance',
        headerClassName: `w-1/8 flex justify-end pr-4 ${defaultHeaderClassName}`,
        className: `w-1/8 justify-end pr-4 ${defaultColumnClassName}`,
    },
];

export function getColumnsByEntity(entityType, standards) {
    const filteredStandards = standards.filter(({ scopes }) => scopes.includes(entityType));
    switch (entityType) {
        case resourceTypes.CLUSTER:
            return getClusterColumns(filteredStandards);
        case resourceTypes.NODE:
            return getNodeColumns(filteredStandards);
        case resourceTypes.NAMESPACE:
            return getNamespaceColumns(filteredStandards);
        case resourceTypes.DEPLOYMENT:
            return getDeploymentColumns(filteredStandards);
        default:
            return controlColumns;
    }
}

export function getColumnsByStandard(standardID) {
    return getStandardColumns(standardBaseTypes[standardID]);
}
