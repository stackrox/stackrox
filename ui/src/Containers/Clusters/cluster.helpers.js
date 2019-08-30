import dateFns from 'date-fns';
import get from 'lodash/get';

import dateTimeFormat from 'constants/dateTimeFormat';

const runtimeOptions = [
    {
        inputOption: 'No Runtime Support',
        tableDisplay: 'None',
        value: 'NO_COLLECTION'
    },
    {
        inputOption: 'Kernel Module Support',
        tableDisplay: 'Kernel Module',
        value: 'KERNEL_MODULE'
    },
    {
        inputOption: 'eBPF Support',
        tableDisplay: 'eBPF',
        value: 'EBPF'
    }
];

const clusterTypeOptions = [
    {
        inputOption: 'Kubernetes',
        tableDisplay: 'Kubernetes',
        value: 'KUBERNETES_CLUSTER'
    },
    {
        inputOption: 'OpenShift',
        tableDisplay: 'OpenShift',
        value: 'OPENSHIFT_CLUSTER'
    }
];

// @TODO: add optional button text and func
const upgradeStates = {
    UNSET: {
        displayValue: 'Upgrade available',
        type: 'download'
    },
    UPGRADE_TRIGGER_SENT: {
        displayValue: 'Upgrade trigger sent',
        type: 'progress'
    },
    UPGRADER_LAUNCHING: {
        displayValue: 'Upgrader launching',
        type: 'progress'
    },
    UPGRADER_LAUNCHED: {
        displayValue: 'Upgrader launched',
        type: 'progress'
    },
    PRE_FLIGHT_CHECKS_COMPLETE: {
        displayValue: 'Pre-flight checks complete',
        type: 'progress'
    },
    PRE_FLIGHT_CHECKS_FAILED: {
        displayValue: 'Pre-flight checks failed.',
        type: 'failure'
    },
    UPGRADE_OPERATIONS_DONE: {
        displayValue: 'Upgrade Operations Done',
        type: 'progress'
    },
    UPGRADE_OPERATIONS_COMPLETE: {
        displayValue: 'Upgrade Operations Complete',
        type: 'current'
    },
    UPGRADE_ERROR_ROLLED_BACK: {
        displayValue: 'Upgrade failed. Rolled back.',
        type: 'failure'
    },
    UPGRADE_ERROR_ROLLBACK_FAILED: {
        displayValue: 'Upgrade failed. Rollback failed.',
        type: 'failure'
    },
    unknown: {
        displayValue: 'Undeterminate upgrade state!',
        type: 'intervention'
    }
};

function findOptionInList(options, value) {
    return options.find(opt => opt.value === value);
}

export function formatClusterType(value) {
    const match = findOptionInList(clusterTypeOptions, value);

    return match.tableDisplay;
}

export function formatCollectionMethod(value) {
    const match = findOptionInList(runtimeOptions, value);

    return match.tableDisplay;
}

export function formatEnabledDisabledField(value) {
    return value ? 'Enabled' : 'Disabled';
}
export function formatLastCheckIn(status) {
    if (status && status.lastContact) {
        return dateFns.format(status.lastContact, dateTimeFormat);
    }

    return 'N/A';
}

export function formatSensorVersion(status) {
    return (status && status.sensorVersion) || 'Not Running';
}

export function parseUpgradeStatus(cluster) {
    const upgradability = get(cluster, 'status.upgradeStatus.upgradability', undefined);
    switch (upgradability) {
        case 'UP_TO_DATE': {
            return {
                displayValue: 'On the latest version',
                type: 'current'
            };
        }
        case 'MANUAL_UPGRADE_REQUIRED': {
            return {
                displayValue: 'Manual upgrade required',
                type: 'intervention'
            };
        }
        case 'AUTO_UPGRADE_POSSIBLE': {
            const upgradeState = get(
                cluster,
                'status.upgradeStatus.upgradeProgress.upgradeState',
                'unknown'
            );

            return upgradeStates[upgradeState] || upgradeStates.unknown;
        }
        default: {
            return upgradeStates.unknown;
        }
    }
}

export const wizardSteps = Object.freeze({
    FORM: 'FORM',
    DEPLOYMENT: 'DEPLOYMENT'
});

export default {
    formatClusterType,
    formatCollectionMethod,
    formatEnabledDisabledField,
    formatLastCheckIn,
    parseUpgradeStatus,
    wizardSteps
};
