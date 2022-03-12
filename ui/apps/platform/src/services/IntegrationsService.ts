import axios from './instance';

type IntegrationSource =
    | 'authPlugins'
    | 'authProviders'
    | 'backups'
    | 'imageIntegrations'
    | 'notifiers'
    | 'signatureIntegrations';

type ActionType = 'create' | 'delete' | 'fetch' | 'save' | 'test' | 'trigger';

function getPath(source: IntegrationSource, action: ActionType): string {
    switch (source) {
        case 'imageIntegrations':
            return '/v1/imageintegrations';
        case 'notifiers':
            return '/v1/notifiers';
        case 'backups':
            return '/v1/externalbackups';
        case 'signatureIntegrations':
            return '/v1/signatureintegrations';
        case 'authPlugins':
            if (action === 'test') {
                return '/v1/scopedaccessctrl';
            }
            if (action === 'fetch') {
                return '/v1/scopedaccessctrl/configs';
            }
            return '/v1/scopedaccessctrl/config';
        default:
            return '';
    }
}

function getJsonFieldBySource(source: IntegrationSource): string {
    switch (source) {
        case 'notifiers':
            return 'notifier';
        case 'backups':
            return 'externalBackup';
        default:
            return 'config';
    }
}

export type IntegrationBase = {
    id: string;
    name: string;
    type: string;
};

export type IntegrationOptions = {
    updatePassword?: boolean; // if integration has stored credentials, aka password
};

/*
 * Fetch list of registered integrations based on source.
 */
export function fetchIntegration(
    source: IntegrationSource
): Promise<{ response: Record<string, unknown> }> {
    const path = getPath(source, 'fetch');
    return axios.get(path).then((response) => ({
        response: response.data,
    }));
}

/*
 * Save an integration by source. If it can potentially use stored credentials, use the
 * updatePassword option to determine if you should use the new API.
 */
export function saveIntegration(
    source: IntegrationSource,
    data: IntegrationBase,
    options: IntegrationOptions = {} // TODO can destructure { updatePassword } for new forms
): Promise<Record<string, never>> {
    const { id } = data;

    if (!id) {
        throw new Error('Integration entity must have an id to be saved');
    }

    const updatePassword = options?.updatePassword; // ROX-7884 because setFormSubmissionOptions can return null

    // if the integration is not one that could possibly have stored credentials, use the previous API
    if (updatePassword === undefined) {
        return axios.put(`${getPath(source, 'save')}/${id}`, data);
    }

    // if it does, format the request data and use the new API
    const integration = {
        [getJsonFieldBySource(source)]: data,
        updatePassword,
    };
    return axios.patch(`${getPath(source, 'save')}/${id}`, integration);
}

// When we migrate completely over, we can remove saveIntegration and rename this
export function saveIntegrationV2(
    source: IntegrationSource,
    data: IntegrationOptions // can also include config, externalBackup, notifier
): Promise<Record<string, never>> {
    const hasUpdatePassword = typeof data.updatePassword === 'boolean';
    if (hasUpdatePassword) {
        // If the data has a config object, use the contents of that config object.
        const config = data[getJsonFieldBySource(source)] as IntegrationBase;
        return axios.patch(`${getPath(source, 'save')}/${config.id}`, data);
    }
    return axios.put(`${getPath(source, 'save')}/${(data as IntegrationBase).id}`, data);
}

/*
 * Create an integration by source.
 */
export function createIntegration(
    source: IntegrationSource,
    data: IntegrationOptions // can also include config, externalBackup, notifier
): Promise<IntegrationBase> {
    // If the data has a config object, use the contents of that config object.
    const hasUpdatePassword = typeof data.updatePassword === 'boolean';
    const createData = hasUpdatePassword ? data[getJsonFieldBySource(source)] : data;

    return axios.post(getPath(source, 'create'), createData);
}

/*
 * Test an integration by source. If it can potentially use stored credentials, use the
 * updatePassword option to determine if you should use the new API.
 */
export function testIntegration(
    source: IntegrationSource,
    data: IntegrationBase,
    options: IntegrationOptions = {} // TODO can destructure { updatePassword } for new forms
): Promise<Record<string, never>> {
    const updatePassword = options?.updatePassword; // ROX-7884 because setFormSubmissionOptions can return null

    // if the integration is not one that could possibly have stored credentials, use the previous API
    if (updatePassword === undefined) {
        return axios.post(`${getPath(source, 'test')}/test`, data);
    }

    // if it does, format the request data and use the new API
    const integration = {
        [getJsonFieldBySource(source)]: data,
        updatePassword,
    };
    return axios.post(`${getPath(source, 'test')}/test/updated`, integration);
}

// When we migrate completely over, we can remove testIntegration and rename this
export function testIntegrationV2(
    source: IntegrationSource,
    data: IntegrationOptions // can also include config, externalBackup, notifier
): Promise<Record<string, never>> {
    if (typeof data.updatePassword === 'boolean') {
        return axios.post(`${getPath(source, 'test')}/test/updated`, data);
    }
    return axios.post(`${getPath(source, 'test')}/test`, data);
}

/*
 * Delete an integration by source.
 */
export function deleteIntegration(
    source: IntegrationSource,
    id: string
): Promise<Record<string, never>> {
    return axios.delete(`${getPath(source, 'delete')}/${id}`);
}

/*
 * Delete an array of integrations by source.
 */
export function deleteIntegrations(
    source: IntegrationSource,
    ids: string[] = []
): Promise<Record<string, never>[]> {
    return Promise.all(ids.map((id) => deleteIntegration(source, id)));
}
