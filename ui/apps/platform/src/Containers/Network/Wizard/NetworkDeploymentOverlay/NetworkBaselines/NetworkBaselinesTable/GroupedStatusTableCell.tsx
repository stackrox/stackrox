import React, { ReactElement } from 'react';
import pluralize from 'pluralize';

import { networkFlowStatus } from 'constants/networkGraph';
import { networkFlowStatusLabels } from 'messages/network';
import { Cell } from './tableTypes';
import { isExpanderCell } from './expanderPlugin';

import TableCell from './TableCell';

export type GroupedStatusTableCellProps = {
    row: {
        cells: Cell[];
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
    const className = `text-left p-2 italic ${
        isAnomalous
            ? 'bg-alert-200 border-b border-alert-300'
            : 'bg-base-100 border-b border-base-300'
    }`;
    const [expanderCell] = cells.filter(isExpanderCell);
    const colSpan = cells.length - (expanderCell ? 1 : 0);

    return (
        <>
            {expanderCell && <TableCell cell={expanderCell} />}
            <td colSpan={colSpan} className={className}>
                {text}
            </td>
        </>
    );
}

export default GroupedStatusTableCell;
