import { subDays } from 'date-fns';
import { ReportStatus } from 'types/reportJob';
import {
    getReportStatusText,
    getCVEsDiscoveredSinceText,
    calculateReportExpiration,
} from './utils';
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
                includeAdvisory: false,
                includeEpssProbability: false,
                // Ross CISA KEV includeExploitable
                includeNvdCvss: false,
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
                includeAdvisory: false,
                includeEpssProbability: false,
                // Ross CISA KEV includeExploitable
                includeNvdCvss: false,
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
                includeAdvisory: false,
                includeEpssProbability: false,
                // Ross CISA KEV includeExploitable
                includeNvdCvss: false,
                cvesDiscoveredSince: 'START_DATE',
                cvesDiscoveredStartDate: '2023-10-02',
                reportScope: null,
            };

            const text = getCVEsDiscoveredSinceText(reportParameters);

            expect(text).toBe('Oct 02, 2023');
        });
    });

    describe('calculateReportExpiration', () => {
        const mockToday = new Date('2025-09-23T14:30:00Z');

        beforeEach(() => {
            vi.useFakeTimers();
            vi.setSystemTime(mockToday);
        });

        afterEach(() => {
            vi.useRealTimers();
        });

        describe('null/undefined inputs', () => {
            it('should return "Pending" when completedAt is null', () => {
                const result = calculateReportExpiration(null, 7);
                expect(result).toBe('Pending');
            });

            it('should return "Unknown" when retentionDays is undefined', () => {
                const completedAt = '2025-09-18T12:00:00Z';
                const result = calculateReportExpiration(completedAt, undefined);
                expect(result).toBe('Unknown');
            });

            it('should return "Pending" when both inputs are null/undefined', () => {
                const result = calculateReportExpiration(null, undefined);
                expect(result).toBe('Pending');
            });
        });

        describe('expired vs non-expired boundary conditions', () => {
            it('should return "Expired" when report completed more than retention days ago', () => {
                // Report completed 10 days ago with 7 day retention
                const completedAt = subDays(mockToday, 10).toISOString();
                const result = calculateReportExpiration(completedAt, 7);
                expect(result).toBe('Expired');
            });

            it('should return "Expired" when report completed exactly retention days + 1 ago', () => {
                // Report completed 8 days ago with 7 day retention
                const completedAt = subDays(mockToday, 8).toISOString();
                const result = calculateReportExpiration(completedAt, 7);
                expect(result).toBe('Expired');
            });

            it('should not be expired when report completed exactly retention days ago', () => {
                // Report completed 7 days ago with 7 day retention - should expire today
                const completedAt = subDays(mockToday, 7).toISOString();
                const result = calculateReportExpiration(completedAt, 7);
                expect(result).toBe('Expires today');
            });

            it('should return correct days remaining when not expired', () => {
                // Report completed 5 days ago with 7 day retention - 2 days left
                const completedAt = subDays(mockToday, 5).toISOString();
                const result = calculateReportExpiration(completedAt, 7);
                expect(result).toBe('2 days');
            });
        });

        describe('"Expires today" vs "1 day" boundary', () => {
            it('should return "Expires today" when 0 days remaining', () => {
                // Report completed exactly retention days ago
                const completedAt = subDays(mockToday, 7).toISOString();
                const result = calculateReportExpiration(completedAt, 7);
                expect(result).toBe('Expires today');
            });

            it('should return "1 day" when exactly 1 day remaining', () => {
                // Report completed retention days - 1 ago
                const completedAt = subDays(mockToday, 6).toISOString();
                const result = calculateReportExpiration(completedAt, 7);
                expect(result).toBe('1 day');
            });

            it('should return "2 days" when exactly 2 days remaining', () => {
                // Report completed retention days - 2 ago
                const completedAt = subDays(mockToday, 5).toISOString();
                const result = calculateReportExpiration(completedAt, 7);
                expect(result).toBe('2 days');
            });
        });

        describe('time normalization', () => {
            it('should ignore time portion and only consider dates', () => {
                // Different times on the same day should give same result
                const morningCompletion = '2025-09-16T06:00:00Z';
                const eveningCompletion = '2025-09-16T22:00:00Z';

                const morningResult = calculateReportExpiration(morningCompletion, 7);
                const eveningResult = calculateReportExpiration(eveningCompletion, 7);

                expect(morningResult).toBe(eveningResult);
                expect(morningResult).toBe('Expires today');
            });
        });
    });
});
