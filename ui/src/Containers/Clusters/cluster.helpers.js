import dateFns from 'date-fns';
import get from 'lodash/get';

import dateTimeFormat from 'constants/dateTimeFormat';

export const runtimeOptions = [
    {
        label: 'No Runtime Support',
        tableDisplay: 'None',
        value: 'NO_COLLECTION'
    },
    {
        label: 'Kernel Module Support',
        tableDisplay: 'Kernel Module',
        value: 'KERNEL_MODULE'
    },
    {
        label: 'eBPF Support',
        tableDisplay: 'eBPF',
        value: 'EBPF'
    }
];

export const clusterTypeOptions = [
    {
        label: 'Kubernetes',
        tableDisplay: 'Kubernetes',
        value: 'KUBERNETES_CLUSTER'
    },
    {
        label: 'OpenShift',
        tableDisplay: 'OpenShift',
        value: 'OPENSHIFT_CLUSTER'
    }
];

export const clusterTablePollingInterval = 5000; // milliseconds
export const clusterDetailPollingInterval = 3000; // milliseconds

const defaultNewClusterType = 'KUBERNETES_CLUSTER';
const defaultCollectionMethod = 'NO_COLLECTION';

export const newClusterDefault = {
    id: null,
    name: '',
    type: defaultNewClusterType,
    mainImage: '',
    collectorImage: '',
    centralApiEndpoint: '',
    runtimeSupport: false,
    monitoringEndpoint: '',
    collectionMethod: defaultCollectionMethod,
    DEPRECATEDProviderMetadata: null,
    admissionController: false,
    DEPRECATEDOrchestratorMetadata: null,
    status: null,
    dynamicConfig: {
        admissionControllerConfig: {
            enabled: false,
            timeoutSeconds: 3,
            scanInline: false,
            disableBypass: false
        }
    }
};

// @TODO: add optional button text and func
const upgradeStates = {
    UP_TO_DATE: {
        displayValue: 'On the latest version',
        type: 'current'
    },
    MANUAL_UPGRADE_REQUIRED: {
        displayValue: 'Manual upgrade required',
        type: 'intervention'
    },
    UPGRADE_AVAILABLE: {
        type: 'download',
        action: {
            actionText: 'Upgrade available'
        }
    },
    UPGRADE_INITIALIZING: {
        displayValue: 'Upgrade initializing',
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
    UPGRADE_OPERATIONS_DONE: {
        displayValue: 'Upgrade operations done',
        type: 'progress'
    },
    UPGRADE_COMPLETE: {
        displayValue: 'Upgrade complete',
        type: 'current'
    },
    UPGRADE_INITIALIZATION_ERROR: {
        displayValue: 'Upgrade initialization error',
        type: 'failure',
        action: {
            actionText: 'Retry upgrade'
        }
    },
    PRE_FLIGHT_CHECKS_FAILED: {
        displayValue: 'Pre-flight checks failed.',
        type: 'failure',
        action: {
            actionText: 'Retry upgrade'
        }
    },
    UPGRADE_ERROR_ROLLING_BACK: {
        displayValue: 'Upgrade failed. Rolling back…',
        type: 'failure'
    },
    UPGRADE_ERROR_ROLLED_BACK: {
        displayValue: 'Upgrade failed. Rolled back.',
        type: 'failure',
        action: {
            actionText: 'Retry upgrade'
        }
    },
    UPGRADE_ERROR_ROLLBACK_FAILED: {
        displayValue: 'Upgrade failed. Rollback failed.',
        type: 'failure',
        action: {
            actionText: 'Retry upgrade'
        }
    },
    UPGRADE_TIMED_OUT: {
        displayValue: 'Upgrade timed out.',
        type: 'failure',
        action: {
            actionText: 'Retry upgrade'
        }
    },
    UPGRADE_ERROR_UNKNOWN: {
        displayValue: 'Upgrade error unknown',
        type: 'failure',
        action: {
            actionText: 'Retry upgrade'
        }
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

export function formatUpgradeMessage(upgradeStatus) {
    if (upgradeStatus.type === 'current') {
        return null;
    }
    const message = {
        message:
            upgradeStatus.displayValue ||
            (upgradeStatus.action && upgradeStatus.action.actionText) ||
            'Unknown status',
        type: ''
    };
    switch (upgradeStatus.type) {
        case 'failure': {
            message.type = 'error';
            break;
        }
        case 'intervention': {
            message.type = 'warn';
            break;
        }
        default: {
            message.type = 'info';
        }
    }
    return message;
}

export function parseUpgradeStatus(cluster) {
    const upgradability = get(cluster, 'status.upgradeStatus.upgradability', undefined);
    switch (upgradability) {
        case 'UP_TO_DATE':
        case 'MANUAL_UPGRADE_REQUIRED': {
            return upgradeStates[upgradability];
        }
        case 'AUTO_UPGRADE_POSSIBLE': {
            const isActive = get(
                cluster,
                'status.upgradeStatus.mostRecentProcess.active',
                undefined
            );
            if (!isActive) {
                return upgradeStates.UPGRADE_AVAILABLE;
            }
            const upgradeState = get(
                cluster,
                'status.upgradeStatus.mostRecentProcess.progress.upgradeState',
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
    runtimeOptions,
    clusterTypeOptions,
    clusterTablePollingInterval,
    clusterDetailPollingInterval,
    newClusterDefault,
    formatClusterType,
    formatCollectionMethod,
    formatEnabledDisabledField,
    formatLastCheckIn,
    formatUpgradeMessage,
    parseUpgradeStatus,
    wizardSteps
};
