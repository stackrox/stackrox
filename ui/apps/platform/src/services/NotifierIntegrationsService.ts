import type { Traits } from 'types/traits.proto';
import axios from './instance';

import type { IntegrationBase } from './IntegrationsService';

const notifierIntegrationsUrl = '/v1/notifiers';

// See proto/storage/notifier.proto

export type NotifierIntegrationBase = {
    uiEndpoint: string;
    labelKey: string;
    labelDefault: string;
    traits?: Traits;
} & IntegrationBase;

/*
 * Read integrations (plural).
 */
export function fetchNotifierIntegrations(): Promise<NotifierIntegrationBase[]> {
    return axios
        .get<{ notifiers: NotifierIntegrationBase[] }>(notifierIntegrationsUrl)
        .then((response) => response?.data?.notifiers ?? []);
}
