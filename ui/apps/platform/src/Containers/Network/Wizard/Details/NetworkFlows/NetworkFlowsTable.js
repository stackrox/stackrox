/* eslint-disable react/display-name */
import React from 'react';
import * as Icon from 'react-feather';
import uniqBy from 'lodash/uniqBy';

import { filterModes, filterLabels } from 'constants/networkFilterModes';
import networkProtocolLabels from 'messages/networkGraph';
import Table, { Expander, rtTrActionsClassName } from 'Components/Table';
import RowActionButton from 'Components/RowActionButton';
import PortsAndProtocolsTable from './PortsAndProtocolsTable';

function renderPortsAndProtocols({ original }) {
    return <PortsAndProtocolsTable portsAndProtocols={original.portsAndProtocols} />;
}

const NetworkFlowsTable = ({
    networkFlows,
    selectedNode,
    page,
    filterState,
    onNavigateToNodeById,
    showPortsAndProtocols,
}) => {
    const filterStateString = filterState !== filterModes.all ? filterLabels[filterState] : '';
    const columns = [
        {
            expander: true,
            Expander: ({ isExpanded, original }) => {
                if (original.portsAndProtocols.length <= 1) {
                    return null;
                }
                return <Expander isExpanded={isExpanded} />;
            },
        },
        {
            Header: 'Traffic',
            accessor: 'traffic',
        },
        {
            Header: 'Deployment',
            accessor: 'deploymentName',
        },
        {
            Header: 'Namespace',
            accessor: 'namespace',
        },
        {
            Header: 'Protocols',
            accessor: 'portsAndProtocols',
            // eslint-disable-next-line react/prop-types
            Cell: ({ value }) => {
                if (value.length === 0) {
                    return '-';
                }
                const protocols = uniqBy(value, (datum) => datum.protocol)
                    .map((datum) => networkProtocolLabels[datum.protocol])
                    .join(', ');
                return protocols;
            },
        },
        {
            Header: 'Ports',
            accessor: 'portsAndProtocols',
            // eslint-disable-next-line react/prop-types
            Cell: ({ value }) => {
                if (value.length === 0) {
                    return '-';
                }
                const ports = uniqBy(value, (datum) => datum.port)
                    .map((datum) => datum.port)
                    .join(', ');
                return ports;
            },
            hidden: !showPortsAndProtocols,
        },
        {
            Header: 'Connection',
            accessor: 'connection',
        },
        {
            accessor: 'deploymentId',
            headerClassName: 'hidden',
            className: rtTrActionsClassName,
            Cell: ({ value }) => {
                return (
                    <div className="border-2 border-r-2 border-base-400 bg-base-100 flex">
                        <RowActionButton
                            text="Navigate to Deployment"
                            onClick={onNavigateToNodeById(value)}
                            icon={<Icon.ArrowUpRight className="my-1 h-4 w-4" />}
                        />
                    </div>
                );
            },
        },
    ];
    const modifiedColumns = columns.filter((column) => {
        return !(
            (column.accessor === 'portsAndProtocols' || column.expander) &&
            !showPortsAndProtocols
        );
    });

    return (
        <Table
            rows={networkFlows}
            columns={modifiedColumns}
            noDataText={`No ${filterStateString} deployment flows`}
            page={page}
            idAttribute="deploymentId"
            selectedRowId={selectedNode?.id}
            SubComponent={showPortsAndProtocols ? renderPortsAndProtocols : null}
        />
    );
};

export default NetworkFlowsTable;
