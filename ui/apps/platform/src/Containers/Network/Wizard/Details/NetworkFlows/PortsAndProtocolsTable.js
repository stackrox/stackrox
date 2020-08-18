/* eslint-disable react/display-name */
import React from 'react';

import networkProtocolLabels from 'messages/networkGraph';
import Table from 'Components/Table';

const PortsAndProtocolsTable = ({ portsAndProtocols }) => {
    const columns = [
        {
            Header: 'Traffic',
            accessor: 'traffic',
        },
        {
            Header: 'Protocol',
            accessor: 'protocol',
            Cell: ({ value }) => {
                return networkProtocolLabels[value];
            },
        },
        {
            Header: 'Port',
            accessor: 'port',
        },
    ];

    return (
        <Table
            rows={portsAndProtocols}
            columns={columns}
            noDataText="No ports and protocols"
            idAttribute="id"
        />
    );
};

export default PortsAndProtocolsTable;
