import qs from 'qs';
import axios from './instance';
import {
    MaxSecuredUnitsUsageResponse,
    SecuredUnitsUsage,
    TimeRange,
} from '../types/administrationUsage.proto';
import { saveFile } from './DownloadService';

export function fetchCurrentAdministrationUsage() {
    return axios.get<SecuredUnitsUsage>('/v1/administration/usage/secured-units/current');
}

export function fetchMaxCurrentUsage({ from, to }: TimeRange) {
    const queryString = qs.stringify(
        {
            from,
            to,
        },
        {
            allowDots: true,
            arrayFormat: 'repeat',
        }
    );
    return axios
        .get<MaxSecuredUnitsUsageResponse>(`/v1/administration/usage/secured-units/max?${queryString}`)
        .then((response) => {
            return response.data;
        });
}

export function downloadAdministrationUsageCsv({ from, to }: TimeRange): Promise<void> {
    const queryString = qs.stringify(
        {
            from,
            to,
        },
        {
            allowDots: true,
            arrayFormat: 'repeat',
        }
    );
    return saveFile({
        method: 'get',
        url: `/api/administration/usage/secured-units/csv?${queryString}`,
        data: null,
    });
}
