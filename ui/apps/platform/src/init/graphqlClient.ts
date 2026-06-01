import { GraphQLClient } from 'graphql-request';
import { buildAxiosFetch } from '@lifeomic/axios-fetch';

import axios from 'services/instance';

const axiosFetch = buildAxiosFetch(axios, (config) => {
    const { operationName } = JSON.parse(config.data);

    return {
        ...config,
        timeout: 180000,
        url: `${config.url}?opname=${operationName}`,
    };
});

export const gqlClient = new GraphQLClient('/api/graphql', {
    fetch: axiosFetch as typeof fetch,
});
