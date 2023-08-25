import axios from './instance';

import { DeclarativeConfigHealth } from '../types/declarativeConfigHealth.proto';

const healthUrl = '/v1/declarative-config/health';

/**
 * Fetches the declarative config health objects.
 */
export function fetchDeclarativeConfigurationsHealth(): Promise<{
    response: { healths: DeclarativeConfigHealth[] };
}> {
    return axios.get<{ healths: DeclarativeConfigHealth[] }>(healthUrl).then((response) => ({
        response: response.data,
    }));
}
