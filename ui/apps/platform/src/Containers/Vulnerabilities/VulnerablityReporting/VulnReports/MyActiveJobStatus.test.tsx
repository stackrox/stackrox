import React, { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import { ReportStatus } from 'services/ReportsService.types';

import MyActiveJobStatus from './MyActiveJobStatus';

describe('MyActiveJobStatus', () => {
    test('should show "PREPARING" when your active job status is preparing', async () => {
        const reportStatus: ReportStatus = {
            runState: 'PREPARING',
            completedAt: '',
            errorMsg: '',
            reportRequestType: 'ON_DEMAND',
            reportNotificationMethod: 'DOWNLOAD',
        };

        render(<MyActiveJobStatus reportStatus={reportStatus} />);

        expect(screen.getByText('Preparing')).toBeInTheDocument();
    });

    test('should show "WAITING" when your active job status is waiting', async () => {
        const reportStatus: ReportStatus = {
            runState: 'WAITING',
            completedAt: '',
            errorMsg: '',
            reportRequestType: 'ON_DEMAND',
            reportNotificationMethod: 'DOWNLOAD',
        };

        render(<MyActiveJobStatus reportStatus={reportStatus} />);

        expect(screen.getByText('Waiting')).toBeInTheDocument();
    });

    test('should show "-" when your active job status is a success', async () => {
        const reportStatus: ReportStatus = {
            runState: 'GENERATED',
            completedAt: '2023-06-20T10:59:46.383433891Z',
            errorMsg: '',
            reportRequestType: 'ON_DEMAND',
            reportNotificationMethod: 'DOWNLOAD',
        };

        render(<MyActiveJobStatus reportStatus={reportStatus} />);

        expect(screen.getByText('-')).toBeInTheDocument();
    });

    test('should show "-" when your active job status is a failure', async () => {
        const reportStatus: ReportStatus = {
            runState: 'FAILURE',
            completedAt: '2023-06-20T10:59:46.383433891Z',
            errorMsg: 'Some error',
            reportRequestType: 'ON_DEMAND',
            reportNotificationMethod: 'DOWNLOAD',
        };

        render(<MyActiveJobStatus reportStatus={reportStatus} />);

        expect(screen.getByText('-')).toBeInTheDocument();
    });
});
