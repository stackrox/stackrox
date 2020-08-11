import dateFns from 'date-fns';
import get from 'lodash/get';
import { AlertCircle, AlertTriangle, Check, Circle, X } from 'react-feather';

import dateTimeFormat from 'constants/dateTimeFormat';

export const runtimeOptions = [
    {
        label: 'No Runtime Collection',
        tableDisplay: 'None',
        value: 'NO_COLLECTION',
    },
    {
        label: 'Kernel Module',
        tableDisplay: 'Kernel Module',
        value: 'KERNEL_MODULE',
    },
    {
        label: 'eBPF Program',
        tableDisplay: 'eBPF',
        value: 'EBPF',
    },
];

export const clusterTypeOptions = [
    {
        label: 'Kubernetes',
        tableDisplay: 'Kubernetes',
        value: 'KUBERNETES_CLUSTER',
    },
    {
        label: 'OpenShift',
        tableDisplay: 'OpenShift',
        value: 'OPENSHIFT_CLUSTER',
    },
];

export const clusterTablePollingInterval = 5000; // milliseconds
export const clusterDetailPollingInterval = 3000; // milliseconds

const defaultNewClusterType = 'KUBERNETES_CLUSTER';
const defaultCollectionMethod = 'KERNEL_MODULE';

export const newClusterDefault = {
    id: null,
    name: '',
    type: defaultNewClusterType,
    mainImage: 'stackrox/main',
    collectorImage: 'stackrox/collector',
    centralApiEndpoint: 'central.stackrox:443',
    runtimeSupport: false,
    collectionMethod: defaultCollectionMethod,
    DEPRECATEDProviderMetadata: null,
    admissionController: false,
    admissionControllerUpdates: false,
    DEPRECATEDOrchestratorMetadata: null,
    status: null,
    tolerationsConfig: {
        disabled: false,
    },
    dynamicConfig: {
        admissionControllerConfig: {
            enabled: false,
            enforceOnUpdates: false,
            timeoutSeconds: 3,
            scanInline: false,
            disableBypass: false,
        },
    },
    slimCollector: false,
};

export const centralEnvDefault = {
    kernelSupportAvailable: false,
};

// Styles for ClusterStatus, SensorStatus, CollectorStatus.
// Colors are similar to LabelChip, but fgColor is slightly lighter 700 instead of 800.
export const healthStatusStyles = {
    UNINITIALIZED: {
        Icon: Circle,
        bgColor: 'bg-base-200',
        fgColor: 'text-base-700',
    },
    UNAVAILABLE: {
        Icon: AlertCircle,
        bgColor: 'bg-secondary-200',
        fgColor: 'text-secondary-700',
    },
    UNHEALTHY: {
        Icon: X,
        bgColor: 'bg-alert-200',
        fgColor: 'text-alert-700',
    },
    DEGRADED: {
        Icon: AlertTriangle,
        bgColor: 'bg-warning-200',
        fgColor: 'text-warning-700',
    },
    HEALTHY: {
        Icon: Check,
        bgColor: 'bg-success-200',
        fgColor: 'text-success-700',
    },
};

// @TODO: add optional button text and func
const upgradeStates = {
    UP_TO_DATE: {
        displayValue: 'Up to date with Central',
        type: 'current',
    },
    MANUAL_UPGRADE_REQUIRED: {
        displayValue: 'Manual upgrade required',
        type: 'intervention',
    },
    UPGRADE_AVAILABLE: {
        type: 'download',
        actionText: 'Upgrade available',
    },
    UPGRADE_INITIALIZING: {
        displayValue: 'Upgrade initializing',
        type: 'progress',
    },
    UPGRADER_LAUNCHING: {
        displayValue: 'Upgrader launching',
        type: 'progress',
    },
    UPGRADER_LAUNCHED: {
        displayValue: 'Upgrader launched',
        type: 'progress',
    },
    PRE_FLIGHT_CHECKS_COMPLETE: {
        displayValue: 'Pre-flight checks complete',
        type: 'progress',
    },
    UPGRADE_OPERATIONS_DONE: {
        displayValue: 'Upgrade operations done',
        type: 'progress',
    },
    UPGRADE_COMPLETE: {
        displayValue: 'Upgrade complete',
        type: 'current',
    },
    UPGRADE_INITIALIZATION_ERROR: {
        displayValue: 'Upgrade initialization error',
        type: 'failure',
        actionText: 'Retry upgrade',
    },
    PRE_FLIGHT_CHECKS_FAILED: {
        displayValue: 'Pre-flight checks failed',
        type: 'failure',
        actionText: 'Retry upgrade',
    },
    UPGRADE_ERROR_ROLLING_BACK: {
        displayValue: 'Upgrade failed. Rolling back…',
        type: 'failure',
    },
    UPGRADE_ERROR_ROLLED_BACK: {
        displayValue: 'Upgrade failed. Rolled back.',
        type: 'failure',
        actionText: 'Retry upgrade',
    },
    UPGRADE_ERROR_ROLLBACK_FAILED: {
        displayValue: 'Upgrade failed. Rollback failed.',
        type: 'failure',
        actionText: 'Retry upgrade',
    },
    UPGRADE_TIMED_OUT: {
        displayValue: 'Upgrade timed out.',
        type: 'failure',
        actionText: 'Retry upgrade',
    },
    UPGRADE_ERROR_UNKNOWN: {
        displayValue: 'Upgrade error unknown',
        type: 'failure',
        actionText: 'Retry upgrade',
    },
    unknown: {
        displayValue: 'Unknown upgrade state. Contact Support.',
        type: 'intervention',
    },
};

function findOptionInList(options, value) {
    return options.find((opt) => opt.value === value);
}

export function formatClusterType(value) {
    const match = findOptionInList(clusterTypeOptions, value);

    return match.tableDisplay;
}

export function formatCollectionMethod(value) {
    const match = findOptionInList(runtimeOptions, value);

    return match.tableDisplay;
}

export function formatConfiguredField(value) {
    return value ? 'Configured' : 'Not configured';
}

export function formatCloudProvider(providerMetadata) {
    if (providerMetadata) {
        const { region } = providerMetadata;

        if (providerMetadata.aws) {
            return `AWS ${region}`;
        }

        if (providerMetadata.azure) {
            return `Azure ${region}`;
        }

        if (providerMetadata.google) {
            return `GCP ${region}`;
        }
    }

    return '-';
}

export function formatLastCheckIn(status) {
    if (status && status.lastContact) {
        return dateFns.format(status.lastContact, dateTimeFormat);
    }

    return 'N/A';
}

const warningDaysThreshold = 30;

export function getCredentialExpirationProps(certExpiryStatus) {
    if (certExpiryStatus?.sensorCertExpiry) {
        const { sensorCertExpiry } = certExpiryStatus;
        const now = new Date();
        const diffInWords = dateFns.distanceInWordsStrict(sensorCertExpiry, now);
        const diffInDays = dateFns.differenceInDays(sensorCertExpiry, now);
        const showExpiringSoon = diffInDays < warningDaysThreshold;
        let messageType;
        if (diffInDays < 7) {
            messageType = 'error';
        } else if (diffInDays < warningDaysThreshold) {
            messageType = 'warn';
        } else {
            messageType = 'info';
        }
        return { messageType, showExpiringSoon, sensorCertExpiry, diffInWords };
    }
    return null;
}

export function formatSensorVersion(sensorVersion) {
    return sensorVersion || 'Not Running';
}

export const isDelayedSensorHealthStatus = (sensorHealthStatus) =>
    sensorHealthStatus === 'UNHEALTHY' || sensorHealthStatus === 'DEGRADED';

export function formatUpgradeMessage(upgradeStateObject, detail) {
    if (upgradeStateObject.type === 'current') {
        return null;
    }
    const message = {
        message:
            upgradeStateObject.displayValue || upgradeStateObject.actionText || 'Unknown status',
        type: '',
        detail,
    };
    switch (upgradeStateObject.type) {
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

/**
 * If the most recent upgrade was a cert rotation, return the initiation time.
 * Else, return null.
 */
export function initiationOfCertRotationIfApplicable(upgradeStatus) {
    const mostRecentProcess = upgradeStatus?.mostRecentProcess;
    if (mostRecentProcess?.type !== 'CERT_ROTATION') {
        return null;
    }
    if (mostRecentProcess?.progress?.upgradeState !== 'UPGRADE_COMPLETE') {
        return null;
    }
    return mostRecentProcess.initiatedAt;
}

export function findUpgradeState(upgradeStatus) {
    const upgradability = get(upgradeStatus, 'upgradability', null);
    if (!upgradability || upgradability === 'UNSET') {
        return null;
    }

    switch (upgradability) {
        case 'UP_TO_DATE': {
            if (!hasRelevantInformationFromMostRecentUpgrade(upgradeStatus)) {
                return upgradeStates.UP_TO_DATE;
            }

            const upgradeState = get(
                upgradeStatus,
                'mostRecentProcess.progress.upgradeState',
                'unknown'
            );

            return upgradeStates[upgradeState] || upgradeStates.unknown;
        }
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

export function isUpToDateStateObject(upgradeStateObject) {
    return upgradeStateObject.type === 'current';
}

export function getUpgradeableClusters(clusters = []) {
    return clusters.filter((cluster) => {
        const upgradeStatus = get(cluster, 'status.upgradeStatus', null);
        const upgradeStateObject = findUpgradeState(upgradeStatus);

        return upgradeStateObject?.actionText; // if property exists, you can try or retry an upgrade
    });
}

export const wizardSteps = Object.freeze({
    FORM: 'FORM',
    DEPLOYMENT: 'DEPLOYMENT',
});

export default {
    runtimeOptions,
    clusterTypeOptions,
    clusterTablePollingInterval,
    clusterDetailPollingInterval,
    newClusterDefault,
    findUpgradeState,
    formatClusterType,
    formatCollectionMethod,
    formatConfiguredField,
    formatLastCheckIn,
    isUpToDateStateObject,
    wizardSteps,
};
