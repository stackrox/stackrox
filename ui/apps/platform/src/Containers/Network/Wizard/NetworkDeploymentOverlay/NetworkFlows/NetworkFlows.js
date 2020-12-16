import React, { useState } from 'react';
import pluralize from 'pluralize';

import { getNetworkFlows } from 'utils/networkGraphUtils';
import { filterModes } from 'constants/networkFilterModes';

import Panel from 'Components/Panel';
import TablePagination from 'Components/TablePagination';
import NetworkFlowsTable from './NetworkFlowsTable';

function getPanelId(filterState) {
    switch (filterState) {
        case filterModes.active:
            return 'active-network-flows';
        case filterModes.allowed:
            return 'allowed-network-flows';
        default:
            return 'network-flows';
    }
}

function getPanelHeaderText(networkFlows, filterState) {
    switch (filterState) {
        case filterModes.active:
            return `${networkFlows.length} active ${pluralize('flow', networkFlows.length)}`;
        case filterModes.allowed:
            return `${networkFlows.length} allowed ${pluralize('flow', networkFlows.length)}`;
        default:
            return `${networkFlows.length} ${pluralize('flow', networkFlows.length)}`;
    }
}

function NetworkFlows({ edges, filterState, onNavigateToDeploymentById }) {
    const { networkFlows } = getNetworkFlows(edges, filterState);
    const [page, setPage] = useState(0);

    const panelId = getPanelId(filterState);
    const panelHeader = getPanelHeaderText(networkFlows, filterState);
    const headerComponents = (
        <TablePagination page={page} dataLength={networkFlows.length} setPage={setPage} />
    );

    return (
        <Panel id={panelId} header={panelHeader} headerComponents={headerComponents}>
            <NetworkFlowsTable
                networkFlows={networkFlows}
                page={page}
                filterState={filterState}
                onNavigateToDeploymentById={onNavigateToDeploymentById}
            />
        </Panel>
    );
}

export default NetworkFlows;
