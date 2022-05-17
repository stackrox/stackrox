import { Metadata } from 'types/metadataService.proto';

import axios from './instance';

/**
 * Fetches metadata.
 * TODO return Promise<Metadata> when component calls directly instead of indirectly via saga.
 */

// eslint-disable-next-line import/prefer-default-export
export function fetchMetadata(): Promise<{ response: Metadata }> {
    const metadataUrl = '/v1/metadata';
    return axios.get<Metadata>(metadataUrl).then((response) => ({
        response: response.data,
    }));
}
