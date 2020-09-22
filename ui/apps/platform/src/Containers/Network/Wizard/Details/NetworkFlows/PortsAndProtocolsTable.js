/* eslint-disable react/display-name */
import React from 'react';

import networkProtocolLabels from 'messages/networkGraph';
import Table, { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';

const PortsAndProtocolsTable = ({ portsAndProtocols }) => {
    const columns = [
        {
            headerClassName: `${defaultHeaderClassName} max-w-10`,
            className: `${defaultColumnClassName} max-w-10 bg-base-300`,
        },
        {
            headerClassName: `${defaultHeaderClassName} w-4`,
            className: `${defaultColumnClassName} w-4 bg-base-200`,
            Header: 'Traffic',
            accessor: 'traffic',
        },
        {
            headerClassName: `${defaultHeaderClassName} w-10`,
            className: `${defaultColumnClassName} w-10 bg-base-200`,
        },
        {
            headerClassName: `${defaultHeaderClassName} w-10`,
            className: `${defaultColumnClassName} w-10 bg-base-200`,
        },
        {
            headerClassName: `${defaultHeaderClassName} w-4`,
            className: `${defaultColumnClassName} w-4 bg-base-200`,
            Header: 'Protocol',
            accessor: 'protocol',
            Cell: ({ value }) => {
                return networkProtocolLabels[value];
            },
        },
        {
            headerClassName: `${defaultHeaderClassName} w-4`,
            className: `${defaultColumnClassName} w-4 bg-base-200`,
            Header: 'Port',
            accessor: 'port',
        },
        {
            headerClassName: `${defaultHeaderClassName} w-4`,
            className: `${defaultColumnClassName} w-4 bg-base-200`,
        },
    ];

    return (
        <Table
            rows={portsAndProtocols}
            columns={columns}
            noDataText="No ports and protocols"
            idAttribute="id"
            showThead={false}
            noHorizontalPadding
        />
    );
};

export default PortsAndProtocolsTable;
