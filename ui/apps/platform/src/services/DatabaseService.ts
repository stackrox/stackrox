import { DatabaseStatus } from 'types/databaseService.proto';

import axios from './instance';

const databaseUrl = '/v1/database/status';

/**
 * Fetches database status.
 */
export function fetchDatabaseStatus(): Promise<DatabaseStatus> {
    return axios.get<DatabaseStatus>(databaseUrl).then((response) => response.data);
}
