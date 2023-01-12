import { DatabaseStatus } from 'types/databaseService.proto';

import axios from './instance';

const databaseUrl = '/v1/database/status';

/**
 * Fetches database.
 * TODO return Promise<DatabaseStatus> when component calls directly instead of indirectly via saga.
 */

export function fetchDatabaseStatus(): Promise<DatabaseStatus> {
    return axios.get<DatabaseStatus>(databaseUrl).then((response) => response.data);
}
