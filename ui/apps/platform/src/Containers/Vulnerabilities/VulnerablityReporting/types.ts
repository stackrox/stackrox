import { ReportConfiguration, ReportSnapshot } from 'services/ReportsService.types';

export type Report = ReportConfiguration & {
    reportSnapshot: ReportSnapshot | null;
};
