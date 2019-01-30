import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';

const complianceColumns = [
    {
        accessor: 'node',
        Header: 'Node'
    },
    {
        accessor: 'pci',
        Header: 'PCI'
    },
    {
        accessor: 'nist',
        Header: 'NIST'
    },
    {
        accessor: 'hippa',
        Header: 'HIPPA'
    },
    {
        accessor: 'cis',
        Header: 'CIS'
    }
];

const clusterColumns = [
    {
        accessor: 'id',
        Header: 'ID'
    },
    {
        accessor: 'name',
        Header: 'Name'
    }
];

const controlColumns = [
    {
        accessor: 'control',
        Header: 'PCI Controls',
        headerClassName: `w-5/6 ${defaultHeaderClassName}`,
        className: `w-5/6 ${defaultColumnClassName}`
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
        Header: 'ID'
    },
    {
        accessor: 'name',
        Header: 'Name'
    }
];

const namespaceColumns = [
    {
        accessor: 'id',
        Header: 'ID'
    },
    {
        accessor: 'name',
        Header: 'Name'
    }
];

const entityToColumns = {
    compliance: complianceColumns,
    clusters: clusterColumns,
    control: controlColumns,
    nodes: nodeColumns,
    namespaces: namespaceColumns
};

export default entityToColumns;
