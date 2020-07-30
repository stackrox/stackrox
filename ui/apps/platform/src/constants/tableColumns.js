import React from 'react';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import {
    resourceTypes,
    standardEntityTypes,
    standardBaseTypes,
    resourceTypeToApplicableStandards,
} from 'constants/entityTypes';
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

function columnsForResourceType(resourceType) {
    return resourceTypeToApplicableStandards[resourceType].map((id) => columnsForStandard[id]);
}

const clusterColumns = [
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
    ...columnsForResourceType(resourceTypes.CLUSTER),
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

const nodeColumns = [
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
    ...columnsForResourceType(resourceTypes.NODE),
    {
        accessor: 'overall.average',
        Header: 'Overall',
    },
];

const namespaceColumns = [
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
    ...columnsForResourceType(resourceTypes.NAMESPACE),
    {
        accessor: 'overall.average',
        Header: 'Overall',
    },
];

const deploymentColumns = [
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
    ...columnsForResourceType(resourceTypes.DEPLOYMENT),
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

const entityTypesToColumns = {
    [resourceTypes.CLUSTER]: clusterColumns,
    [resourceTypes.NODE]: nodeColumns,
    [resourceTypes.NAMESPACE]: namespaceColumns,
    [resourceTypes.DEPLOYMENT]: deploymentColumns,
    [standardEntityTypes.CONTROL]: controlColumns,
};

function filterColumnsByStandardType(columns, excludedStandardTypes) {
    if (!columns || !columns.length) {
        return columns;
    }
    if (!excludedStandardTypes || !excludedStandardTypes.length) {
        return columns;
    }
    return columns.filter(
        (column) => !excludedStandardTypes.find((standardType) => standardType === column.accessor)
    );
}

export function getColumnsByEntity(entityID, excludedStandardTypes) {
    return filterColumnsByStandardType(entityTypesToColumns[entityID], excludedStandardTypes);
}

export function getColumnsByStandard(standardID) {
    return filterColumnsByStandardType(getStandardColumns(standardBaseTypes[standardID]));
}
