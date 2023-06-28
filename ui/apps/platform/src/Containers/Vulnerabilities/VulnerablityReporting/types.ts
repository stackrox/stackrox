import { ReportConfiguration } from 'types/reportConfigurationService.proto';
import { ReportStatus } from 'types/report.proto';

export type Report = ReportConfiguration & {
    reportStatus: ReportStatus;
    reportLastRunStatus: ReportStatus;
};
