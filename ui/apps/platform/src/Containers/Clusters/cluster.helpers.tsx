import React from 'react';
import { differenceInDays, differenceInMinutes } from 'date-fns';
import get from 'lodash/get';
import { DownloadCloud } from 'react-feather';
import {
    CheckCircleIcon,
    ExclamationCircleIcon,
    InfoCircleIcon,
    InProgressIcon,
    MinusCircleIcon,
    ResourcesEmptyIcon,
    TimesCircleIcon,
} from '@patternfly/react-icons';

import { getDate } from 'utils/dateUtils';
import { CertExpiryStatus } from './clusterTypes';

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

export const clusterTypes = {
    KUBERNETES: 'KUBERNETES_CLUSTER',
    OPENSHIFT_3: 'OPENSHIFT_CLUSTER',
    OPENSHIFT_4: 'OPENSHIFT4_CLUSTER',
};

export const clusterTypeOptions = [
    {
        label: 'Kubernetes',
        tableDisplay: 'Kubernetes',
        value: clusterTypes.KUBERNETES,
    },
    {
        label: 'OpenShift 3.x compatiblity mode',
        tableDisplay: 'OpenShift 3.x compatiblity mode',
        value: clusterTypes.OPENSHIFT_3,
    },
    {
        label: 'OpenShift 4.x',
        tableDisplay: 'OpenShift 4.x',
        value: clusterTypes.OPENSHIFT_4,
    },
];

export const clusterTablePollingInterval = 5000; // milliseconds
export const clusterDetailPollingInterval = 3000; // milliseconds

const defaultNewClusterType = 'KUBERNETES_CLUSTER';
const defaultCollectionMethod = 'EBPF';

export const newClusterDefault = {
    id: undefined,
    name: '',
    type: defaultNewClusterType,
    mainImage: 'stackrox/main',
    collectorImage: 'stackrox/collector',
    centralApiEndpoint: 'central.stackrox:443',
    runtimeSupport: false,
    collectionMethod: defaultCollectionMethod,
    DEPRECATEDProviderMetadata: null,
    admissionControllerEvents: true,
    admissionController: false,
    admissionControllerUpdates: false,
    DEPRECATEDOrchestratorMetadata: null,
    status: undefined,
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
        registryOverride: '',
    },
    healthStatus: undefined,
    slimCollector: false,
};

export const centralEnvDefault = {
    kernelSupportAvailable: false,
};

type MinusCircleRotate45Props = {
    className: string;
};

const MinusCircleRotate45 = ({ className }: MinusCircleRotate45Props) => (
    <MinusCircleIcon className={`${className} transform rotate-45`} />
);

export const styleUninitialized = {
    Icon: MinusCircleRotate45,
    bgColor: 'bg-base-200',
    fgColor: 'text-base-700',
};

export const styleHealthy = {
    Icon: CheckCircleIcon,
    bgColor: 'bg-success-200',
    fgColor: 'text-success-700',
};

export const styleDegraded = {
    Icon: ExclamationCircleIcon,
    bgColor: 'bg-warning-200',
    fgColor: 'text-warning-700',
};

export const styleUnhealthy = {
    Icon: TimesCircleIcon,
    bgColor: 'bg-alert-200',
    fgColor: 'text-alert-700',
};

// PatternFly versions of cluster style constants
export const styleUninitializedPF = {
    Icon: MinusCircleRotate45,
    bgColor: 'pf-u-background-color-100',
    fgColor: 'pf-u-default-color-300',
};

export const styleHealthyPF = {
    Icon: CheckCircleIcon,
    bgColor: 'pf-u-background-color-success',
    fgColor: 'pf-u-success-color-100',
};

export const styleDegradedPF = {
    Icon: ExclamationCircleIcon,
    bgColor: 'pf-u-background-color-warning',
    fgColor: 'pf-u-warning-color-100',
};

export const styleUnhealthyPF = {
    Icon: TimesCircleIcon,
    bgColor: 'pf-u-background-color-danger',
    fgColor: 'pf-u-danger-color-100',
};

// Styles for ClusterStatus, SensorStatus, CollectorStatus.
// Colors are similar to LabelChip, but fgColor is slightly lighter 700 instead of 800.
export const healthStatusStyles = {
    UNINITIALIZED: styleUninitialized,
    UNAVAILABLE: {
        Icon: ResourcesEmptyIcon,
        bgColor: 'bg-secondary-200',
        fgColor: 'text-secondary-700',
    },
    UNHEALTHY: styleUnhealthy,
    DEGRADED: styleDegraded,
    HEALTHY: styleHealthy,
};

// Special case for Collector when Sensor is UNHEALTHY or DELAYED.
export const delayedCollectorStatusStyle = {
    Icon: InfoCircleIcon,
    bgColor: 'bg-base-200',
    fgColor: 'text-base-700',
};

// Special case for Admission Control when Sensor is UNHEALTHY or DELAYED.
export const delayedAdmissionControlStatusStyle = {
    Icon: InfoCircleIcon,
    bgColor: 'bg-base-200',
    fgColor: 'text-base-700',
};

// Special case for Scanner when Sensor is UNHEALTHY or DELAYED.
export const delayedScannerStatusStyle = {
    Icon: InfoCircleIcon,
    bgColor: 'bg-base-200',
    fgColor: 'text-base-700',
};

export const sensorUpgradeStyles = {
    current: styleHealthy,
    progress: {
        Icon: InProgressIcon,
        bgColor: 'bg-tertiary-200',
        fgColor: 'text-tertiary-700',
    },
    download: {
        Icon: DownloadCloud,
        bgColor: 'bg-tertiary-200',
        fgColor: 'text-tertiary-700',
    },
    intervention: styleDegraded,
    failure: styleUnhealthy,
};

type UpgradeState = {
    displayValue?: string;
    type: string;
    actionText?: string;
};
type UpgradeStates = Record<string, UpgradeState>;

// @TODO: add optional button text and func
const upgradeStates: UpgradeStates = {
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

export function formatKubernetesVersion(orchestratorMetadata: { version: string }) {
    return orchestratorMetadata?.version || 'Not applicable';
}

export function formatBuildDate(orchestratorMetadata) {
    return orchestratorMetadata?.buildDate
        ? getDate(orchestratorMetadata.buildDate)
        : 'Not applicable';
}

type ProviderMetadata = {
    region: string;
    aws?: any;
    azure?: any;
    google?: any;
};

export function formatCloudProvider(providerMetadata: ProviderMetadata) {
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

    return 'Not applicable';
}

const shortLivedCertMaxDays = 14;

const longLivedCertThresholds = {
    thresholdDegradedMinutes: 7 * 24 * 60, // Unhealthy if less than a week before expiry
    thresholdHealthyMinutes: 30 * 24 * 60, // Degraded if less than a month before expiry
};

const shortLivedCertThresholds = {
    thresholdDegradedMinutes: 15, // Unhealthy if less than 15 minutes before expiry
    thresholdHealthyMinutes: 59, // Degraded if less than an hour before expiry
};

const resolveThresholds = (expiryStatus: CertExpiryStatus) => {
    const certDurationDays = differenceInDays(
        expiryStatus.sensorCertExpiry,
        expiryStatus.sensorCertNotBefore
    );
    return certDurationDays <= shortLivedCertMaxDays
        ? shortLivedCertThresholds
        : longLivedCertThresholds;
};

/*
 * Adapt health status categories to certificate expiration.
 */
export const getClusterDeletionStatus = (daysUntilDeletion: number) => {
    if (daysUntilDeletion < 7) {
        return 'UNHEALTHY';
    }
    if (daysUntilDeletion < 30) {
        return 'DEGRADED';
    }
    return 'UNINITIALIZED';
};

/*
 * Adapt health status categories to certificate expiration.
 */
export const getCredentialExpirationStatus = (
    sensorCertExpiryStatus: CertExpiryStatus,
    currentDatetime
) => {
    const { sensorCertExpiry } = sensorCertExpiryStatus;
    const diffInMinutes = differenceInMinutes(sensorCertExpiry, currentDatetime);
    const { thresholdDegradedMinutes, thresholdHealthyMinutes } =
        resolveThresholds(sensorCertExpiryStatus);

    if (diffInMinutes < thresholdDegradedMinutes) {
        return 'UNHEALTHY';
    }

    if (diffInMinutes < thresholdHealthyMinutes) {
        return 'DEGRADED';
    }

    return 'HEALTHY';
};

export const isCertificateExpiringSoon = (
    sensorCertExpiryStatus: CertExpiryStatus,
    currentDatetime
) => getCredentialExpirationStatus(sensorCertExpiryStatus, currentDatetime) !== 'HEALTHY';

export function formatSensorVersion(sensorVersion: string) {
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

export function getUpgradeStatusDetail(upgradeStatus: string): string {
    return get(upgradeStatus, 'mostRecentProcess.progress.upgradeStatusDetail', '') as string;
}

export type UpgradeStatus = {
    mostRecentProcess: {
        type: string;
        progress?: {
            upgradeState: string;
        };
        initiatedAt?: string;
        active?: boolean;
        upgradability?: string;
    };
};

/**
 * If the most recent upgrade was a cert rotation, return the initiation time.
 * Else, return null.
 */
export function initiationOfCertRotationIfApplicable(upgradeStatus: UpgradeStatus) {
    const mostRecentProcess = upgradeStatus?.mostRecentProcess;
    if (mostRecentProcess?.type !== 'CERT_ROTATION') {
        return null;
    }
    if (mostRecentProcess?.progress?.upgradeState !== 'UPGRADE_COMPLETE') {
        return null;
    }
    return mostRecentProcess.initiatedAt;
}

export function findUpgradeState(
    upgradeStatus: UpgradeStatus | null | undefined
): UpgradeState | null {
    const upgradability = get(upgradeStatus, 'upgradability', null);
    if (!upgradability || upgradability === 'UNSET') {
        return null;
    }

    switch (upgradability) {
        case 'UP_TO_DATE': {
            if (!upgradeStatus?.mostRecentProcess?.active) {
                return upgradeStates.UP_TO_DATE;
            }

            // Display active progress while using automatic upgrade to re-issue certificates.
            const upgradeState: string = get(
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
    return upgradeStateObject?.type === 'current';
}

export function getUpgradeableClusters(clusters = []) {
    return clusters.filter((cluster) => {
        const upgradeStatus: UpgradeStatus | null = get(cluster, 'status.upgradeStatus', null);
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
    isUpToDateStateObject,
    wizardSteps,
};
