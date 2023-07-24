import { ImageType } from 'services/ReportsService.types';
import { CVESDiscoveredSince } from './forms/useReportFormValues';

export const imageTypeLabelMap: Record<ImageType, string> = {
    DEPLOYED: 'Deployed images',
    WATCHED: 'Watched images',
};

export const cvesDiscoveredSinceLabelMap: Record<CVESDiscoveredSince, string> = {
    ALL_VULN: 'All time',
    SINCE_LAST_REPORT: 'Last successful scheduled run report',
    START_DATE: 'Custom start date',
};

export const commaSeparateWithAnd = (arr: string[]) => {
    if (arr.length === 0) {
        return '';
    }
    if (arr.length === 1) {
        return arr[0];
    }
    const last = arr.pop();
    if (!last) {
        return arr.join(', ');
    }
    return `${arr.join(', ')} and ${last}`;
};
