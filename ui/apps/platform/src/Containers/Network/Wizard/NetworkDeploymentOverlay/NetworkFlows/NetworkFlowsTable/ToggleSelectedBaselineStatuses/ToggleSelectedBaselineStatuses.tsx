import React, { ReactElement } from 'react';

import { networkFlowStatus } from 'constants/networkGraph';
import { FlattenedNetworkBaseline } from 'Containers/Network/Wizard/NetworkDeploymentOverlay/NetworkFlows/networkTypes';

import { CondensedButton, CondensedAlertButton } from '@stackrox/ui-components';

import { Row } from '../tableTypes';

export type ToggleSelectedBaselineStatusesProps = {
    row: Row;
    rows: Row[];
    selectedFlatRows: Row[];
};

export function getSelectedRows(selectedFlatRows: Row[]): FlattenedNetworkBaseline[] {
    const selectedRowsMap = selectedFlatRows.reduce((acc, curr) => {
        if (curr.groupByID === 'status') {
            return acc;
        }
        if (curr.isGrouped && curr?.subRows?.length) {
            curr.subRows.forEach((subRow) => {
                if (!acc[subRow.id]) {
                    acc[subRow.id] = subRow.original;
                }
            });
        } else if (!acc[curr.id]) {
            acc[curr.id] = curr.original;
        }
        return acc;
    }, {} as Record<string, FlattenedNetworkBaseline>);
    return Object.values(selectedRowsMap);
}

function ToggleSelectedBaselineStatuses({
    rows,
    row,
    selectedFlatRows,
}: ToggleSelectedBaselineStatusesProps): ReactElement | null {
    const selectedRows = getSelectedRows(selectedFlatRows);

    const anomalousSelectedRows = selectedRows.filter(
        (datum) => datum.status === networkFlowStatus.ANOMALOUS
    );
    const baselineSelectedRows = selectedRows.filter(
        (datum) => datum.status === networkFlowStatus.BASELINE
    );
    const isAnomalousGroup = row.groupByVal === networkFlowStatus.ANOMALOUS;

    function onClick(): void {
        if (isAnomalousGroup) {
            if (anomalousSelectedRows.length) {
                // Replace this with an API call to mark selected rows as anomalous
                // eslint-disable-next-line no-console
                console.log('mark selected as anomalous', anomalousSelectedRows);
            } else {
                const allAnomalousRows = rows.filter(
                    (datum) => datum?.original?.status === networkFlowStatus.ANOMALOUS
                );
                // Replace this with an API call to mark all rows as anomalous
                // eslint-disable-next-line no-console
                console.log('mark all anomalous', allAnomalousRows);
            }
        } else if (baselineSelectedRows.length) {
            // Replace this with an API call to add selected rows to baseline
            // eslint-disable-next-line no-console
            console.log('add selected to baseline', baselineSelectedRows);
        } else {
            const allBaselineRows = rows.filter(
                (datum) => datum?.original?.status === networkFlowStatus.BASELINE
            );
            // Replace this with an API call to add all rows to baseline
            // eslint-disable-next-line no-console
            console.log('add all baseline', allBaselineRows);
        }
    }

    if (isAnomalousGroup) {
        return (
            <CondensedButton type="button" onClick={onClick}>
                Add {anomalousSelectedRows.length || 'all'} to baseline
            </CondensedButton>
        );
    }
    return (
        <CondensedAlertButton type="button" onClick={onClick}>
            Mark {baselineSelectedRows.length || 'all'} as anomalous
        </CondensedAlertButton>
    );
}

export default ToggleSelectedBaselineStatuses;
