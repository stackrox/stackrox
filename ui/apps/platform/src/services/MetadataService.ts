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

// CapabilityAvailable means that UI and APIs should be available for users to use.
// This does not automatically mean that the functionality is 100% available and any calls to APIs will result
// in successful execution. Rather it means that users should be allowed to leverage the functionality as
// opposed to CapabilityDisabled when functionality should be blocked.

// CapabilityDisabled means the corresponding UI should be disabled and attempts to use related APIs
// should lead to errors.

export type CentralServicesCapabilityStatus = 'CapabilityAvailable' | 'CapabilityDisabled';

export type CentralServicesCapabilities = {
    // Ability to use container IAM role for scanning images from Amazon ECR using Scanner deployed as part of Central
    // Services.
    // Note that CapabilityAvailable status does not mean that Scanner container actually has IAM role attached. Such
    // check isn't implemented at the moment and an attempt to use the corresponding setting may lead to errors when
    // the role is not actually there. It's user's responsibility to check the presence of role and integration status
    // when the corresponding setting is enabled.
    centralScanningCanUseContainerIamRoleForEcr: CentralServicesCapabilityStatus;

    // Ability to configure and perform Central backups to Amazon S3 or Google Cloud Storage.
    centralCanUseCloudBackupIntegrations: CentralServicesCapabilityStatus;

    // Ability to present health of declarative config resources (e.g. auth providers, roles, access scopes, permission
    // sets, notifiers) to the user.
    centralCanDisplayDeclarativeConfigHealth: CentralServicesCapabilityStatus;
};

export type CentralCapabilitiesFlags = keyof CentralServicesCapabilities;

export function fetchCentralCapabilities(): Promise<CentralServicesCapabilities> {
    return axios
        .get<CentralServicesCapabilities>('/v1/central-capabilities')
        .then((response) => response.data);
}
