import axios from './instance';

/**
 * Fetch array of integration health objects.
 *
 * @returns {Promise<Object, Error>} fulfilled with array of the integration source
 */

export const fetchBackupIntegrationsHealth = () =>
    axios
        .get('/v1/integrationhealth/externalbackups')
        .then((response) => response?.data?.integrationHealth ?? []);

export const fetchImageIntegrationsHealth = () =>
    axios
        .get('/v1/integrationhealth/imageintegrations')
        .then((response) => response?.data?.integrationHealth ?? []);

export const fetchPluginIntegrationsHealth = () =>
    axios
        .get('/v1/integrationhealth/notifiers')
        .then((response) => response?.data?.integrationHealth ?? []);
