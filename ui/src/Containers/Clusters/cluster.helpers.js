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
        displayValue: 'Pre-flight checks failed',
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
        displayValue: 'Indeterminate upgrade state!',
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

export function formatUpgradeMessage(upgradeStatus, detail) {
    if (upgradeStatus.type === 'current') {
        return null;
    }
    const message = {
        message:
            upgradeStatus.displayValue ||
            (upgradeStatus.action && upgradeStatus.action.actionText) ||
            'Unknown status',
        type: '',
        detail
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

// This function looks at a cluster upgrade status, and figures out whether the most recent
// upgrade has any information that is of relevance to the user.
function hasRelevantInformationFromMostRecentUpgrade(upgradeStatus) {
    // No information from the most recent upgrade -- probably means no upgrade has been done before.
    // Not interesting.
    if (get(upgradeStatus, 'mostRecentProcess', null) === null) {
        return false;
    }
    const isActive = get(upgradeStatus, 'mostRecentProcess.active', false);
    // The upgrade is currently active. Definitely show the user information about it.
    if (isActive) {
        return true;
    }

    // The upgrade is not active. This means that this is the most recently completed upgrade.
    // If it was COMPLETE, the information is not interesting to the user.
    // Else, we show the user the information.
    return (
        get(upgradeStatus, 'mostRecentProcess.progress.upgradeState', undefined) !==
        'UPGRADE_COMPLETE'
    );
}

export function getUpgradeStatusDetail(upgradeStatus) {
    return get(upgradeStatus, 'mostRecentProcess.progress.upgradeStatusDetail', '');
}

export function parseUpgradeStatus(upgradeStatus) {
    const upgradability = get(upgradeStatus, 'upgradability', undefined);
    if (!upgradability) {
        return null;
    }

    switch (upgradability) {
        case 'UP_TO_DATE':
        case 'MANUAL_UPGRADE_REQUIRED': {
            return upgradeStates[upgradability];
        }
        // Auto upgrades are possible even in the case of SENSOR_VERSION_HIGHER (it's not technically an upgrade,
        // and not really something we ever expect, but eh.) If the backend detects this to be the case, it will not
        // trigger an upgrade unless asked to by the user.
        case 'SENSOR_VERSION_HIGHER':
        case 'AUTO_UPGRADE_POSSIBLE': {
            if (!hasRelevantInformationFromMostRecentUpgrade(upgradeStatus)) {
                return upgradeStates.UPGRADE_AVAILABLE;
            }

            const upgradeState = get(
                upgradeStatus,
                'mostRecentProcess.progress.upgradeState',
                'unknown'
            );

            return upgradeStates[upgradeState] || upgradeStates.unknown;
        }
        default: {
            return upgradeStates.unknown;
        }
    }
}

export function getUpgradeableClusters(clusters = []) {
    return clusters.filter(cluster => {
        const upgradeStatus = get(cluster, 'status.upgradeStatus', null);
        const statusObj = parseUpgradeStatus(upgradeStatus);

        return statusObj.action; // if action property exists, you can try or retry an upgrade
    });
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
