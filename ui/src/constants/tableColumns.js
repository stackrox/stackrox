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

const getStandardColumns = standard => [
    {
        accessor: 'control',
        Header: `${standard} Controls`,
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
    'PCI DSS 3.2': getStandardColumns('PCI'),
    'NIST 800-190': getStandardColumns('NIST'),
    'HIPAA 164': getStandardColumns('HIPAA'),
    'CIS Kubernetes v1.2.0': getStandardColumns('CIS Kubernetes'),
    'CIS Docker v1.1.0': getStandardColumns('CIS Docker'),
    nodes: nodeColumns,
    namespaces: namespaceColumns
};

export default entityToColumns;
