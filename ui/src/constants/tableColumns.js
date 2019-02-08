import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import { standardTypes, resourceTypes } from 'constants/entityTypes';

const getColumnValue = (row, accessor) => (row[accessor] ? row[accessor] : 'N/A');

const complianceColumns = [
    {
        accessor: standardTypes.PCI_DSS_3_2,
        Header: 'PCI',
        Cell: ({ original }) => getColumnValue(original, standardTypes.PCI_DSS_3_2)
    },
    {
        accessor: standardTypes.NIST_800_190,
        Header: 'NIST',
        Cell: ({ original }) => getColumnValue(original, standardTypes.NIST_800_190)
    },
    {
        accessor: standardTypes.HIPAA_164,
        Header: 'HIPAA',
        Cell: ({ original }) => getColumnValue(original, standardTypes.HIPAA_164)
    },
    {
        accessor: standardTypes.CIS_KUBERENETES_V1_2_0,
        Header: 'CIS Kube',
        Cell: ({ original }) => getColumnValue(original, standardTypes.CIS_KUBERENETES_V1_2_0)
    },
    {
        accessor: standardTypes.CIS_DOCKER_V1_1_0,
        Header: 'CIS Docker',
        Cell: ({ original }) => getColumnValue(original, standardTypes.CIS_DOCKER_V1_1_0)
    }
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
        Header: 'Cluster'
    },
    ...complianceColumns
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
        Header: 'id',
        headerClassName: 'hidden',
        className: 'hidden'
    },
    {
        accessor: 'name',
        Header: 'Node'
    },
    {
        accessor: standardTypes.NIST_800_190,
        Header: 'NIST',
        Cell: ({ original }) => getColumnValue(original, standardTypes.NIST_800_190)
    },
    {
        accessor: standardTypes.CIS_KUBERENETES_V1_2_0,
        Header: 'CIS Kube',
        Cell: ({ original }) => getColumnValue(original, standardTypes.CIS_KUBERENETES_V1_2_0)
    },
    {
        accessor: standardTypes.CIS_DOCKER_V1_1_0,
        Header: 'CIS Docker',
        Cell: ({ original }) => getColumnValue(original, standardTypes.CIS_DOCKER_V1_1_0)
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
        Header: 'Namespace'
    },
    ...complianceColumns
];

const entityToColumns = {
    [resourceTypes.CLUSTER]: clusterColumns,
    [standardTypes.PCI_DSS_3_2]: getStandardColumns('PCI'),
    [standardTypes.NIST_800_190]: getStandardColumns('NIST'),
    [standardTypes.HIPAA_164]: getStandardColumns('HIPAA'),
    [standardTypes.CIS_KUBERENETES_V1_2_0]: getStandardColumns('CIS Kubernetes'),
    [standardTypes.CIS_DOCKER_V1_1_0]: getStandardColumns('CIS Docker'),
    [resourceTypes.NODE]: nodeColumns,
    [resourceTypes.NAMESPACE]: namespaceColumns
};

export default entityToColumns;
