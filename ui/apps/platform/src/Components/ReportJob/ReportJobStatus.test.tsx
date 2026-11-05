import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import ReportJobStatus from './ReportJobStatus';

/*
  The following list enumerates possible states that can be checked:
    * 'Preparing': This state indicates the system is in the process of running.
    * 'Waiting': The system is idle, waiting for necessary input or resources to proceed.
    * 'Failure': An error occurred, causing the run to fail. Check for issues and try again.
    * 'Download - Not actionable / No permissions': The download is available but cannot be accessed or manipulated due to insufficient permissions.
    * 'Download - Actionable': The full report download is available and can be accessed or manipulated by the user.
    * 'Download - Partial report actionable': A partial report download is available and can be accessed or manipulated by the user.
    * 'Download - Deleted': The download was previously available but has since been removed or is no longer accessible.
    * 'Email - Delivered': The full report has been successfully sent and delivered to the recipient.
    * 'Email - Partial report delivered': A partial report has been successfully sent and delivered to the recipient.
    * 'Unknown': The current state cannot be determined or does not match any of the predefined states listed above. Typescript should safegaurd against this. This is mostly in case the API returns a different value
*/
describe('ReportJobStatus', () => {
    test('should display the status text and icon when preparing', () => {
        render(
            <ReportJobStatus
                reportStatus={{
                    runState: 'PREPARING',
                    reportRequestType: 'ON_DEMAND',
                    reportNotificationMethod: 'DOWNLOAD',
                    completedAt: '',
                    errorMsg: '',
                }}
                isDownloadAvailable={false}
                areDownloadActionsDisabled={false}
                onDownload={() => {}}
            />
        );

        const statusTextElement = screen.getByText('Preparing');

        expect(statusTextElement).toBeInTheDocument();
    });

    test('should display the status text and icon when waiting', () => {
        render(
            <ReportJobStatus
                reportStatus={{
                    runState: 'WAITING',
                    reportRequestType: 'ON_DEMAND',
                    reportNotificationMethod: 'DOWNLOAD',
                    completedAt: '',
                    errorMsg: '',
                }}
                isDownloadAvailable={false}
                areDownloadActionsDisabled={false}
                onDownload={() => {}}
            />
        );

        const statusTextElement = screen.getByText('Waiting');

        expect(statusTextElement).toBeInTheDocument();
    });

    test('should display the status text and icon when failed', async () => {
        render(
            <ReportJobStatus
                reportStatus={{
                    runState: 'FAILURE',
                    reportRequestType: 'ON_DEMAND',
                    reportNotificationMethod: 'DOWNLOAD',
                    completedAt: '',
                    errorMsg: 'scan failed on Cluster "Prod-East"',
                }}
                isDownloadAvailable={false}
                areDownloadActionsDisabled={false}
                onDownload={() => {}}
            />
        );

        const statusTextElement = screen.getByText('Report failed to generate');

        expect(statusTextElement).toBeInTheDocument();
    });

    test('should display the status text and icon when download is available but not accessible', async () => {
        render(
            <ReportJobStatus
                reportStatus={{
                    runState: 'GENERATED',
                    reportRequestType: 'ON_DEMAND',
                    reportNotificationMethod: 'DOWNLOAD',
                    completedAt: '',
                    errorMsg: '',
                }}
                isDownloadAvailable
                areDownloadActionsDisabled
                onDownload={() => {}}
            />
        );

        const statusTextElement = screen.getByText('Report ready for download');

        expect(statusTextElement).toBeInTheDocument();
    });

    test('should display the status text and icon when download is available and accessible', () => {
        render(
            <ReportJobStatus
                reportStatus={{
                    runState: 'GENERATED',
                    reportRequestType: 'ON_DEMAND',
                    reportNotificationMethod: 'DOWNLOAD',
                    completedAt: '',
                    errorMsg: '',
                }}
                isDownloadAvailable
                areDownloadActionsDisabled={false}
                onDownload={() => {}}
            />
        );

        const statusTextElement = screen.getByText('Report ready for download');

        expect(statusTextElement).toBeInTheDocument();
    });

    test('should display the status text and icon when a partial report download is available and accessible', () => {
        render(
            <ReportJobStatus
                reportStatus={{
                    runState: 'PARTIAL_SCAN_ERROR_DOWNLOAD',
                    reportRequestType: 'ON_DEMAND',
                    reportNotificationMethod: 'DOWNLOAD',
                    completedAt: '',
                    errorMsg: '',
                }}
                isDownloadAvailable
                areDownloadActionsDisabled={false}
                onDownload={() => {}}
            />
        );

        const statusTextElement = screen.getByText('Partial report ready for download');

        expect(statusTextElement).toBeInTheDocument();
    });

    test('should display the status text and icon when download is deleted', async () => {
        render(
            <ReportJobStatus
                reportStatus={{
                    runState: 'GENERATED',
                    reportRequestType: 'ON_DEMAND',
                    reportNotificationMethod: 'DOWNLOAD',
                    completedAt: '',
                    errorMsg: '',
                }}
                isDownloadAvailable={false}
                areDownloadActionsDisabled={false}
                onDownload={() => {}}
            />
        );

        const statusTextElement = screen.getByText('Report download deleted');

        expect(statusTextElement).toBeInTheDocument();
    });

    test('should display the status text and icon when email is sent', () => {
        render(
            <ReportJobStatus
                reportStatus={{
                    runState: 'DELIVERED',
                    reportRequestType: 'ON_DEMAND',
                    reportNotificationMethod: 'EMAIL',
                    completedAt: '',
                    errorMsg: '',
                }}
                isDownloadAvailable
                areDownloadActionsDisabled={false}
                onDownload={() => {}}
            />
        );

        const statusTextElement = screen.getByText('Report successfully sent');

        expect(statusTextElement).toBeInTheDocument();
    });

    test('should display the status text and icon when a partial report is sent via email', () => {
        render(
            <ReportJobStatus
                reportStatus={{
                    runState: 'PARTIAL_SCAN_ERROR_EMAIL',
                    reportRequestType: 'ON_DEMAND',
                    reportNotificationMethod: 'EMAIL',
                    completedAt: '',
                    errorMsg: '',
                }}
                isDownloadAvailable={false}
                areDownloadActionsDisabled={false}
                onDownload={() => {}}
            />
        );

        const statusTextElement = screen.getByText('Partial report successfully sent');

        expect(statusTextElement).toBeInTheDocument();
    });
});
