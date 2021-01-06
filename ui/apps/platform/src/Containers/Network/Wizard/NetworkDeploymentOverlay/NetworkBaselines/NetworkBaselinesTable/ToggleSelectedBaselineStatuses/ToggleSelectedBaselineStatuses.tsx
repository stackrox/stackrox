import React, { ReactElement } from 'react';

import { networkFlowStatus } from 'constants/networkGraph';
import { FlattenedNetworkBaseline } from 'Containers/Network/networkTypes';

import { CondensedButton, CondensedAlertButton } from '@stackrox/ui-components';

import { Row } from '../tableTypes';

export type ToggleSelectedBaselineStatusesProps = {
    row: Row;
    rows: Row[];
    selectedFlatRows: Row[];
    toggleBaselineStatuses: (networkBaselines: FlattenedNetworkBaseline[]) => void;
};

export function getSelectedRows(selectedFlatRows: Row[]): FlattenedNetworkBaseline[] {
    const selectedRowsMap = selectedFlatRows.reduce<Record<string, FlattenedNetworkBaseline>>(
        (acc, curr) => {
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
        },
        {}
    );
    return Object.values(selectedRowsMap);
}

function ToggleSelectedBaselineStatuses({
    rows,
    row,
    selectedFlatRows,
    toggleBaselineStatuses,
}: ToggleSelectedBaselineStatusesProps): ReactElement | null {
    const selectedRows = getSelectedRows(selectedFlatRows);

    const anomalousSelectedRows = selectedRows.filter(
        (datum) => datum.status === networkFlowStatus.ANOMALOUS
    );
    const baselineSelectedRows = selectedRows.filter(
        (datum) => datum.status === networkFlowStatus.BASELINE
    );
    const isAnomalousGroup = row.groupByVal === networkFlowStatus.ANOMALOUS;

    function onClickHandler(): void {
        if (isAnomalousGroup) {
            if (anomalousSelectedRows.length) {
                toggleBaselineStatuses(anomalousSelectedRows);
            } else {
                const allAnomalousRows = rows
                    .filter(
                        (datum) =>
                            !(datum.isGrouped && datum.groupByID === 'status') &&
                            datum.values.status === networkFlowStatus.ANOMALOUS
                    )
                    .reduce<FlattenedNetworkBaseline[]>((acc, curr) => {
                        if (curr?.subRows?.length) {
                            curr.subRows.forEach((subRow) => {
                                acc.push(subRow.original);
                            });
                        }
                        return acc;
                    }, []);
                toggleBaselineStatuses(allAnomalousRows);
            }
        } else if (baselineSelectedRows.length) {
            toggleBaselineStatuses(baselineSelectedRows);
        } else {
            const allBaselineRows = rows
                .filter(
                    (datum) =>
                        !(datum.isGrouped && datum.groupByID === 'status') &&
                        datum.values.status === networkFlowStatus.BASELINE
                )
                .reduce<FlattenedNetworkBaseline[]>((acc, curr) => {
                    if (curr?.subRows?.length) {
                        curr.subRows.forEach((subRow) => {
                            acc.push(subRow.original);
                        });
                    }

                    return acc;
                }, []);
            toggleBaselineStatuses(allBaselineRows);
        }
    }

    if (isAnomalousGroup) {
        return (
            <CondensedButton type="button" onClick={onClickHandler}>
                Add {anomalousSelectedRows.length || 'all'} to baseline
            </CondensedButton>
        );
    }
    return (
        <CondensedAlertButton type="button" onClick={onClickHandler}>
            Mark {baselineSelectedRows.length || 'all'} as anomalous
        </CondensedAlertButton>
    );
}

export default ToggleSelectedBaselineStatuses;
