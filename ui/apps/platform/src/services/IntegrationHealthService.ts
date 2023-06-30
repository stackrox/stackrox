import { IntegrationHealth } from '../types/integrationHealth.proto';

import axios from './instance';

/**
 * Fetch array of integration health objects.
 */

export const fetchBackupIntegrationsHealth = () =>
    axios
        .get<{ integrationHealth: IntegrationHealth[] }>('/v1/integrationhealth/externalbackups')
        .then((response) => response?.data?.integrationHealth ?? []);

export const fetchImageIntegrationsHealth = () =>
    axios
        .get<{ integrationHealth: IntegrationHealth[] }>('/v1/integrationhealth/imageintegrations')
        .then((response) => response?.data?.integrationHealth ?? []);

export const fetchNotifierIntegrationsHealth = () =>
    axios
        .get<{ integrationHealth: IntegrationHealth[] }>('/v1/integrationhealth/notifiers')
        .then((response) => response?.data?.integrationHealth ?? []);

export const fetchDeclarativeConfigurationsHealth = () =>
    axios
        .get<{ integrationHealth: IntegrationHealth[] }>('/v1/integrationhealth/declarativeconfigs')
        .then((response) => response?.data?.integrationHealth ?? []);

export type VulnerabilityDefinitionsInfo = {
    lastUpdatedTimestamp: string; // ISO 8601 timestamp
};

export const fetchVulnerabilityDefinitionsInfo = () =>
    axios
        .get<VulnerabilityDefinitionsInfo>('/v1/integrationhealth/vulndefinitions')
        .then((response) => response?.data ?? {});
