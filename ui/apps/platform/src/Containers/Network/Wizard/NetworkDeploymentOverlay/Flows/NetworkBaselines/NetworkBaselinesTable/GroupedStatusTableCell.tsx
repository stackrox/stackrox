import React, { ReactElement } from 'react';
import pluralize from 'pluralize';

import { BaselineStatus } from 'Containers/Network/networkTypes';
import { networkFlowStatusLabels } from 'messages/network';
import { Cell } from './tableTypes';
import { isExpanderCell } from './expanderPlugin';

import TableCell from './TableCell';
import { TableColorStyles } from '../networkBaseline.utils';

export type GroupedStatusTableCellProps = {
    row: {
        cells: Cell[];
        leafRows: {
            length: number;
        };
        groupByVal: BaselineStatus;
    };
    colorStyles: TableColorStyles;
};

function GroupedStatusTableCell({ row, colorStyles }: GroupedStatusTableCellProps): ReactElement {
    const { cells, leafRows, groupByVal } = row;
    const { bgColor, borderColor, textColor } = colorStyles;

    const flowText = pluralize('Flow', leafRows.length);
    const text = `${leafRows.length} ${networkFlowStatusLabels[groupByVal]} ${flowText}`;
    const className = `sticky z-1 top-8 text-left p-2 italic border-b border-t ${bgColor} ${borderColor} ${textColor}`;
    const [expanderCell] = cells.filter(isExpanderCell);
    const colSpan = cells.length - (expanderCell ? 1 : 0);

    return (
        <>
            {expanderCell && <TableCell cell={expanderCell} colorStyles={colorStyles} isSticky />}
            <td colSpan={colSpan} className={className}>
                {text}
            </td>
        </>
    );
}

export default GroupedStatusTableCell;
