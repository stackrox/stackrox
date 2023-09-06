import { ReportStatus } from 'services/ReportsService.types';
import { getReportStatusText } from './utils';

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
});
