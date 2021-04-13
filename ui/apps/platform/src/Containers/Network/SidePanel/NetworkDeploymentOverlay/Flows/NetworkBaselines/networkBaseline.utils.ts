import { networkFlowStatus } from 'constants/networkGraph';
import { BaselineStatus } from 'Containers/Network/networkTypes';
import { TableColorStyles } from 'Components/TableV7';

const bgAlert = 'bg-alert-200';
const borderAlert = 'border-alert-300';
const textAlert = 'text-alert-800';

const bgWarning = 'bg-warning-200';
const borderWarning = 'border-warning-700';
const textWarning = 'text-warning-800';

const EMPTY_FLOW_ROW_COLORS: Record<string, TableColorStyles> = {
    [networkFlowStatus.ANOMALOUS]: {
        bgColor: bgAlert,
        borderColor: borderAlert,
        textColor: textAlert,
    },
    [networkFlowStatus.BLOCKED]: {
        bgColor: bgWarning,
        borderColor: borderWarning,
        textColor: textWarning,
    },
    [networkFlowStatus.BASELINE]: {
        bgColor: '',
        borderColor: '',
        textColor: '',
    },
};

const FLOW_ROW_COLORS: Record<string, TableColorStyles> = {
    [networkFlowStatus.ANOMALOUS]: {
        bgColor: bgAlert,
        borderColor: borderAlert,
        textColor: textAlert,
    },
    [networkFlowStatus.BLOCKED]: {
        bgColor: bgWarning,
        borderColor: borderWarning,
        textColor: textWarning,
    },
    [networkFlowStatus.BASELINE]: {
        bgColor: 'bg-base-100',
        borderColor: 'border-base-300',
        textColor: '',
    },
};

export function getEmptyFlowRowColors(type: BaselineStatus): TableColorStyles {
    return EMPTY_FLOW_ROW_COLORS[type] || EMPTY_FLOW_ROW_COLORS[networkFlowStatus.BASELINE];
}

export function getFlowRowColors(type: BaselineStatus): TableColorStyles {
    return FLOW_ROW_COLORS[type] || FLOW_ROW_COLORS[networkFlowStatus.BASELINE];
}
