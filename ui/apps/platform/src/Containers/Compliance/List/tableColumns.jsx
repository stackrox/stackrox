import React from 'react';
import isEmpty from 'lodash/isEmpty';
import qs from 'qs';

import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import TableCellLink from 'Components/TableCellLink';
import { standardBaseTypes } from 'constants/entityTypes';
import { sortVersion } from 'sorters/sorters';
import { complianceBasePath } from 'routePaths';

const getColumnValue = (row, accessor) => (row[accessor] ? row[accessor] : 'N/A');

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
        Cell: ({ original, pdf }) => {
            const url = `${complianceBasePath}/clusters/${original.id}`;
            return (
                <TableCellLink pdf={pdf} url={url}>
                    {original.name}
                </TableCellLink>
            );
        },
    },
    ...standards.map(({ id }) => getColumnForStandard(id)),
    {
        accessor: 'overall.average',
        Header: 'Overall',
    },
];

// TODO verify that this is obsolete.
// If not obsolete, it might need query argument like getColumnsForControl function.
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
        Cell: ({ original, pdf }) => {
            const url = `${complianceBasePath}/controls/${original.id}`;
            return (
                <TableCellLink pdf={pdf} url={url}>
                    {`${original.control} - ${original.description}`}
                </TableCellLink>
            );
        },
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
        Cell: ({ original, pdf }) => {
            const url = `${complianceBasePath}/nodes/${original.id}`;
            return (
                <TableCellLink pdf={pdf} url={url}>
                    {original.name}
                </TableCellLink>
            );
        },
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
        Cell: ({ original, pdf }) => {
            const url = `${complianceBasePath}/namespaces/${original.id}`;
            return (
                <TableCellLink pdf={pdf} url={url}>
                    {original.name}
                </TableCellLink>
            );
        },
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
        Cell: ({ original, pdf }) => {
            const url = `${complianceBasePath}/deployments/${original.id}`;
            return (
                <TableCellLink pdf={pdf} url={url}>
                    {original.name}
                </TableCellLink>
            );
        },
    },
    {
        accessor: 'cluster',
        Header: 'Cluster Name',
    },
    {
        accessor: 'namespace',
        Header: 'Namespace',
    },
    ...standards.map(({ id }) => columnsForStandard[id]),
    {
        accessor: 'overall.average',
        Header: 'Overall',
    },
];

export function getColumnsForControl(query) {
    return [
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
            Cell: ({ original, pdf }) => {
                const search = isEmpty(query)
                    ? ''
                    : qs.stringify(query, {
                          addQueryPrefix: true,
                          arrayFormat: 'indices',
                          encodeValuesOnly: true,
                      });
                const url = `${complianceBasePath}/controls/${original.id}${search}`;
                return (
                    <TableCellLink pdf={pdf} url={url}>
                        {`${original.control} - ${original.description}`}
                    </TableCellLink>
                );
            },
        },
        {
            accessor: 'compliance',
            Header: 'Compliance',
            headerClassName: `w-1/8 flex justify-end pr-4 ${defaultHeaderClassName}`,
            className: `w-1/8 justify-end pr-4 ${defaultColumnClassName}`,
        },
    ];
}

export function getColumnsByEntity(entityType, standards) {
    const filteredStandards = standards.filter(({ scopes }) => scopes.includes(entityType));
    switch (entityType) {
        case 'CLUSTER':
            return getClusterColumns(filteredStandards);
        case 'DEPLOYMENT':
            return getDeploymentColumns(filteredStandards);
        case 'NAMESPACE':
            return getNamespaceColumns(filteredStandards);
        case 'NODE':
            return getNodeColumns(filteredStandards);
        default:
            return [];
    }
}

// TODO verify that this is obsolete.
export function getColumnsByStandard(standardID) {
    return getStandardColumns(standardBaseTypes[standardID]);
}
