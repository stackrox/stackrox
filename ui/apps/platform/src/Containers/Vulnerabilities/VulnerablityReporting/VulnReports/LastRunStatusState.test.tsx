import React, { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import { ReportStatus } from 'services/ReportsService.types';

import LastRunStatusState from './LastRunStatusState';

describe('LastRunStatusState', () => {
    test('should show the correct rendered output for a successful email', async () => {
        const reportStatus: ReportStatus = {
            runState: 'SUCCESS',
            completedAt: '2023-06-20T10:59:46.383433891Z',
            errorMsg: '',
            reportRequestType: 'ON_DEMAND',
            reportNotificationMethod: 'EMAIL',
        };

        // ARRANGE
        render(<LastRunStatusState reportStatus={reportStatus} />);

        // ASSERT
        expect(screen.getByTitle('Report run was successful')).toBeDefined();
        expect(screen.getByText('Emailed')).toBeDefined();
    });

    test('should show the correct rendered output for a successful download', async () => {
        const reportStatus: ReportStatus = {
            runState: 'SUCCESS',
            completedAt: '2023-06-20T10:59:46.383433891Z',
            errorMsg: '',
            reportRequestType: 'ON_DEMAND',
            reportNotificationMethod: 'DOWNLOAD',
        };

        // ARRANGE
        render(<LastRunStatusState reportStatus={reportStatus} />);

        // ASSERT
        expect(screen.getByTitle('Report run was successful')).toBeDefined();
        expect(screen.getByText('Download prepared')).toBeDefined();
    });

    test('should show the correct rendered output for a generic success', async () => {
        const reportStatus: ReportStatus = {
            runState: 'SUCCESS',
            completedAt: '2023-06-20T10:59:46.383433891Z',
            errorMsg: '',
            reportRequestType: 'ON_DEMAND',
            reportNotificationMethod: 'UNSET',
        };

        // ARRANGE
        render(<LastRunStatusState reportStatus={reportStatus} />);

        // ASSERT
        expect(screen.getByTitle('Report run was successful')).toBeDefined();
        expect(screen.getByText('Success')).toBeDefined();
    });

    test('should show the correct rendered output for an error when emailing', async () => {
        const reportStatus: ReportStatus = {
            runState: 'FAILURE',
            completedAt: '2023-06-20T10:59:46.383433891Z',
            errorMsg: '',
            reportRequestType: 'ON_DEMAND',
            reportNotificationMethod: 'EMAIL',
        };

        // ARRANGE
        render(<LastRunStatusState reportStatus={reportStatus} />);

        // ASSERT
        expect(screen.getByTitle('Report run was unsuccessful')).toBeDefined();
        expect(screen.getByText('Email attempted')).toBeDefined();
    });

    test('should show the correct rendered output for an error when preparing a download', async () => {
        const reportStatus: ReportStatus = {
            runState: 'FAILURE',
            completedAt: '2023-06-20T10:59:46.383433891Z',
            errorMsg: '',
            reportRequestType: 'ON_DEMAND',
            reportNotificationMethod: 'DOWNLOAD',
        };

        // ARRANGE
        render(<LastRunStatusState reportStatus={reportStatus} />);

        // ASSERT
        expect(screen.getByTitle('Report run was unsuccessful')).toBeDefined();
        expect(screen.getByText('Failed to generate download')).toBeDefined();
    });

    test('should show the correct rendered output for a generic error', async () => {
        const reportStatus: ReportStatus = {
            runState: 'FAILURE',
            completedAt: '2023-06-20T10:59:46.383433891Z',
            errorMsg: '',
            reportRequestType: 'ON_DEMAND',
            reportNotificationMethod: 'UNSET',
        };

        // ARRANGE
        render(<LastRunStatusState reportStatus={reportStatus} />);

        // ASSERT
        expect(screen.getByTitle('Report run was unsuccessful')).toBeDefined();
        expect(screen.getByText('Error')).toBeDefined();
    });

    test('should show the correct rendered output for waiting for a report', async () => {
        const reportStatus: ReportStatus = {
            runState: 'WAITING',
            completedAt: '2023-06-20T10:59:46.383433891Z',
            errorMsg: '',
            reportRequestType: 'ON_DEMAND',
            reportNotificationMethod: 'DOWNLOAD',
        };

        // ARRANGE
        render(<LastRunStatusState reportStatus={reportStatus} />);

        // ASSERT
        expect(screen.queryByTitle('Report run was successful')).toBeNull();
        expect(screen.queryByTitle('Report run was unsuccessful')).toBeNull();
        expect(screen.getByText('-')).toBeDefined();
    });

    test('should show the correct rendered output for preparing a report', async () => {
        const reportStatus: ReportStatus = {
            runState: 'PREPARING',
            completedAt: '2023-06-20T10:59:46.383433891Z',
            errorMsg: '',
            reportRequestType: 'ON_DEMAND',
            reportNotificationMethod: 'DOWNLOAD',
        };

        // ARRANGE
        render(<LastRunStatusState reportStatus={reportStatus} />);

        // ASSERT
        expect(screen.queryByTitle('Report run was successful')).toBeNull();
        expect(screen.queryByTitle('Report run was unsuccessful')).toBeNull();
        expect(screen.getByText('-')).toBeDefined();
    });
});
