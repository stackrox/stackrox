import React, { ReactElement } from 'react';
import pluralize from 'pluralize';

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
    const flowText = pluralize('Flow', leafRows.length);
    const text = `${leafRows.length} ${networkFlowStatusLabels[groupByVal]} ${flowText}`;

    return (
        <td colSpan={cells.length} className="text-left p-2 italic">
            {text}
        </td>
    );
}

export default GroupedStatusTableCell;
