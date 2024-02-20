import React, { ReactElement } from 'react';
import { CheckCircleIcon, SecurityIcon, UnknownIcon } from '@patternfly/react-icons';

import {
    DiscoveredClusterProviderType,
    DiscoveredClusterStatus,
    DiscoveredClusterType,
} from 'services/DiscoveredClusterService';

// providerType

const providerTypeMap: Record<DiscoveredClusterProviderType, string> = {
    PROVIDER_TYPE_AWS: 'AWS',
    PROVIDER_TYPE_AZURE: 'Azure',
    PROVIDER_TYPE_GCP: 'GCP',
    PROVIDER_TYPE_UNSPECIFIED: 'Not available',
};

export function getProviderRegionText(providerType: DiscoveredClusterProviderType, region: string) {
    const providerText = providerTypeMap[providerType] ?? providerTypeMap.PROVIDER_TYPE_UNSPECIFIED;
    return region ? `${providerText} (${region})` : providerText;
}

// status

const iconMap: Record<DiscoveredClusterStatus, ReactElement> = {
    STATUS_SECURED: <CheckCircleIcon color="var(--pf-global--success-color--100)" />,
    STATUS_UNSECURED: <SecurityIcon color="var(--pf-global--danger-color--100)" />,
    STATUS_UNSPECIFIED: <UnknownIcon />,
};

export function getStatusIcon(status: DiscoveredClusterStatus): ReactElement {
    return iconMap[status] ?? iconMap.STATUS_UNSPECIFIED;
}

const statusMap: Record<DiscoveredClusterStatus, string> = {
    STATUS_SECURED: 'Secured',
    STATUS_UNSECURED: 'Unsecured',
    STATUS_UNSPECIFIED: 'Undetermined',
};

export function getStatusText(status: DiscoveredClusterStatus) {
    return statusMap[status] ?? statusMap.STATUS_UNSPECIFIED;
}

const statusTipMap: Record<DiscoveredClusterStatus, string> = {
    STATUS_SECURED: 'This cluster is already secured.',
    STATUS_UNSECURED: 'This cluster has been discovered by a cloud source, but is not yet secured.',
    STATUS_UNSPECIFIED:
        'This cluster has has been discovered by a cloud source, but has undetermined status because metadata collected from secured clusters does not guarantee a unique match. Sensor requires access to the AWS EC2 instance tags to determine the cluster state.',
};

export function getStatusTip(status: DiscoveredClusterStatus) {
    return statusTipMap[status] ?? statusTipMap.STATUS_UNSPECIFIED;
}

// type

export function getTypeText(type: DiscoveredClusterType) {
    // Return AKS and so on, except for special case.
    return type !== 'UNSPECIFIED' ? type : 'Not available';
}
