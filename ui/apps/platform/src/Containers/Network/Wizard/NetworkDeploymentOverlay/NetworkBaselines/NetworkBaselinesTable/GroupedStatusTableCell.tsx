import React, { ReactElement } from 'react';
import pluralize from 'pluralize';

import { networkFlowStatus } from 'constants/networkGraph';
import { networkFlowStatusLabels } from 'messages/network';

export type GroupedStatusTableCellProps = {
    row: {
        cells: {
            length: number;
        };
        leafRows: {
            length: number;
        };
        groupByVal: 'ANOMALOUS' | 'BASELINE';
    };
};

function GroupedStatusTableCell({ row }: GroupedStatusTableCellProps): ReactElement {
    const { cells, leafRows, groupByVal } = row;
    const isAnomalous = row.groupByVal === networkFlowStatus.ANOMALOUS;

    const flowText = pluralize('Flow', leafRows.length);
    const text = `${leafRows.length} ${networkFlowStatusLabels[groupByVal]} ${flowText}`;

    return (
        <td
            colSpan={cells.length}
            className={`text-left p-2 italic ${
                isAnomalous
                    ? 'bg-alert-200 border-b border-alert-300'
                    : 'bg-base-100 border-b border-base-300'
            }`}
        >
            {text}
        </td>
    );
}

export default GroupedStatusTableCell;
