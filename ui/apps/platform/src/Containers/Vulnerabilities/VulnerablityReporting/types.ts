import { ReportConfiguration, ReportStatus } from 'services/ReportsService.types';

export type Report = ReportConfiguration & {
    reportStatus: ReportStatus | null;
    reportLastRunStatus: ReportStatus | null;
};
