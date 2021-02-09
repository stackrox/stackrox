import React, { ReactElement } from 'react';

import { BaselineStatus } from 'Containers/Network/networkTypes';
import { networkFlowStatusLabels } from 'messages/network';
import { getEmptyFlowRowColors } from '../networkBaseline.utils';

export type EmptyGroupedStatusRowProps = {
    baselineStatus: BaselineStatus;
    columnCount: number;
};
function EmptyGroupedStatusRow({
    baselineStatus,
    columnCount = 0,
}: EmptyGroupedStatusRowProps): ReactElement {
    const { bgColor, borderColor, textColor } = getEmptyFlowRowColors(baselineStatus);

    const flowTypeText = networkFlowStatusLabels[baselineStatus]?.toLowerCase();

    return (
        <tr className={`relative border-b ${borderColor} ${bgColor} ${textColor}`}>
            <td
                colSpan={columnCount + 1}
                className="p-2 text-center"
            >{`No ${flowTypeText} flows.`}</td>
            <td className="flex overflow-visible w-0" />
        </tr>
    );
}

export default EmptyGroupedStatusRow;
