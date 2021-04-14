import { TableColorStyles } from 'Components/TableV7';
import { SimulatedBaselineStatus } from './baselineSimulationTypes';

const bgSuccess = 'bg-success-200';
const borderSuccess = 'border-success-300';
const textSuccess = 'text-success-800';

const bgAlert = 'bg-alert-200';
const borderAlert = 'border-alert-300';
const textAlert = 'text-alert-800';

const bgWarning = 'bg-warning-200';
const borderWarning = 'border-warning-300';
const textWarning = 'text-warning-800';

const SIMULATED_ROW_COLORS: Record<SimulatedBaselineStatus, TableColorStyles> = {
    ADDED: {
        bgColor: bgSuccess,
        borderColor: borderSuccess,
        textColor: textSuccess,
    },
    REMOVED: {
        bgColor: bgAlert,
        borderColor: borderAlert,
        textColor: textAlert,
    },
    MODIFIED: {
        bgColor: bgWarning,
        borderColor: borderWarning,
        textColor: textWarning,
    },
    UNMODIFIED: {
        bgColor: 'bg-base-100',
        borderColor: 'border-base-300',
        textColor: '',
    },
};

export default function getRowColorStylesByStatus(
    status: SimulatedBaselineStatus
): TableColorStyles {
    const result = SIMULATED_ROW_COLORS[status];
    return result || SIMULATED_ROW_COLORS.UNMODIFIED;
}
