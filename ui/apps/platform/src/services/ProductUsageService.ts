import qs from 'qs';
import axios from './instance';
import {
    MaxSecuredUnitsUsageResponse,
    SecuredUnitsUsage,
    TimeRange,
} from '../types/productUsage.proto';
import { saveFile } from './DownloadService';

export function fetchCurrentProductUsage() {
    return axios.get<SecuredUnitsUsage>('/v1/product/usage/secured-units/current');
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
        .get<MaxSecuredUnitsUsageResponse>(`/v1/product/usage/secured-units/max?${queryString}`)
        .then((response) => {
            return response.data;
        });
}

export function downloadProductUsageCsv({ from, to }: TimeRange): Promise<void> {
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
        url: `/api/product/usage/secured-units/csv?${queryString}`,
        data: null,
    });
}
