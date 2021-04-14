import { TableColorStyles } from 'Components/TableV7';
import { SimulatedBaselineStatus } from '../baselineSimulationTypes';

const SIMULATED_BASELINE_STATES = {
    ADDED: 'ADDED',
    REMOVED: 'REMOVED',
    MODIFIED: 'MODIFIED',
    UNMODIFIED: 'UNMODIFIED',
};

const bgSuccess = 'bg-success-200';
const borderSuccess = 'border-success-300';
const textSuccess = 'text-success-800';

const bgAlert = 'bg-alert-200';
const borderAlert = 'border-alert-300';
const textAlert = 'text-alert-800';

const bgWarning = 'bg-warning-200';
const borderWarning = 'border-warning-300';
const textWarning = 'text-warning-800';

const SIMULATED_ROW_COLORS: Record<string, TableColorStyles> = {
    [SIMULATED_BASELINE_STATES.ADDED]: {
        bgColor: bgSuccess,
        borderColor: borderSuccess,
        textColor: textSuccess,
    },
    [SIMULATED_BASELINE_STATES.REMOVED]: {
        bgColor: bgAlert,
        borderColor: borderAlert,
        textColor: textAlert,
    },
    [SIMULATED_BASELINE_STATES.MODIFIED]: {
        bgColor: bgWarning,
        borderColor: borderWarning,
        textColor: textWarning,
    },
    [SIMULATED_BASELINE_STATES.UNMODIFIED]: {
        bgColor: 'bg-base-100',
        borderColor: 'border-base-300',
        textColor: '',
    },
};

export default function getRowColorStylesByStatus(
    status: SimulatedBaselineStatus
): TableColorStyles {
    const result = SIMULATED_ROW_COLORS[status];
    return result || SIMULATED_ROW_COLORS[SIMULATED_BASELINE_STATES.UNMODIFIED];
}
