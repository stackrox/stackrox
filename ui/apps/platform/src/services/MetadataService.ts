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
