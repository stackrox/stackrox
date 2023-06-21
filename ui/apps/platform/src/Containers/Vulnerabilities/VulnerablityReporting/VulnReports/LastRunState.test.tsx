import React, { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import { ReportStatus } from 'types/report.proto';

import LastRunState from './LastRunState';

describe('LastRunState', () => {
    test('should show the time stamp of the last successful prepared download', async () => {
        const reportStatus: ReportStatus = {
            runState: 'SUCCESS',
            runTime: '2023-06-20T10:59:46.383433891Z',
            errorMsg: '',
            reportMethod: 'ON_DEMAND',
            reportNotificationMethod: 'DOWNLOAD',
        };

        // ARRANGE
        render(<LastRunState reportStatus={reportStatus} />);

        // ASSERT
        expect(screen.getByText('06/20/2023 | 3:59:46AM')).toBeDefined();
    });

    test('should show the time stamp of the last error when preparing a download', async () => {
        const reportStatus: ReportStatus = {
            runState: 'FAILURE',
            runTime: '2023-06-20T10:59:46.383433891Z',
            errorMsg: '',
            reportMethod: 'ON_DEMAND',
            reportNotificationMethod: 'DOWNLOAD',
        };

        // ARRANGE
        render(<LastRunState reportStatus={reportStatus} />);

        // ASSERT
        expect(screen.getByText('06/20/2023 | 3:59:46AM')).toBeDefined();
    });

    test('should show the time stamp of the last successful email', async () => {
        const reportStatus: ReportStatus = {
            runState: 'SUCCESS',
            runTime: '2023-06-20T10:59:46.383433891Z',
            errorMsg: '',
            reportMethod: 'ON_DEMAND',
            reportNotificationMethod: 'EMAIL',
        };

        // ARRANGE
        render(<LastRunState reportStatus={reportStatus} />);

        // ASSERT
        expect(screen.getByText('06/20/2023 | 3:59:46AM')).toBeDefined();
    });

    test('should show the time stamp of the last error attempting to email', async () => {
        const reportStatus: ReportStatus = {
            runState: 'FAILURE',
            runTime: '2023-06-20T10:59:46.383433891Z',
            errorMsg: '',
            reportMethod: 'ON_DEMAND',
            reportNotificationMethod: 'EMAIL',
        };

        // ARRANGE
        render(<LastRunState reportStatus={reportStatus} />);

        // ASSERT
        expect(screen.getByText('06/20/2023 | 3:59:46AM')).toBeDefined();
    });

    test('should show a spinner and the correct text when preparing a report', async () => {
        const reportStatus: ReportStatus = {
            runState: 'PREPARING',
            runTime: '2023-06-20T10:59:46.383433891Z',
            errorMsg: '',
            reportMethod: 'ON_DEMAND',
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
            runTime: '2023-06-20T10:59:46.383433891Z',
            errorMsg: '',
            reportMethod: 'ON_DEMAND',
            reportNotificationMethod: 'EMAIL',
        };

        // ARRANGE
        render(<LastRunState reportStatus={reportStatus} />);

        // ASSERT
        expect(screen.getByText('Never run')).toBeDefined();
    });
});
