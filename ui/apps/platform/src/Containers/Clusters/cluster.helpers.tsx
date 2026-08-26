import { differenceInDays, differenceInMinutes } from 'date-fns';
import {
    BanIcon,
    CheckCircleIcon,
    ExclamationCircleIcon,
    ExclamationTriangleIcon,
    InfoCircleIcon,
    MinusCircleIcon,
    ResourcesEmptyIcon,
    UnknownIcon,
} from '@patternfly/react-icons';

import type {
    ClusterHealthStatusLabel,
    ClusterProviderMetadata,
    SensorVersionCompatibility,
} from 'types/cluster.proto';
import { getDate, getDistanceStrict } from 'utils/dateUtils';

import { healthStatusLabels } from './cluster.constants';
import type { CertExpiryStatus } from './clusterTypes';

export const runtimeOptions = [
    {
        label: 'No Runtime Collection',
        tableDisplay: 'None',
        value: 'NO_COLLECTION',
    },
    {
        label: 'CORE BPF',
        tableDisplay: 'CORE BPF',
        value: 'CORE_BPF',
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
const defaultCollectionMethod = 'CORE_BPF';

export const newClusterDefault = {
    // TODO Add Cluster type and add missing properties?
    id: undefined, // TODO empty string?
    name: '',
    type: defaultNewClusterType,
    mainImage: 'stackrox/main',
    collectorImage: 'stackrox/collector',
    centralApiEndpoint: 'central.stackrox:443',
    runtimeSupport: false,
    collectionMethod: defaultCollectionMethod,
    admissionControllerEvents: true,
    admissionController: true, // default changed in 4.9
    admissionControllerUpdates: true, // default changed in 4.9
    admissionControllerFailOnError: false, // property added in 4.9 false means Fail open
    status: null,
    tolerationsConfig: {
        disabled: false,
    },
    dynamicConfig: {
        admissionControllerConfig: {
            enabled: true, // default changed in 4.9
            enforceOnUpdates: true, // default changed in 4.9
            timeoutSeconds: 0, // default changed in 4.9
            scanInline: true, // default changed in 4.9
            disableBypass: false,
        },
        registryOverride: '',
        disableAuditLogs: false,
        autoLockProcessBaselinesConfig: null,
    },
    healthStatus: undefined,
    slimCollector: false,
};

export const centralEnvDefault = {
    kernelSupportAvailable: false,
};

const styleUninitializedLegacy = {
    Icon: MinusCircleIcon,
    fgColor: '',
};

const styleUninitialized = {
    Icon: BanIcon,
    fgColor: '',
};

const styleHealthy = {
    Icon: CheckCircleIcon,
    fgColor: 'pf-v6-u-icon-color-status-success',
};

const styleDegraded = {
    Icon: ExclamationTriangleIcon,
    fgColor: 'pf-v6-u-icon-color-status-warning',
};

const styleUnhealthy = {
    Icon: ExclamationCircleIcon,
    fgColor: 'pf-v6-u-icon-color-status-danger',
};

const styleUnavailable = {
    Icon: UnknownIcon,
    fgColor: '',
};

// Styles for ClusterStatus, SensorStatus, CollectorStatus.
// Colors are similar to LabelChip, but fgColor is slightly lighter 700 instead of 800.
export const healthStatusStylesLegacy = {
    UNINITIALIZED: styleUninitializedLegacy,
    UNAVAILABLE: {
        Icon: ResourcesEmptyIcon,
        fgColor: '',
    },
    UNHEALTHY: styleUnhealthy,
    DEGRADED: styleDegraded,
    HEALTHY: styleHealthy,
};

export const healthStatusStyles = {
    UNINITIALIZED: styleUninitialized,
    UNAVAILABLE: styleUnavailable,
    UNHEALTHY: styleUnhealthy,
    DEGRADED: styleDegraded,
    HEALTHY: styleHealthy,
};

// Special case for Collector when Sensor is UNHEALTHY or DELAYED.
export const delayedCollectorStatusStyle = {
    Icon: InfoCircleIcon,
    fgColor: '',
};

// Special case for Admission Control when Sensor is UNHEALTHY or DELAYED.
export const delayedAdmissionControlStatusStyle = {
    Icon: InfoCircleIcon,
    fgColor: '',
};

// Special case for Scanner when Sensor is UNHEALTHY or DELAYED.
export const delayedScannerStatusStyle = {
    Icon: InfoCircleIcon,
    fgColor: '',
};

export const sensorCompatibilityMap = {
    SENSOR_VERSION_COMPATIBILITY_MATCHED: {
        displayValue: 'Matched',
        zoneLabel: 'Matched',
        Icon: CheckCircleIcon,
        fgColor: 'pf-v6-u-icon-color-status-success',
    },
    SENSOR_VERSION_COMPATIBILITY_COMPATIBLE_BEHIND: {
        displayValue: 'Compatible (Behind)',
        zoneLabel: 'Behind',
        Icon: InfoCircleIcon,
        fgColor: 'pf-v6-u-icon-color-status-info',
    },
    SENSOR_VERSION_COMPATIBILITY_COMPATIBLE_AHEAD: {
        displayValue: 'Compatible (Ahead)',
        zoneLabel: 'Ahead',
        Icon: InfoCircleIcon,
        fgColor: 'pf-v6-u-icon-color-status-info',
    },
    SENSOR_VERSION_COMPATIBILITY_INCOMPATIBLE_BEHIND: {
        displayValue: 'Incompatible (Behind)',
        zoneLabel: 'Incompatible',
        Icon: ExclamationCircleIcon,
        fgColor: 'pf-v6-u-icon-color-status-danger',
    },
    SENSOR_VERSION_COMPATIBILITY_INCOMPATIBLE_AHEAD: {
        displayValue: 'Incompatible (Ahead)',
        zoneLabel: 'Incompatible',
        Icon: ExclamationCircleIcon,
        fgColor: 'pf-v6-u-icon-color-status-danger',
    },
    SENSOR_VERSION_COMPATIBILITY_UNKNOWN: {
        displayValue: 'Unknown',
        zoneLabel: '',
        Icon: UnknownIcon,
        fgColor: 'pf-v6-u-icon-color-subtle',
    },
} as const;

export type SensorCompatibilityInfo =
    (typeof sensorCompatibilityMap)[keyof typeof sensorCompatibilityMap];

export function formatKubernetesVersion(orchestratorMetadata: { version: string }) {
    return orchestratorMetadata?.version || 'Not available';
}

export function formatBuildDate(orchestratorMetadata) {
    return orchestratorMetadata?.buildDate
        ? getDate(orchestratorMetadata.buildDate)
        : 'Not available';
}

export function formatCloudProvider(providerMetadata: ClusterProviderMetadata | undefined) {
    if (providerMetadata) {
        const { region } = providerMetadata;

        if ('aws' in providerMetadata) {
            return `AWS ${region}`;
        }

        if ('azure' in providerMetadata) {
            return `Azure ${region}`;
        }

        if ('google' in providerMetadata) {
            return `GCP ${region}`;
        }
    }

    return 'Not available';
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

export const isDelayedSensorHealthStatus = (sensorHealthStatus) =>
    sensorHealthStatus === 'UNHEALTHY' || sensorHealthStatus === 'DEGRADED';

export function buildStatusMessage(
    healthStatus: ClusterHealthStatusLabel,
    lastContact: string | null | undefined,
    sensorHealthStatus: ClusterHealthStatusLabel,
    formatDelayedText: (distance: string) => string = (distance) => `${distance} ago`
): string {
    let message = healthStatusLabels[healthStatus];

    const isDelayed = !!(lastContact && isDelayedSensorHealthStatus(sensorHealthStatus));

    if (isDelayed && lastContact) {
        const distance = getDistanceStrict(lastContact, new Date());
        message += ` ${formatDelayedText(distance)}`;
    }
    return message;
}

const defaultSensorCompatibility = sensorCompatibilityMap.SENSOR_VERSION_COMPATIBILITY_UNKNOWN;

export function getSensorCompatibilityInfo(compatibility: SensorVersionCompatibility | undefined) {
    return compatibility
        ? (sensorCompatibilityMap[compatibility] ?? defaultSensorCompatibility)
        : defaultSensorCompatibility;
}

// The version range chart can only be rendered when the compatibility state is
// known and Central has advertised a compatible sensor version range.
export function shouldShowSensorVersionRangeChart(
    compatibility: SensorVersionCompatibility | undefined,
    compatibleVersions: string[]
): boolean {
    return (
        compatibility !== undefined &&
        compatibility !== 'SENSOR_VERSION_COMPATIBILITY_UNKNOWN' &&
        compatibleVersions.length > 0
    );
}

export default {
    runtimeOptions,
    clusterTypeOptions,
    clusterTablePollingInterval,
    clusterDetailPollingInterval,
    newClusterDefault,
};
