import { Metadata } from 'types/metadataService.proto';

import axios from './instance';

const metadataUrl = '/v1/metadata';

/**
 * Fetches metadata.
 * TODO return Promise<Metadata> when component calls directly instead of indirectly via saga.
 */

export function fetchMetadata(): Promise<{ response: Metadata }> {
    return axios.get<Metadata>(metadataUrl).then((response) => ({
        response: response.data,
    }));
}

// Provides availability of certain functionality of Central Services in the current configuration.
// The initial intended use is to disable certain functionality that does not make sense in the Cloud Service context.

// "Unknown" means "not disabled".
// The reason it's not called "Enabled" is that extended checks might be required to confirm true ability to use
// e.g. container AWS IAM role or GCP workload identity, and we can't positively say "Enabled" while such checks
// aren't implemented.
// This means the user should be allowed to use the capability (both via UI and API) but an attempt may not be
// successful if the corresponding configuration does not match the actual environment.

// Capability is disabled, meaning the corresponding UI should be disabled and attempts to use related APIs
// should lead to errors.

export type CentralServicesCapabilityStatus = 'Unknown' | 'Disabled';

export type CentralServicesCapabilities = {
    // Ability to use container IAM role for scanning images from Amazon ECR using Scanner deployed as part of Central Services.
    centralScanningUseContainerIamRoleForEcr: CentralServicesCapabilityStatus;

    // Ability to configure and perform Central backups to Amazon S3 or Google Cloud Storage.
    centralCloudBackupIntegrations: CentralServicesCapabilityStatus;
};

export function fetchCentralCapabilities(): Promise<CentralServicesCapabilities> {
    return axios
        .get<CentralServicesCapabilities>('/v1/central-capabilities')
        .then((response) => response.data);
}
