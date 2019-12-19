import React from 'react';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import {
    standardTypes,
    resourceTypes,
    standardEntityTypes,
    standardBaseTypes
} from 'constants/entityTypes';
import { sortVersion } from 'sorters/sorters';

const getColumnValue = (row, accessor) => (row[accessor] ? row[accessor] : 'N/A');
const getNameCell = name => <div data-test-id="table-row-name">{name}</div>;

// eslint-disable-next-line func-names
const columnsForStandards = (function getColumnsForStandards() {
    const ret = {};
    Object.entries(standardBaseTypes).forEach(([baseType, columnName]) => {
        ret[baseType] = {
            accessor: baseType,
            Header: columnName,
            Cell: ({ original }) => getColumnValue(original, baseType)
        };
    });
    return ret;
})();

const complianceColumns = [
    columnsForStandards[standardTypes.CIS_Docker_v1_2_0],
    columnsForStandards[standardTypes.CIS_Kubernetes_v1_5],
    columnsForStandards[standardTypes.HIPAA_164],
    columnsForStandards[standardTypes.NIST_800_190],
    columnsForStandards[standardTypes.PCI_DSS_3_2]
];

const clusterColumns = [
    {
        accessor: 'id',
        Header: 'id',
        headerClassName: 'hidden',
        className: 'hidden'
    },
    {
        accessor: 'name',
        Header: 'Cluster',
        Cell: ({ original }) => getNameCell(original.name)
    },
    ...complianceColumns,
    {
        accessor: 'overall.average',
        Header: 'Overall'
    }
];

const getStandardColumns = standard => [
    {
        accessor: 'id',
        Header: 'id',
        headerClassName: 'hidden',
        className: 'hidden'
    },
    {
        accessor: 'control',
        sortMethod: sortVersion,
        Header: `${standard} Controls`,
        headerClassName: `w-5/6 ${defaultHeaderClassName}`,
        className: `w-5/6 ${defaultColumnClassName}`,
        Cell: ({ original }) => getNameCell(`${original.control} - ${original.description}`)
    },
    {
        accessor: 'compliance',
        Header: 'Compliance',
        headerClassName: `w-1/8 flex justify-end pr-4 ${defaultHeaderClassName}`,
        className: `w-1/8 justify-end pr-4 ${defaultColumnClassName}`
    }
];

const nodeColumns = [
    {
        accessor: 'id',
        Header: 'id',
        headerClassName: 'hidden',
        className: 'hidden'
    },
    {
        accessor: 'name',
        Header: 'Node',
        Cell: ({ original }) => getNameCell(original.name)
    },
    {
        accessor: 'cluster',
        Header: 'Cluster'
    },
    columnsForStandards[standardTypes.CIS_Docker_v1_2_0],
    columnsForStandards[standardTypes.CIS_Kubernetes_v1_5],
    columnsForStandards[standardTypes.NIST_800_190],
    {
        accessor: 'overall.average',
        Header: 'Overall'
    }
];

const namespaceColumns = [
    {
        accessor: 'id',
        Header: 'id',
        headerClassName: 'hidden',
        className: 'hidden'
    },
    {
        accessor: 'name',
        Header: 'Namespace',
        Cell: ({ original }) => getNameCell(original.name)
    },
    {
        accessor: 'cluster',
        Header: 'Cluster'
    },
    columnsForStandards[standardTypes.HIPAA_164],
    columnsForStandards[standardTypes.NIST_800_190],
    columnsForStandards[standardTypes.PCI_DSS_3_2],
    {
        accessor: 'overall.average',
        Header: 'Overall'
    }
];

const deploymentColumns = [
    {
        accessor: 'id',
        Header: 'id',
        headerClassName: 'hidden',
        className: 'hidden'
    },
    {
        accessor: 'name',
        Header: 'Name',
        Cell: ({ original }) => getNameCell(original.name)
    },
    {
        accessor: 'cluster',
        Header: 'Cluster Name',
        Cell: ({ original }) => getNameCell(original.cluster)
    },
    {
        accessor: 'namespace',
        Header: 'Namespace',
        Cell: ({ original }) => getNameCell(original.namespace)
    },
    columnsForStandards[standardTypes.HIPAA_164],
    columnsForStandards[standardTypes.NIST_800_190],
    columnsForStandards[standardTypes.PCI_DSS_3_2],
    {
        accessor: 'overall.average',
        Header: 'Overall'
    }
];

const controlColumns = [
    {
        accessor: 'id',
        Header: 'id',
        headerClassName: 'hidden',
        className: 'hidden'
    },
    {
        accessor: 'control',
        sortMethod: sortVersion,
        Header: `Control`,
        headerClassName: `w-5/6 ${defaultHeaderClassName}`,
        className: `w-5/6 ${defaultColumnClassName}`,
        Cell: ({ original }) => getNameCell(`${original.control} - ${original.description}`)
    },
    {
        accessor: 'compliance',
        Header: 'Compliance',
        headerClassName: `w-1/8 flex justify-end pr-4 ${defaultHeaderClassName}`,
        className: `w-1/8 justify-end pr-4 ${defaultColumnClassName}`
    }
];

const entityToColumns = (function getEntityToColumns() {
    const ret = {
        [resourceTypes.CLUSTER]: clusterColumns,
        [resourceTypes.NODE]: nodeColumns,
        [resourceTypes.NAMESPACE]: namespaceColumns,
        [resourceTypes.DEPLOYMENT]: deploymentColumns,
        [standardEntityTypes.CONTROL]: controlColumns
    };

    Object.entries(standardBaseTypes).forEach(([baseType, standardName]) => {
        ret[baseType] = getStandardColumns(standardName);
    });
    return ret;
})();

export default entityToColumns;
