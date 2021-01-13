import React, { ReactElement } from 'react';

import { networkFlowStatus } from 'constants/networkGraph';
import { BaselineStatus } from 'Containers/Network/networkTypes';
import { networkFlowStatusLabels } from 'messages/network';

export type EmptyGroupedStatusRowProps = {
    type: BaselineStatus;
    columnCount: number;
};

function EmptyGroupedStatusRow({
    type,
    columnCount = 0,
}: EmptyGroupedStatusRowProps): ReactElement {
    const bgColor = type === networkFlowStatus.ANOMALOUS ? 'bg-alert-200' : '';
    const borderColor = type === networkFlowStatus.ANOMALOUS ? 'border-alert-300' : '';
    const textColor = type === networkFlowStatus.ANOMALOUS ? 'text-alert-800' : '';
    const flowTypeText = networkFlowStatusLabels[type]?.toLowerCase();

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
