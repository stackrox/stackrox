import React, { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import { ReportSnapshot, ReportStatus } from 'services/ReportsService.types';

import MyLastReportJobStatus from './MyLastReportJobStatus';

describe('MyLastReportJobStatus', () => {
    test('should show "PREPARING" when your last job status is preparing', async () => {
        const reportStatus: ReportStatus = {
            runState: 'PREPARING',
            completedAt: '',
            errorMsg: '',
            reportRequestType: 'ON_DEMAND',
            reportNotificationMethod: 'DOWNLOAD',
        };
        const reportSnapshot: ReportSnapshot = {
            reportConfigId: 'report-config-id-1',
            reportJobId: 'report-job-id-1',
            name: 'test-name-1',
            description: 'test-description-1',
            vulnReportFilters: {
                fixability: 'FIXABLE',
                severities: ['LOW_VULNERABILITY_SEVERITY'],
                imageTypes: ['DEPLOYED'],
                allVuln: true,
            },
            collectionSnapshot: {
                id: 'test-collection-id-1',
                name: 'test-collection-name-1',
            },
            schedule: null,
            user: {
                id: 'test-user-1',
                name: 'test-user-name-1',
            },
            reportStatus,
            notifiers: [],
            isDownloadAvailable: false,
        };

        render(
            <MyLastReportJobStatus
                reportSnapshot={reportSnapshot}
                isLoadingReportSnapshots={false}
                currentUserId="test-user-1"
            />
        );

        const statusTextElement = screen.getByText('Preparing');
        const statusIconElement = screen.getByTitle('Report run is preparing');

        expect(statusTextElement).toBeInTheDocument();
        expect(statusIconElement).toBeInTheDocument();
    });

    test('should show "WAITING" when your last job status is waiting', async () => {
        const reportStatus: ReportStatus = {
            runState: 'WAITING',
            completedAt: '',
            errorMsg: '',
            reportRequestType: 'ON_DEMAND',
            reportNotificationMethod: 'DOWNLOAD',
        };
        const reportSnapshot: ReportSnapshot = {
            reportConfigId: 'report-config-id-1',
            reportJobId: 'report-job-id-1',
            name: 'test-name-1',
            description: 'test-description-1',
            vulnReportFilters: {
                fixability: 'FIXABLE',
                severities: ['LOW_VULNERABILITY_SEVERITY'],
                imageTypes: ['DEPLOYED'],
                allVuln: true,
            },
            collectionSnapshot: {
                id: 'test-collection-id-1',
                name: 'test-collection-name-1',
            },
            schedule: null,
            user: {
                id: 'test-user-1',
                name: 'test-user-name-1',
            },
            reportStatus,
            notifiers: [],
            isDownloadAvailable: false,
        };

        render(
            <MyLastReportJobStatus
                reportSnapshot={reportSnapshot}
                isLoadingReportSnapshots={false}
                currentUserId="test-user-1"
            />
        );

        const statusTextElement = screen.getByText('Waiting');
        const statusIconElement = screen.getByTitle('Report run is waiting');

        expect(statusTextElement).toBeInTheDocument();
        expect(statusIconElement).toBeInTheDocument();
    });

    test('should show "Ready for download" when your last job status successfully generates a download', async () => {
        const reportStatus: ReportStatus = {
            runState: 'GENERATED',
            completedAt: '2023-06-20T10:59:46.383433891Z',
            errorMsg: '',
            reportRequestType: 'ON_DEMAND',
            reportNotificationMethod: 'DOWNLOAD',
        };
        const reportSnapshot: ReportSnapshot = {
            reportConfigId: 'report-config-id-1',
            reportJobId: 'report-job-id-1',
            name: 'test-name-1',
            description: 'test-description-1',
            vulnReportFilters: {
                fixability: 'FIXABLE',
                severities: ['LOW_VULNERABILITY_SEVERITY'],
                imageTypes: ['DEPLOYED'],
                allVuln: true,
            },
            collectionSnapshot: {
                id: 'test-collection-id-1',
                name: 'test-collection-name-1',
            },
            schedule: null,
            user: {
                id: 'test-user-1',
                name: 'test-user-name-1',
            },
            reportStatus,
            notifiers: [],
            isDownloadAvailable: true,
        };

        render(
            <MyLastReportJobStatus
                reportSnapshot={reportSnapshot}
                isLoadingReportSnapshots={false}
                currentUserId="test-user-1"
            />
        );

        const statusTextElement = screen.getByText('Ready for download');
        const statusIconElement = screen.getByTitle('Report download was successfully prepared');

        expect(statusTextElement).toBeInTheDocument();
        expect(statusIconElement).toBeInTheDocument();
    });

    test('should show "Error" when your last job status fails to generate a download', async () => {
        const reportStatus: ReportStatus = {
            runState: 'FAILURE',
            completedAt: '2023-06-20T10:59:46.383433891Z',
            errorMsg: 'Some error',
            reportRequestType: 'ON_DEMAND',
            reportNotificationMethod: 'DOWNLOAD',
        };
        const reportSnapshot: ReportSnapshot = {
            reportConfigId: 'report-config-id-1',
            reportJobId: 'report-job-id-1',
            name: 'test-name-1',
            description: 'test-description-1',
            vulnReportFilters: {
                fixability: 'FIXABLE',
                severities: ['LOW_VULNERABILITY_SEVERITY'],
                imageTypes: ['DEPLOYED'],
                allVuln: true,
            },
            collectionSnapshot: {
                id: 'test-collection-id-1',
                name: 'test-collection-name-1',
            },
            schedule: null,
            user: {
                id: 'test-user-1',
                name: 'test-user-name-1',
            },
            reportStatus,
            notifiers: [],
            isDownloadAvailable: true,
        };

        render(
            <MyLastReportJobStatus
                reportSnapshot={reportSnapshot}
                isLoadingReportSnapshots={false}
                currentUserId="test-user-1"
            />
        );

        const statusTextElement = screen.getByText('Error');
        const statusIconElement = screen.getByRole('img', { name: 'Report run was unsuccessful' });

        expect(statusTextElement).toBeInTheDocument();
        expect(statusIconElement).toBeInTheDocument();
    });
});
