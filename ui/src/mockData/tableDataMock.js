import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';

const groupedData = [
    {
        name:
            'install and maintain something install and maintain something install and maintain something install and maintain something install and maintain something install and maintain something install and maintain something',
        rows: [
            {
                control: '1.2',
                compliance: '10%'
            },
            {
                control: '1.6.3',
                compliance: '86%'
            }
        ]
    },
    {
        name: 'maintain somethin blakjsdfoi slk',
        rows: [
            {
                control: '1.4',
                compliance: '10%'
            },
            {
                control: '1.8.3',
                compliance: '86%'
            }
        ]
    }
];

const subTableColumns = [
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

const tableData = [
    {
        node: 'Node 1',
        pci: '30%',
        nist: '10%',
        hippa: '55%',
        cis: '90%'
    },
    {
        node: 'Node 2',
        pci: '60%',
        nist: '13%',
        hippa: '25%',
        cis: '95%'
    },
    {
        node: 'Node 3',
        pci: '6%',
        nist: '30%',
        hippa: '22%',
        cis: '57%'
    }
];

const tableColumns = [
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

export { groupedData, subTableColumns, tableData, tableColumns };
