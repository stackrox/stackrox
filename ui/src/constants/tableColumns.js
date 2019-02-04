import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import { standardTypes } from 'constants/entityTypes';

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
        accessor: 'namespace',
        Header: 'Name'
    }
];

const entityToColumns = {
    compliance: complianceColumns,
    clusters: clusterColumns,
    [standardTypes.PCI_DSS_3_2]: getStandardColumns('PCI'),
    [standardTypes.NIST_800_190]: getStandardColumns('NIST'),
    [standardTypes.HIPAA_164]: getStandardColumns('HIPAA'),
    [standardTypes.CIS_KUBERENETES_V1_2_0]: getStandardColumns('CIS Kubernetes'),
    [standardTypes.CIS_DOCKER_V1_1_0]: getStandardColumns('CIS Docker'),
    nodes: nodeColumns,
    namespaces: namespaceColumns
};

export default entityToColumns;
