import React, { useState, ReactElement } from 'react';
import { PlusCircle, MinusCircle } from 'react-feather';

import CustomDialogue from 'Components/CustomDialogue';
import { networkFlowStatus } from 'constants/networkGraph';
import { FlattenedNetworkBaseline } from 'Containers/Network/networkTypes';

import { CondensedButton, CondensedAlertButton } from '@stackrox/ui-components';

import { Row } from '../tableTypes';

function getAllRowsByType(rows: Row[], flowType: string): FlattenedNetworkBaseline[] {
    return rows
        .filter(
            (datum) =>
                !(datum.isGrouped && datum.groupByID === 'status') &&
                datum.values.status === flowType
        )
        .reduce<FlattenedNetworkBaseline[]>((acc, curr) => {
            if (curr?.subRows?.length) {
                curr.subRows.forEach((subRow) => {
                    acc.push(subRow.original);
                });
            }
            return acc;
        }, []);
}

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
                        acc[subRow.id] = subRow?.original;
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
    const [showMoveFlowDialog, setShowMoveFlowDialog] = useState(false);

    const selectedRows = getSelectedRows(selectedFlatRows);

    const anomalousSelectedRows = selectedRows.filter(
        (datum) => datum.status === networkFlowStatus.ANOMALOUS
    );
    const baselineSelectedRows = selectedRows.filter(
        (datum) => datum.status === networkFlowStatus.BASELINE
    );
    const isAnomalousGroup = row.groupByVal === networkFlowStatus.ANOMALOUS;

    function moveFlows(): void {
        if (isAnomalousGroup) {
            if (anomalousSelectedRows.length) {
                toggleBaselineStatuses(anomalousSelectedRows);
            } else {
                const allAnomalousRows = getAllRowsByType(rows, networkFlowStatus.ANOMALOUS);
                toggleBaselineStatuses(allAnomalousRows);
            }
        } else if (baselineSelectedRows.length) {
            toggleBaselineStatuses(baselineSelectedRows);
        } else {
            const allBaselineRows = getAllRowsByType(rows, networkFlowStatus.BASELINE);
            toggleBaselineStatuses(allBaselineRows);
        }
    }

    function onClickHandler(): void {
        setShowMoveFlowDialog(true);
    }

    function cancelMovingFlows(): void {
        setShowMoveFlowDialog(false);
    }

    const ToggleFlowButton = isAnomalousGroup ? CondensedButton : CondensedAlertButton;
    const numRows = isAnomalousGroup
        ? anomalousSelectedRows.length || 'all'
        : baselineSelectedRows.length || 'all';
    const buttonText = isAnomalousGroup
        ? `Add ${numRows} to baseline`
        : `Mark ${numRows} as anomalous`;
    const IconToShow = isAnomalousGroup ? PlusCircle : MinusCircle;
    const hasRowsToToggle = isAnomalousGroup
        ? anomalousSelectedRows.length > 0 ||
          getAllRowsByType(rows, networkFlowStatus.ANOMALOUS).length > 0
        : baselineSelectedRows.length > 0 ||
          getAllRowsByType(rows, networkFlowStatus.BASELINE).length > 0;

    return (
        <>
            {hasRowsToToggle && (
                <ToggleFlowButton type="button" onClick={onClickHandler}>
                    <IconToShow className="h-3 w-3 mr-1" />
                    {buttonText}
                </ToggleFlowButton>
            )}
            {showMoveFlowDialog && (
                <CustomDialogue
                    title={`${buttonText}?`}
                    onConfirm={moveFlows}
                    confirmText="Yes"
                    onCancel={cancelMovingFlows}
                />
            )}
        </>
    );
}

export default ToggleSelectedBaselineStatuses;
