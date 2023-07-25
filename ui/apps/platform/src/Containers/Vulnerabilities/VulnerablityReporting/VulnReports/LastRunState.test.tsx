import React, { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import { ReportStatus } from 'services/ReportsService.types';

import LastRunState from './LastRunState';

describe('LastRunState', () => {
    let originalTimeZone;

    beforeAll(() => {
        // Save original timezone and set new one for testing
        originalTimeZone = process.env.TZ;
        process.env.TZ = 'UTC';
    });

    afterAll(() => {
        // Restore original timezone
        process.env.TZ = originalTimeZone;
    });

    test('should show the time stamp of the last successful prepared download', async () => {
        const reportStatus: ReportStatus = {
            runState: 'SUCCESS',
            completedAt: '2023-06-20T10:59:46.383433891Z',
            errorMsg: '',
            reportRequestType: 'ON_DEMAND',
            reportNotificationMethod: 'DOWNLOAD',
        };

        // ARRANGE
        render(<LastRunState reportStatus={reportStatus} />);

        // ASSERT
        expect(screen.getByText('06/20/2023 | 10:59:46AM')).toBeDefined();
    });

    test('should show the time stamp of the last error when preparing a download', async () => {
        const reportStatus: ReportStatus = {
            runState: 'FAILURE',
            completedAt: '2023-06-20T10:59:46.383433891Z',
            errorMsg: '',
            reportRequestType: 'ON_DEMAND',
            reportNotificationMethod: 'DOWNLOAD',
        };

        // ARRANGE
        render(<LastRunState reportStatus={reportStatus} />);

        // ASSERT
        expect(screen.getByText('06/20/2023 | 10:59:46AM')).toBeDefined();
    });

    test('should show the time stamp of the last successful email', async () => {
        const reportStatus: ReportStatus = {
            runState: 'SUCCESS',
            completedAt: '2023-06-20T10:59:46.383433891Z',
            errorMsg: '',
            reportRequestType: 'ON_DEMAND',
            reportNotificationMethod: 'EMAIL',
        };

        // ARRANGE
        render(<LastRunState reportStatus={reportStatus} />);

        // ASSERT
        expect(screen.getByText('06/20/2023 | 10:59:46AM')).toBeDefined();
    });

    test('should show the time stamp of the last error attempting to email', async () => {
        const reportStatus: ReportStatus = {
            runState: 'FAILURE',
            completedAt: '2023-06-20T10:59:46.383433891Z',
            errorMsg: '',
            reportRequestType: 'ON_DEMAND',
            reportNotificationMethod: 'EMAIL',
        };

        // ARRANGE
        render(<LastRunState reportStatus={reportStatus} />);

        // ASSERT
        expect(screen.getByText('06/20/2023 | 10:59:46AM')).toBeDefined();
    });

    test('should show a spinner and the correct text when preparing a report', async () => {
        const reportStatus: ReportStatus = {
            runState: 'PREPARING',
            completedAt: '2023-06-20T10:59:46.383433891Z',
            errorMsg: '',
            reportRequestType: 'ON_DEMAND',
            reportNotificationMethod: 'EMAIL',
        };

        // ARRANGE
        render(<LastRunState reportStatus={reportStatus} />);

        // ASSERT
        expect(screen.getByLabelText('Preparing report')).toBeDefined();
        expect(screen.getByText('Preparing')).toBeDefined();
    });

    test("should show that a run didn't happen when waiting for a report", async () => {
        const reportStatus: ReportStatus = {
            runState: 'WAITING',
            completedAt: '2023-06-20T10:59:46.383433891Z',
            errorMsg: '',
            reportRequestType: 'ON_DEMAND',
            reportNotificationMethod: 'EMAIL',
        };

        // ARRANGE
        render(<LastRunState reportStatus={reportStatus} />);

        // ASSERT
        expect(screen.getByText('Never run')).toBeDefined();
    });
});
