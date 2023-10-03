import { ReportStatus } from 'services/ReportsService.types';
import { getReportStatusText, getCVEsDiscoveredSinceText } from './utils';
import { ReportParametersFormValues } from './forms/useReportFormValues';

// @TODO: Consider making a more unique name for general utils file under Vulnerability Reporting
describe('utils', () => {
    describe('getReportStatusText', () => {
        it('should show the correct text for different report statuses', () => {
            const reportStatus: ReportStatus = {
                runState: 'DELIVERED',
                completedAt: '2023-06-20T10:59:46.383433891Z',
                errorMsg: '',
                reportRequestType: 'ON_DEMAND',
                reportNotificationMethod: 'EMAIL',
            };
            let isDownloadAvailable = false;
            expect(getReportStatusText(reportStatus, isDownloadAvailable)).toEqual('Emailed');

            reportStatus.runState = 'GENERATED';
            reportStatus.reportNotificationMethod = 'DOWNLOAD';
            isDownloadAvailable = true;

            expect(getReportStatusText(reportStatus, isDownloadAvailable)).toEqual(
                'Download prepared'
            );

            reportStatus.runState = 'GENERATED';
            reportStatus.reportNotificationMethod = 'DOWNLOAD';
            isDownloadAvailable = false;

            expect(getReportStatusText(reportStatus, isDownloadAvailable)).toEqual(
                'Download deleted'
            );

            reportStatus.runState = 'FAILURE';
            reportStatus.reportNotificationMethod = 'EMAIL';

            expect(getReportStatusText(reportStatus, isDownloadAvailable)).toEqual(
                'Email attempted'
            );

            reportStatus.runState = 'FAILURE';
            reportStatus.reportNotificationMethod = 'DOWNLOAD';

            expect(getReportStatusText(reportStatus, isDownloadAvailable)).toEqual(
                'Failed to generate download'
            );

            reportStatus.runState = 'FAILURE';
            reportStatus.reportNotificationMethod = 'UNSET';

            expect(getReportStatusText(reportStatus, isDownloadAvailable)).toEqual('Error');

            reportStatus.runState = 'PREPARING';
            reportStatus.reportNotificationMethod = 'DOWNLOAD';

            expect(getReportStatusText(reportStatus, isDownloadAvailable)).toEqual('Preparing');

            reportStatus.runState = 'PREPARING';
            reportStatus.reportNotificationMethod = 'EMAIL';

            expect(getReportStatusText(reportStatus, isDownloadAvailable)).toEqual('Preparing');

            reportStatus.runState = 'WAITING';
            reportStatus.reportNotificationMethod = 'DOWNLOAD';

            expect(getReportStatusText(reportStatus, isDownloadAvailable)).toEqual('Waiting');

            reportStatus.runState = 'WAITING';
            reportStatus.reportNotificationMethod = 'EMAIL';

            expect(getReportStatusText(reportStatus, isDownloadAvailable)).toEqual('Waiting');
        });
    });

    describe('getCVEsDiscoveredSinceText', () => {
        it('should display the correct text when presenting CVEs discovered from all time', () => {
            const reportParameters: ReportParametersFormValues = {
                reportName: 'Test Report',
                reportDescription: '',
                cveSeverities: [],
                cveStatus: [],
                imageType: [],
                cvesDiscoveredSince: 'ALL_VULN',
                cvesDiscoveredStartDate: undefined,
                reportScope: null,
            };

            const text = getCVEsDiscoveredSinceText(reportParameters);

            expect(text).toBe('All time');
        });

        it('should display the correct text when presenting CVEs discovered since the last scheduled report that was successfully sent', () => {
            const reportParameters: ReportParametersFormValues = {
                reportName: 'Test Report',
                reportDescription: '',
                cveSeverities: [],
                cveStatus: [],
                imageType: [],
                cvesDiscoveredSince: 'SINCE_LAST_REPORT',
                cvesDiscoveredStartDate: undefined,
                reportScope: null,
            };

            const text = getCVEsDiscoveredSinceText(reportParameters);

            expect(text).toBe('Last scheduled report that was successfully sent');
        });

        it('should display the correct text when presenting CVEs discovered since a specific start date', () => {
            const reportParameters: ReportParametersFormValues = {
                reportName: 'Test Report',
                reportDescription: '',
                cveSeverities: [],
                cveStatus: [],
                imageType: [],
                cvesDiscoveredSince: 'START_DATE',
                cvesDiscoveredStartDate: '2023-10-02',
                reportScope: null,
            };

            const text = getCVEsDiscoveredSinceText(reportParameters);

            expect(text).toBe('10/02/2023');
        });
    });
});
